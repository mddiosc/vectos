package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	mcpSDK "github.com/modelcontextprotocol/go-sdk/mcp"

	"vectos/internal/config"
	"vectos/internal/embeddings"
	"vectos/internal/indexer"
	"vectos/internal/storage"
	"vectos/internal/workspace"
)

// --- search_code ---

func makeSearchCodeHandler(projectBaseDir string, embedConfig config.EmbeddingConfig) func(context.Context, *mcpSDK.CallToolRequest, searchCodeInput) (*mcpSDK.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcpSDK.CallToolRequest, input searchCodeInput) (*mcpSDK.CallToolResult, any, error) {
		return runSearchCode(projectBaseDir, embedConfig, input)
	}
}

func runSearchCode(projectBaseDir string, embedConfig config.EmbeddingConfig, input searchCodeInput) (*mcpSDK.CallToolResult, any, error) {
	scope, pm, err := resolveScopeAndProjectManager(projectBaseDir, input)
	if err != nil {
		return nil, nil, err
	}

	store, err := openStorageForScope(pm, scope, false)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open storage: %w", err)
	}
	defer store.Close()

	if empty, err := isCodeIndexEmpty(store, scope); err != nil {
		return nil, nil, err
	} else if empty {
		return mcpTextResult(buildMCPMissingIndexPayload(scope))
	}

	run, err := executeSearch(store, embedConfig, input.Query, 5)
	if err != nil {
		return nil, nil, err
	}

	payload := buildMCPSearchPayload(scope, input.Query, run)
	if len(payload.Results) == 0 {
		annotateWithDocsGuidance(pm, scope, &payload)
	}

	return mcpTextResult(payload)
}

// resolveScopeAndProjectManager resolves the tool scope and creates a project
// manager for the given project base directory.
func resolveScopeAndProjectManager(projectBaseDir string, input searchCodeInput) (*workspace.Scope, *storage.ProjectManager, error) {
	scope, err := resolveToolScope(input.Path, input.Project)
	if err != nil {
		return nil, nil, err
	}
	pm, err := newProjectManager(projectBaseDir)
	if err != nil {
		return nil, nil, err
	}
	return scope, pm, nil
}

// isCodeIndexEmpty checks whether the code index has any chunks.
// Returns (true, nil) when the index is empty, (false, nil) when chunks
// exist, or (false, err) on stats error.
func isCodeIndexEmpty(store *storage.SQLiteStorage, scope *workspace.Scope) (bool, error) {
	stats, err := store.Stats()
	if err != nil {
		return false, fmt.Errorf("failed to inspect index state: %w", err)
	}
	return stats.ChunkCount == 0, nil
}

func annotateWithDocsGuidance(pm *storage.ProjectManager, scope *workspace.Scope, payload *mcpSearchPayload) {
	hasDocs, err := docsIndexHasChunks(pm, scope)
	if err != nil || !hasDocs {
		return
	}
	payload.Guidance = "TRY_DOCS"
	payload.NextAction = "Try search_docs tool instead, or run index_project with docs: true to index documentation."
}

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

// --- search_docs ---

func makeSearchDocsHandler(projectBaseDir string, embedConfig config.EmbeddingConfig) func(context.Context, *mcpSDK.CallToolRequest, searchDocsInput) (*mcpSDK.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcpSDK.CallToolRequest, input searchDocsInput) (*mcpSDK.CallToolResult, any, error) {
		return runSearchDocs(projectBaseDir, embedConfig, input)
	}
}

func runSearchDocs(projectBaseDir string, embedConfig config.EmbeddingConfig, input searchDocsInput) (*mcpSDK.CallToolResult, any, error) {
	scope, err := resolveToolScope(input.Path, input.Project)
	if err != nil {
		return nil, nil, err
	}

	pm, err := newProjectManager(projectBaseDir)
	if err != nil {
		return nil, nil, err
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
		return mcpDocsIndexMissingResult(scope, input.Query)
	}

	run, err := executeSearchDocs(store, embedConfig, input.Query, 5)
	if err != nil {
		return nil, nil, err
	}

	return mcpTextResult(buildMCPSearchPayload(scope, input.Query, run))
}

func mcpDocsIndexMissingResult(scope *workspace.Scope, query string) (*mcpSDK.CallToolResult, any, error) {
	payload := buildMCPSearchPayload(scope, query, searchRun{Results: []storage.CodeChunk{}})
	payload.Guidance = "IDX_DOCS_MISSING"
	payload.NextAction = "Use index_project with docs: true to index documentation files first."
	return mcpTextResult(payload)
}

// --- index_project ---

func makeIndexProjectHandler(projectBaseDir string, embedConfig config.EmbeddingConfig) func(context.Context, *mcpSDK.CallToolRequest, indexProjectInput) (*mcpSDK.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcpSDK.CallToolRequest, input indexProjectInput) (*mcpSDK.CallToolResult, any, error) {
		return runIndexProject(projectBaseDir, embedConfig, input)
	}
}

func runIndexProject(projectBaseDir string, embedConfig config.EmbeddingConfig, input indexProjectInput) (*mcpSDK.CallToolResult, any, error) {
	scope, store, embedClient, paths, skippedPaths, changedPaths, err := setupIndexRequest(projectBaseDir, embedConfig, input)
	if err != nil {
		return nil, nil, err
	}
	defer store.Close()

	if err := prepareStoreForIndexing(store, changedPaths); err != nil {
		return nil, nil, err
	}

	indexedFiles, count, err := indexPaths(store, embedClient, paths, embedConfig.Embedded.BatchSize)
	if err != nil {
		return nil, nil, err
	}

	if err := deleteSkippedChunks(store, skippedPaths); err != nil {
		return nil, nil, err
	}

	return mcpTextResult(buildMCPIndexPayload(scope, changedPaths, indexedFiles, count, len(skippedPaths)))
}

func setupIndexRequest(projectBaseDir string, embedConfig config.EmbeddingConfig, input indexProjectInput) (
	scope workspace.Scope,
	store *storage.SQLiteStorage,
	embedClient embeddings.Embedder,
	paths []string,
	skippedPaths []string,
	changedPaths []string,
	err error,
) {
	scope, err = workspace.ResolveScope(input.Path, input.Project)
	if err != nil {
		return
	}

	var pm *storage.ProjectManager
	pm, err = newProjectManager(projectBaseDir)
	if err != nil {
		return
	}

	store, err = openIndexStore(pm, scope.Name, input.Docs)
	if err != nil {
		return
	}
	defer func() {
		if err != nil {
			if closeErr := store.Close(); closeErr != nil {
				err = fmt.Errorf("%w; additionally failed to close storage: %v", err, closeErr)
			}
		}
	}()

	var providerInfo embeddings.ProviderInfo
	embedClient, providerInfo, err = embeddings.ResolveEmbedder(embedConfig)
	if err != nil {
		return
	}

	if err = store.SetIndexMetadata(storage.IndexMetadata{
		Provider:   providerInfo.Provider,
		Model:      providerInfo.Model,
		Dimensions: providerInfo.Dimensions,
	}); err != nil {
		return
	}

	paths, skippedPaths, err = resolveIndexPaths(scope, input)
	if err != nil {
		return
	}

	changedPaths = parseChangedPaths(input.Changed)
	return
}

func openIndexStore(pm *storage.ProjectManager, projectName string, docs bool) (*storage.SQLiteStorage, error) {
	if docs {
		return storage.NewSQLiteStorageForDocsProjectName(pm, projectName)
	}
	store, err := storage.NewSQLiteStorageForProjectName(pm, projectName)
	if err != nil {
		return nil, fmt.Errorf("failed to open storage: %w", err)
	}
	return store, nil
}

func resolveIndexPaths(scope workspace.Scope, input indexProjectInput) (paths, skippedPaths []string, err error) {
	changedPaths := parseChangedPaths(input.Changed)
	paths, skippedPaths, err = collectIndexablePaths(scope.Roots, input.Docs)
	if err != nil {
		return nil, nil, err
	}
	if len(changedPaths) > 0 {
		paths, skippedPaths, err = filterChangedPaths(scope, paths, skippedPaths, changedPaths)
		if err != nil {
			return nil, nil, err
		}
	}
	return paths, skippedPaths, nil
}

func indexPaths(store *storage.SQLiteStorage, embedClient embeddings.Embedder, paths []string, batchSize int) (indexedFiles, count int, err error) {
	chunker := indexer.NewSimpleChunker(indexer.ChunkConfig{MaxLines: 10, BatchSize: batchSize}, embedClient)

	// Phase 1: chunk every file raw.
	var pending []struct {
		chunk   indexer.ChunkResult
		path    string
		lang    string
		hash    string
	}
	for _, path := range paths {
		hash, err := computeFileHash(path)
		if err != nil {
			return 0, 0, err
		}
		language, err := detectLanguage(path)
		if err != nil {
			return 0, 0, err
		}
		chunks, err := chunker.ChunkFileRaw(path, language)
		if err != nil {
			return 0, 0, err
		}
		if err := store.RemoveDeletedFile(path); err != nil {
			return 0, 0, err
		}
		for _, c := range chunks {
			pending = append(pending, struct {
				chunk   indexer.ChunkResult
				path    string
				lang    string
				hash    string
			}{chunk: c, path: path, lang: language, hash: hash})
		}
		indexedFiles++
	}

	// Phase 2: batch-embed.
	allChunks := make([]indexer.ChunkResult, len(pending))
	for i, p := range pending {
		allChunks[i] = p.chunk
	}
	if err := chunker.BatchEmbedChunks(allChunks, 0); err != nil {
		return 0, 0, err
	}

	// Phase 3: save.
	for i, p := range pending {
		if allChunks[i].Vector == nil {
			continue
		}
		if _, err := store.SaveChunk(storage.CodeChunk{
			FilePath:  p.path,
			Content:   allChunks[i].Content,
			StartLine: allChunks[i].StartLine,
			EndLine:   allChunks[i].EndLine,
			Language:  p.lang,
			Category:  classifyCategory(p.lang),
			Vector:    allChunks[i].Vector,
			Signature: allChunks[i].Signature,
			Purpose:   allChunks[i].Purpose,
		}); err != nil {
			return 0, 0, err
		}
		if err := store.UpsertIndexedFile(p.path, p.hash); err != nil {
			return 0, 0, err
		}
		count++
	}
	return indexedFiles, count, nil
}

func deleteSkippedChunks(store *storage.SQLiteStorage, skippedPaths []string) error {
	for _, path := range skippedPaths {
		if err := store.DeleteChunksByPath(path); err != nil {
			return err
		}
	}
	return nil
}

// --- list_projects ---

func makeListProjectsHandler() func(context.Context, *mcpSDK.CallToolRequest, listProjectsInput) (*mcpSDK.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcpSDK.CallToolRequest, input listProjectsInput) (*mcpSDK.CallToolResult, any, error) {
		return runListProjects(input)
	}
}

func runListProjects(input listProjectsInput) (*mcpSDK.CallToolResult, any, error) {
	path := strings.TrimSpace(input.Path)
	if path == "" {
		wd, err := os.Getwd()
		if err != nil {
			return nil, nil, err
		}
		path = wd
	}

	projects, err := workspace.DiscoverNxProjectNames(path)
	if err != nil {
		return nil, nil, err
	}
	if projects == nil {
		projects = []string{}
	}

	return mcpTextResult(listProjectsOutput{Projects: projects})
}

func newProjectManager(projectBaseDir string) (*storage.ProjectManager, error) {
	pm, err := storage.NewProjectManager(projectBaseDir)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize project manager: %w", err)
	}
	return pm, nil
}

// --- shared helpers ---

func mcpTextResult(result any) (*mcpSDK.CallToolResult, any, error) {
	text, err := stringifyMCPResult(result)
	if err != nil {
		return nil, nil, err
	}
	return &mcpSDK.CallToolResult{Content: []mcpSDK.Content{&mcpSDK.TextContent{Text: text}}}, nil, nil
}
