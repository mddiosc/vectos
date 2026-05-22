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
	"vectos/internal/workspace"
)

type indexEnv struct {
	store       *storage.SQLiteStorage
	chunker     *indexer.SimpleChunker
	embedConfig config.EmbeddingConfig
	metadata    syncIndexMetadataResult
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
	store.SetVectorIndexParams(embedConfig.VectorIndex.HNSW_M, embedConfig.VectorIndex.HNSW_EfConstruction, embedConfig.VectorIndex.HNSW_EfSearch)

	chunkerConfig := indexChunkerConfig(embedConfig.Embedded.BatchSize)
	metadata, err := syncIndexMetadata(store, providerInfo, currentIndexFingerprint(chunkerConfig))
	if err != nil {
		log.Fatalf("error preparing index metadata: %v", err)
	}
	if metadata.Message != "" {
		fmt.Println(metadata.Message)
	}

	chunker := indexer.NewSimpleChunker(chunkerConfig, embedClient)
	return &indexEnv{store: store, chunker: chunker, embedConfig: embedConfig, metadata: metadata}
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

	if len(changedPaths) > 0 && !env.metadata.FullRebuild {
		paths, skippedPaths, err = filterChangedPaths(scope, paths, skippedPaths, changedPaths)
		if err != nil {
			log.Fatalf("error filtering changed paths: %v", err)
		}
	} else if len(changedPaths) > 0 && env.metadata.FullRebuild {
		fmt.Println("Ignoring changed-path filter because index format changed and requires a full rebuild")
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

func buildVectorIndex(store *storage.SQLiteStorage, _ *indexer.SimpleChunker, viCfg config.VectorIndexConfig) {
	if store == nil {
		return
	}
	store.SetVectorIndexParams(viCfg.HNSW_M, viCfg.HNSW_EfConstruction, viCfg.HNSW_EfSearch)

	// Quick skip: the in-memory index is already loaded and valid.
	if idx, _, _, _, err := store.LoadVectorIndex(); err == nil && idx != nil {
		fmt.Printf("Vector index up to date: %d vectors, %d layers (skipped rebuild)\n", idx.Len(), idx.MaxLevel()+1)
		return
	}

	fmt.Println("Building vector index...")
	if err := store.RebuildVectorIndex(); err != nil {
		log.Printf("warning: vector index build failed: %v", err)
		return
	}
	fmt.Println("Vector index built.")
}
