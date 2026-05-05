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
	run := searchRun{Mode: "text"}

	embedClient, providerInfo, err := embeddings.ResolveEmbedder(embedConfig)
	if err != nil {
		results, textErr := store.SearchText(query)
		if textErr != nil {
			return searchRun{}, textErr
		}
		run.Results = limitResults(results, limit)
		return run, nil
	}

	requiresReindex, err := store.RequiresReindex(providerInfo.Provider, providerInfo.Model, providerInfo.Dimensions)
	if err == nil && requiresReindex {
		results, textErr := store.SearchText(query)
		if textErr != nil {
			return searchRun{}, textErr
		}
		run.Warning = "index metadata does not match current embedding provider; semantic results may be stale until reindex"
		run.Mode = "text_stale_index"
		run.Results = limitResults(results, limit)
		return run, nil
	}

	queryVector, err := embedClient.GetEmbedding(query)
	if err == nil {
		results, semanticErr := store.SearchSemantic(queryVector, hybridCandidateLimit, false)
		if semanticErr != nil {
			return searchRun{}, semanticErr
		}
		if len(results) > 0 {
			run.Mode = "semantic_hybrid"
			run.Results = rerankHybridResults(query, results, limit)
			run.FileResults = storage.CollapseFileResults(run.Results, 5)
			return run, nil
		}
	}

	results, textErr := store.SearchText(query)
	if textErr != nil {
		return searchRun{}, textErr
	}
	run.Results = limitResults(results, limit)
	return run, nil
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
