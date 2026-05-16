package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"vectos/internal/config"
	"vectos/internal/content"
	"vectos/internal/embeddings"
	"vectos/internal/indexer"
	"vectos/internal/storage"
	"vectos/internal/vectorindex"
	"vectos/internal/workspace"
)

func prepareStoreForIndexing(store *storage.SQLiteStorage, changedPaths []string) error {
	if len(changedPaths) > 0 {
		return nil
	}
	// No longer delete all chunks on full reindex — we use hash-based
	// caching to skip unchanged files. Stale files (deleted from disk)
	// are cleaned up by cleanupExcludedAndSkipped after indexing.
	return nil
}

type indexEnv struct {
	store       *storage.SQLiteStorage
	chunker     *indexer.SimpleChunker
	embedConfig config.EmbeddingConfig
}

func resolveAndPrintScope(absolutePath, projectName string) workspace.Scope {
	scope, err := workspace.ResolveScope(absolutePath, projectName)
	if err != nil {
		log.Fatalf("error resolving project scope: %v", err)
	}
	fmt.Printf("Project: %s\n", scope.Name)
	if scope.IsWorkspace() {
		fmt.Printf("Workspace: %s (%s)\n", scope.WorkspaceRoot, scope.WorkspaceType)
	}
	fmt.Printf("Root: %s\n", scope.PrimaryRoot)
	if scope.IsWorkspace() && len(scope.Roots) > 1 {
		fmt.Println("Internal libs:")
		for i, root := range scope.Roots {
			if i == 0 {
				continue // skip primary root, already printed
			}
			fmt.Printf("  - %s\n", root)
		}
	}
	for _, w := range scope.Warnings {
		fmt.Printf("Warning: %s\n", w)
	}
	return scope
}

func setupIndexing(projectBaseDir string, scope workspace.Scope, embedConfig config.EmbeddingConfig, docsOnly bool) *indexEnv {
	pm, err := storage.NewProjectManager(projectBaseDir)
	if err != nil {
		log.Fatalf("error initializing project manager: %v", err)
	}

	embedClient, providerInfo, err := embeddings.ResolveEmbedder(embedConfig)
	if err != nil {
		log.Fatalf("error resolving embedding provider: %v", err)
	}

	var store *storage.SQLiteStorage
	if docsOnly {
		store, err = storage.NewSQLiteStorageForDocsProjectName(pm, scope.Name)
	} else {
		store, err = storage.NewSQLiteStorageForProjectName(pm, scope.Name)
	}
	if err != nil {
		log.Fatalf("error opening database: %v", err)
	}

	if err := store.SetIndexMetadata(storage.IndexMetadata{
		Provider:   providerInfo.Provider,
		Model:      providerInfo.Model,
		Dimensions: providerInfo.Dimensions,
	}); err != nil {
		log.Fatalf("error saving index metadata: %v", err)
	}

	chunker := indexer.NewSimpleChunker(indexer.ChunkConfig{MaxLines: 10, BatchSize: embedConfig.Embedded.BatchSize}, embedClient)
	return &indexEnv{store: store, chunker: chunker, embedConfig: embedConfig}
}

func runIndex(projectBaseDir string, embedConfig config.EmbeddingConfig, filePath string, projectName string, changedPaths []string, docsOnly bool) {
	totalStart := time.Now()
	fmt.Printf("Indexing: %s\n", filePath)

	if docsOnly {
		fmt.Printf("Mode: documentation only\n")
	}

	absolutePath, err := filepath.Abs(filePath)
	if err != nil {
		log.Fatalf("error resolving path: %v", err)
	}

	scope := resolveAndPrintScope(absolutePath, projectName)

	setupStart := time.Now()
	env := setupIndexing(projectBaseDir, scope, embedConfig, docsOnly)
	fmt.Printf("Setup (model init): %v\n", time.Since(setupStart))
	defer env.store.Close()

	// Merge exclusion patterns: global config + project config + gitignore.
	// When the user's home directory cannot be determined, fail open by
	// passing an empty path so LoadIndexConfig skips the global layer
	// cleanly — falling back to a project-local path would silently
	// re-introduce the bug where the "global" config lives inside the
	// project, which is not the documented behaviour.
	globalCfgPath, err := config.GlobalConfigPath()
	if err != nil {
		log.Printf("warning: skipping global config layer: %v", err)
		globalCfgPath = ""
	}
	indexCfg := config.LoadIndexConfig(globalCfgPath, absolutePath)
	excludePatterns := config.MergeExclusionPatterns(
		indexCfg.ExclusionPatterns(docsOnly),
		config.ReadGitignorePatterns(absolutePath),
	)

	paths, skippedPaths, err := content.CollectIndexablePathsWithExclusions(scope.Roots, docsOnly, excludePatterns)
	if err != nil {
		log.Fatalf("error collecting indexable paths: %v", err)
	}

	if len(changedPaths) > 0 {
		paths, skippedPaths, err = filterChangedPaths(scope, paths, skippedPaths, changedPaths)
		if err != nil {
			log.Fatalf("error filtering changed paths: %v", err)
		}
	}
	if err := prepareStoreForIndexing(env.store, changedPaths); err != nil {
		log.Fatalf("error preparing index storage: %v", err)
	}

	totalFiles := len(paths)
	if len(changedPaths) > 0 {
		fmt.Printf("Found %d changed supported files\n", totalFiles)
	} else {
		fmt.Printf("Found %d supported files\n", totalFiles)
	}
	fmt.Println("Processing files...")

	indexedFiles, count := indexPathsIntoStore(env.store, env.chunker, paths, os.Stdout)
	buildVectorIndex(env.store, env.chunker, env.embedConfig.VectorIndex)

	fmt.Println("Cleaning excluded directories...")
	cleanupExcludedAndSkipped(env.store, scope, skippedPaths)

	fmt.Printf("Done: %d files, %d chunks indexed (project: %s) — total wall time: %v\n", indexedFiles, count, scope.Name, time.Since(totalStart))
}

func buildVectorIndex(store *storage.SQLiteStorage, chunker *indexer.SimpleChunker, viCfg config.VectorIndexConfig) {
	if store == nil || chunker == nil {
		return
	}

	// Try to load existing vector index — if it's still valid (hash matches),
	// skip the expensive rebuild entirely.
	if idx, _, _, _, err := store.LoadVectorIndex(); err == nil && idx != nil {
		fmt.Printf("Vector index up to date: %d vectors, %d layers (skipped rebuild)\n", idx.Len(), idx.MaxLevel()+1)
		return
	}

	fmt.Println("Building vector index...")
	embeddingsByID, err := store.GetAllEmbeddings()
	if err != nil {
		log.Printf("warning: vector index build skipped: %v", err)
		return
	}
	if len(embeddingsByID) == 0 {
		log.Println("vector index build skipped: no embeddings found")
		return
	}

	ids := make([]int, 0, len(embeddingsByID))
	var dimension int
	for id, vector := range embeddingsByID {
		ids = append(ids, id)
		if dimension == 0 {
			dimension = len(vector)
		}
	}

	if dimension == 0 {
		log.Println("vector index build skipped: empty embeddings")
		return
	}

	m := viCfg.HNSW_M
	if m <= 0 {
		m = 16
	}
	efCons := viCfg.HNSW_EfConstruction
	if efCons <= 0 {
		efCons = 200
	}
	efSearch := viCfg.HNSW_EfSearch
	if efSearch <= 0 {
		efSearch = 200
	}
	idx := vectorindex.NewHNSW(dimension, vectorindex.Config{M: m, EfConstruction: efCons, EfSearch: efSearch})
	total := len(ids)
	buildStart := time.Now()
	for i, id := range ids {
		idx.Insert(id, embeddingsByID[id])
		if (i+1)%100 == 0 || i+1 == total {
			fmt.Printf("Building vector index: %d/%d vectors\n", i+1, total)
		}
	}
	buildElapsed := time.Since(buildStart)
	fmt.Printf("Vector index insertion: %d vectors in %v\n", total, buildElapsed)

	contentHash, err := store.ChunkTableContentHash()
	if err != nil {
		log.Printf("warning: vector index hash unavailable: %v", err)
		return
	}

	if err := idx.Save(store.VectorIndexPath(), contentHash, "none", nil); err != nil {
		log.Printf("warning: failed to save vector index: %v", err)
		return
	}

	store.SetVectorIndex(idx)
	fmt.Printf("Vector index built: %d vectors, %d layers\n", idx.Len(), idx.MaxLevel()+1)
}
