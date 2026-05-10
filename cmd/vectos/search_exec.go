package main

import (
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
	embedClient, providerInfo, err := embeddings.ResolveEmbedder(embedConfig)
	if err != nil {
		return textSearchFallback(store, query, limit, "text", "")
	}

	requiresReindex, err := store.RequiresReindex(providerInfo.Provider, providerInfo.Model, providerInfo.Dimensions)
	if err == nil && requiresReindex {
		return textSearchFallback(store, query, limit, "text_stale_index",
			"index metadata does not match current embedding provider; semantic results may be stale until reindex")
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

// trySemanticSearch attempts a semantic/embedding-based search. Returns
// (result, true) on success, or (empty, false) with no error when a
// text-only fallback should be used.
func trySemanticSearch(embedClient embeddings.Embedder, store *storage.SQLiteStorage, query string, limit int) (searchRun, bool, error) {
	queryVector, err := embedClient.GetEmbedding(query)
	if err != nil {
		return searchRun{}, false, nil
	}

	results, err := store.SearchSemantic(queryVector, hybridCandidateLimit, false)
	if err != nil {
		return searchRun{}, false, err
	}
	if len(results) == 0 {
		return searchRun{}, false, nil
	}

	reranked := rerankHybridResults(query, results, limit)
	return searchRun{
		Mode:        "semantic_hybrid",
		Results:     reranked,
		FileResults: storage.CollapseFileResults(reranked, 5),
	}, true, nil
}

// textSearchFallback performs a plain text search and wraps the result
// in a searchRun with the given mode and optional warning message.
func textSearchFallback(store *storage.SQLiteStorage, query string, limit int, mode, warning string) (searchRun, error) {
	results, err := store.SearchText(query)
	if err != nil {
		return searchRun{}, err
	}
	return searchRun{
		Mode:    mode,
		Warning: warning,
		Results: limitResults(results, limit),
	}, nil
}

func executeSearchDocs(store *storage.SQLiteStorage, embedConfig config.EmbeddingConfig, query string, limit int) (searchRun, error) {
	run := searchRun{Mode: "text"}

	embedClient, _, err := embeddings.ResolveEmbedder(embedConfig)
	if err != nil {
		results, textErr := store.SearchText(query)
		if textErr != nil {
			return searchRun{}, textErr
		}
		run.Results = limitResults(results, limit)
		run.FileResults = storage.CollapseFileResults(run.Results, 5)
		return run, nil
	}

	queryVector, err := embedClient.GetEmbedding(query)
	if err == nil {
		results, semanticErr := store.SearchSemantic(queryVector, limit, true)
		if semanticErr != nil {
			return searchRun{}, semanticErr
		}
		if len(results) > 0 {
			run.Mode = "semantic_hybrid"
			run.Results = results
			run.FileResults = storage.CollapseFileResults(results, 5)
			return run, nil
		}
	}

	results, textErr := store.SearchText(query)
	if textErr != nil {
		return searchRun{}, textErr
	}
	run.Results = limitResults(results, limit)
	run.FileResults = storage.CollapseFileResults(run.Results, 5)
	return run, nil
}
