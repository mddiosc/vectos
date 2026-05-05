package main

import (
	"context"
	"fmt"

	mcpSDK "github.com/modelcontextprotocol/go-sdk/mcp"

	"vectos/internal/config"
	"vectos/internal/embeddings"
	"vectos/internal/indexer"
	"vectos/internal/storage"
	"vectos/internal/workspace"
)

func docsIndexHasChunks(pm *storage.ProjectManager, scope *workspace.Scope) (bool, error) {
	store, err := openStorageForScope(pm, scope, true)
	if err != nil {
		return false, err
	}
	defer store.Close()

	stats, err := store.Stats()
	if err != nil {
		return false, err
	}

	return stats.ChunkCount > 0, nil
}

func makeSearchCodeHandler(projectBaseDir string, embedConfig config.EmbeddingConfig) func(context.Context, *mcpSDK.CallToolRequest, searchCodeInput) (*mcpSDK.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcpSDK.CallToolRequest, input searchCodeInput) (*mcpSDK.CallToolResult, any, error) {
		scope, err := resolveToolScope(input.Path, input.Project)
		if err != nil {
			return nil, nil, err
		}

		pm, err := storage.NewProjectManager(projectBaseDir)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to initialize project manager: %w", err)
		}

		store, err := openStorageForScope(pm, scope, false)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to open storage: %w", err)
		}
		defer store.Close()

		stats, err := store.Stats()
		if err != nil {
			return nil, nil, fmt.Errorf("failed to inspect index state: %w", err)
		}
		if stats.ChunkCount == 0 {
			text, err := stringifyMCPResult(buildMCPMissingIndexPayload(scope))
			if err != nil {
				return nil, nil, err
			}
			return &mcpSDK.CallToolResult{Content: []mcpSDK.Content{&mcpSDK.TextContent{Text: text}}}, nil, nil
		}

		searchRun, err := executeSearch(store, embedConfig, input.Query, 5)
		if err != nil {
			return nil, nil, err
		}

		payload := buildMCPSearchPayload(scope, input.Query, searchRun)
		if len(payload.Results) == 0 {
			hasDocs, docsErr := docsIndexHasChunks(pm, scope)
			if docsErr == nil && hasDocs {
				payload.Guidance = "TRY_DOCS"
				payload.NextAction = "Try search_docs tool instead, or run index_project with docs: true to index documentation."
			}
		}

		text, err := stringifyMCPResult(payload)
		if err != nil {
			return nil, nil, err
		}

		return &mcpSDK.CallToolResult{Content: []mcpSDK.Content{&mcpSDK.TextContent{Text: text}}}, nil, nil
	}
}

func makeSearchDocsHandler(projectBaseDir string, embedConfig config.EmbeddingConfig) func(context.Context, *mcpSDK.CallToolRequest, searchDocsInput) (*mcpSDK.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcpSDK.CallToolRequest, input searchDocsInput) (*mcpSDK.CallToolResult, any, error) {
		scope, err := resolveToolScope(input.Path, input.Project)
		if err != nil {
			return nil, nil, err
		}

		pm, err := storage.NewProjectManager(projectBaseDir)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to initialize project manager: %w", err)
		}

		store, err := openStorageForScope(pm, scope, true)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to open docs storage: %w", err)
		}
		defer store.Close()

		stats, err := store.Stats()
		if err != nil {
			return nil, nil, fmt.Errorf("failed to inspect docs index state: %w", err)
		}
		if stats.ChunkCount == 0 {
			payload := buildMCPSearchPayload(scope, input.Query, searchRun{Results: []storage.CodeChunk{}})
			payload.Guidance = "IDX_DOCS_MISSING"
			payload.NextAction = "Use index_project with docs: true to index documentation files first."
			text, err := stringifyMCPResult(payload)
			if err != nil {
				return nil, nil, err
			}
			return &mcpSDK.CallToolResult{Content: []mcpSDK.Content{&mcpSDK.TextContent{Text: text}}}, nil, nil
		}

		searchRun, err := executeSearchDocs(store, embedConfig, input.Query, 5)
		if err != nil {
			return nil, nil, err
		}

		payload := buildMCPSearchPayload(scope, input.Query, searchRun)
		text, err := stringifyMCPResult(payload)
		if err != nil {
			return nil, nil, err
		}

		return &mcpSDK.CallToolResult{Content: []mcpSDK.Content{&mcpSDK.TextContent{Text: text}}}, nil, nil
	}
}

func makeIndexProjectHandler(projectBaseDir string, embedConfig config.EmbeddingConfig) func(context.Context, *mcpSDK.CallToolRequest, indexProjectInput) (*mcpSDK.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcpSDK.CallToolRequest, input indexProjectInput) (*mcpSDK.CallToolResult, any, error) {
		scope, err := workspace.ResolveScope(input.Path, input.Project)
		if err != nil {
			return nil, nil, err
		}

		pm, err := storage.NewProjectManager(projectBaseDir)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to initialize project manager: %w", err)
		}

		embedClient, providerInfo, err := embeddings.ResolveEmbedder(embedConfig)
		if err != nil {
			return nil, nil, err
		}
		var store *storage.SQLiteStorage
		if input.Docs {
			store, err = storage.NewSQLiteStorageForDocsProjectName(pm, scope.Name)
		} else {
			store, err = storage.NewSQLiteStorageForProjectName(pm, scope.Name)
		}
		if err != nil {
			return nil, nil, fmt.Errorf("failed to open storage: %w", err)
		}
		defer store.Close()

		if err := store.SetIndexMetadata(storage.IndexMetadata{
			Provider:   providerInfo.Provider,
			Model:      providerInfo.Model,
			Dimensions: providerInfo.Dimensions,
		}); err != nil {
			return nil, nil, err
		}

		chunker := indexer.NewSimpleChunker(indexer.ChunkConfig{MaxLines: 10}, embedClient)
		paths, skippedPaths, err := collectIndexablePaths(scope.Roots, input.Docs)
		if err != nil {
			return nil, nil, err
		}

		changedPaths := parseChangedPaths(input.Changed)
		if len(changedPaths) > 0 {
			paths, skippedPaths, err = filterChangedPaths(scope, paths, skippedPaths, changedPaths)
			if err != nil {
				return nil, nil, err
			}
		}
		if err := prepareStoreForIndexing(store, changedPaths); err != nil {
			return nil, nil, err
		}

		indexedFiles := 0
		count := 0
		for _, path := range paths {
			language, err := detectLanguage(path)
			if err != nil {
				return nil, nil, err
			}

			chunks, err := chunker.ChunkFile(path, language)
			if err != nil {
				return nil, nil, err
			}

			if err := store.DeleteChunksByPath(path); err != nil {
				return nil, nil, err
			}

			for _, c := range chunks {
				_, err := store.SaveChunk(storage.CodeChunk{
					FilePath:  path,
					Content:   c.Content,
					StartLine: c.StartLine,
					EndLine:   c.EndLine,
					Language:  language,
					Category:  classifyCategory(language),
					Vector:    c.Vector,
					Signature: c.Signature,
					Purpose:   c.Purpose,
				})
				if err != nil {
					return nil, nil, err
				}
				count++
			}
			indexedFiles++
		}

		for _, skippedPath := range skippedPaths {
			if err := store.DeleteChunksByPath(skippedPath); err != nil {
				return nil, nil, err
			}
		}

		payload := buildMCPIndexPayload(scope, changedPaths, indexedFiles, count, len(skippedPaths))
		text, err := stringifyMCPResult(payload)
		if err != nil {
			return nil, nil, err
		}

		return &mcpSDK.CallToolResult{Content: []mcpSDK.Content{&mcpSDK.TextContent{Text: text}}}, nil, nil
	}
}
