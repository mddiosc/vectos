package main

import (
	"fmt"
	"vectos/internal/config"
	"vectos/internal/embeddings"
	"vectos/internal/storage"
)

type searchRun struct {
	Results     []storage.CodeChunk
	FileResults []storage.SearchFileResult
	Mode        string
	Warning     string
}

func executeSearch(store *storage.SQLiteStorage, embedConfig config.EmbeddingConfig, query string, limit int) (searchRun, error) {
	// Ensure the store can auto-rebuild its vector index if it becomes stale.
	store.SetVectorIndexParams(embedConfig.VectorIndex.HNSW_M, embedConfig.VectorIndex.HNSW_EfConstruction, embedConfig.VectorIndex.HNSW_EfSearch)

	embedClient, providerInfo, err := embeddings.ResolveEmbedder(embedConfig)
	if err != nil {
		return textSearchFallback(store, query, limit, "text", "")
	}

	requiresReindex, err := store.RequiresReindex(providerInfo.Provider, providerInfo.Model, providerInfo.Dimensions, currentIndexFingerprint(indexChunkerConfig(embedConfig.Embedded.BatchSize)))
	if err == nil && requiresReindex {
		warning := fmt.Sprintf("index uses different embedding model; reindex required for accurate semantic results (current: %s/%dd)", providerInfo.Model, providerInfo.Dimensions)
		if stored, metaErr := store.GetIndexMetadata(); metaErr == nil {
			warning = fmt.Sprintf("index was built with %s/%dd but current provider is %s/%dd; reindex required for accurate results", stored.Model, stored.Dimensions, providerInfo.Model, providerInfo.Dimensions)
		}
		return textSearchFallback(store, query, limit, "text_stale_index", warning)
	}

	run, ok, err := trySemanticSearch(embedClient, store, query, limit)
	if err != nil {
		return searchRun{}, err
	}
	if ok {
		return run, nil
	}

	return textSearchFallback(store, query, limit, "text", "")
}

// trySemanticSearch attempts a semantic/embedding-based search with RRF fusion.
// Runs both vector and keyword searches independently, then fuses with RRF.
// Returns (result, true) on success, or (empty, false) with no error when a
// text-only fallback should be used.
func trySemanticSearch(embedClient embeddings.Embedder, store *storage.SQLiteStorage, query string, limit int) (searchRun, bool, error) {
	queryVector, err := embedClient.GetEmbedding(query)
	if err != nil {
		return searchRun{}, false, nil
	}

	// Run vector and keyword searches independently
	vectorResults, err := store.SearchSemantic(queryVector, rrfVectorLimit, false)
	if err != nil {
		return searchRun{}, false, err
	}

	keywordResults, err := store.SearchTextRanked(query, rrfKeywordLimit)
	if err != nil {
		// Keyword search failed; proceed with vector-only
		keywordResults = nil
	}

	if len(vectorResults) == 0 && len(keywordResults) == 0 {
		return searchRun{}, false, nil
	}

	// Fuse with RRF and apply structural penalties
	fused := fuseResults(vectorResults, keywordResults, rrfConstant, 0)
	fused = applyFusionPenalties(fused)

	// Dedup by file (max 2 per file), return top results
	deduped := dedupeByFile(fused, limit, rrfResultLimitPerFile)
	return searchRun{
		Mode:        "semantic_hybrid",
		Results:     deduped,
		FileResults: storage.CollapseFileResults(deduped, 5),
	}, true, nil
}

// textSearchFallback performs a keyword text search and wraps the result
// in a searchRun with the given mode and optional warning message.
func textSearchFallback(store *storage.SQLiteStorage, query string, limit int, mode, warning string) (searchRun, error) {
	results, err := store.SearchTextRanked(query, limit)
	if err != nil {
		// Fall back to plain LIKE search if FTS5 fails
		results, err = store.SearchText(query)
		if err != nil {
			return searchRun{}, err
		}
		return searchRun{
			Mode:    mode,
			Warning: warning,
			Results: limitResults(results, limit),
		}, nil
	}
	return searchRun{
		Mode:    mode,
		Warning: warning,
		Results: results,
	}, nil
}

func executeSearchDocs(store *storage.SQLiteStorage, embedConfig config.EmbeddingConfig, query string, limit int) (searchRun, error) {
	store.SetVectorIndexParams(embedConfig.VectorIndex.HNSW_M, embedConfig.VectorIndex.HNSW_EfConstruction, embedConfig.VectorIndex.HNSW_EfSearch)
	run := searchRun{Mode: "text"}

	embedClient, _, err := embeddings.ResolveEmbedder(embedConfig)
	if err != nil {
		results, textErr := store.SearchTextRanked(query, limit)
		if textErr != nil {
			results, textErr = store.SearchText(query)
			if textErr != nil {
				return searchRun{}, textErr
			}
		}
		run.Results = limitResults(results, limit)
		run.FileResults = storage.CollapseFileResults(run.Results, 5)
		return run, nil
	}

	queryVector, err := embedClient.GetEmbedding(query)
	if err == nil {
		vectorResults, _ := store.SearchSemantic(queryVector, rrfVectorLimit, true)
		keywordResults, _ := store.SearchTextRanked(query, rrfKeywordLimit)
		if len(vectorResults) > 0 || len(keywordResults) > 0 {
			fused := fuseResults(vectorResults, keywordResults, rrfConstant, 0)
			fused = applyFusionPenalties(fused)
			deduped := dedupeByFile(fused, limit, rrfResultLimitPerFile)
			run.Mode = "semantic_hybrid"
			run.Results = deduped
			run.FileResults = storage.CollapseFileResults(deduped, 5)
			return run, nil
		}
	}

	results, textErr := store.SearchTextRanked(query, limit)
	if textErr != nil {
		results, textErr = store.SearchText(query)
		if textErr != nil {
			return searchRun{}, textErr
		}
	}
	run.Results = limitResults(results, limit)
	run.FileResults = storage.CollapseFileResults(run.Results, 5)
	return run, nil
}
