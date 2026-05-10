package main

import (
	"fmt"
	"log"
	"path/filepath"

	"vectos/internal/config"
	"vectos/internal/embeddings"
	"vectos/internal/indexer"
	"vectos/internal/storage"
	"vectos/internal/workspace"
)

func prepareStoreForIndexing(store *storage.SQLiteStorage, changedPaths []string) error {
	if len(changedPaths) > 0 {
		return nil
	}
	return store.DeleteAllChunks()
}

type indexEnv struct {
	store   *storage.SQLiteStorage
	chunker *indexer.SimpleChunker
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

	chunker := indexer.NewSimpleChunker(indexer.ChunkConfig{MaxLines: 10}, embedClient)
	return &indexEnv{store: store, chunker: chunker}
}

func runIndex(projectBaseDir string, embedConfig config.EmbeddingConfig, filePath string, projectName string, changedPaths []string, docsOnly bool) {
	fmt.Printf("Indexing: %s\n", filePath)

	if docsOnly {
		fmt.Printf("Mode: documentation only\n")
	}

	absolutePath, err := filepath.Abs(filePath)
	if err != nil {
		log.Fatalf("error resolving path: %v", err)
	}

	scope := resolveAndPrintScope(absolutePath, projectName)
	env := setupIndexing(projectBaseDir, scope, embedConfig, docsOnly)
	defer env.store.Close()

	paths, skippedPaths, err := collectIndexablePaths(scope.Roots, docsOnly)
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

	indexedFiles, count := indexPathsIntoStore(env.store, env.chunker, paths)
	if indexedFiles > 0 {
		fmt.Printf("Progress: %d/%d files, %d chunks indexed\n", indexedFiles, totalFiles, count)
	}

	fmt.Println("Cleaning excluded directories...")
	cleanupExcludedAndSkipped(env.store, scope, skippedPaths)

	fmt.Printf("Done: %d files, %d chunks indexed (project: %s)\n", indexedFiles, count, scope.Name)
}
