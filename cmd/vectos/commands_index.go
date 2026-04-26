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

func runIndex(projectBaseDir string, embedConfig config.EmbeddingConfig, filePath string, projectName string, changedPaths []string) {
	fmt.Printf("Indexing: %s\n", filePath)

	absolutePath, err := filepath.Abs(filePath)
	if err != nil {
		log.Fatalf("error resolving path: %v", err)
	}

	scope, err := workspace.ResolveScope(absolutePath, projectName)
	if err != nil {
		log.Fatalf("error resolving project scope: %v", err)
	}

	fmt.Printf("Project: %s\n", scope.Name)
	if scope.IsWorkspace() {
		fmt.Printf("Workspace: %s (%s)\n", scope.WorkspaceRoot, scope.WorkspaceType)
	}
	fmt.Printf("Root: %s\n", scope.PrimaryRoot)

	pm, err := storage.NewProjectManager(projectBaseDir)
	if err != nil {
		log.Fatalf("error initializing project manager: %v", err)
	}

	embedClient, providerInfo, err := embeddings.ResolveEmbedder(embedConfig)
	if err != nil {
		log.Fatalf("error resolving embedding provider: %v", err)
	}
	store, err := storage.NewSQLiteStorageForProjectName(pm, scope.Name)
	if err != nil {
		log.Fatalf("error opening database: %v", err)
	}
	defer store.Close()

	if err := store.SetIndexMetadata(storage.IndexMetadata{
		Provider:   providerInfo.Provider,
		Model:      providerInfo.Model,
		Dimensions: providerInfo.Dimensions,
	}); err != nil {
		log.Fatalf("error saving index metadata: %v", err)
	}

	chunker := indexer.NewSimpleChunker(indexer.ChunkConfig{MaxLines: 10}, embedClient)
	paths, skippedPaths, err := collectIndexablePaths(scope.Roots)
	if err != nil {
		log.Fatalf("error collecting indexable paths: %v", err)
	}

	if len(changedPaths) > 0 {
		paths, skippedPaths, err = filterChangedPaths(scope, paths, skippedPaths, changedPaths)
		if err != nil {
			log.Fatalf("error filtering changed paths: %v", err)
		}
	}

	totalFiles := len(paths)
	if len(changedPaths) > 0 {
		fmt.Printf("Found %d changed supported files\n", totalFiles)
	} else {
		fmt.Printf("Found %d supported files\n", totalFiles)
	}
	fmt.Println("Processing files...")

	indexedFiles := 0
	count := 0
	for i, path := range paths {
		language, err := detectLanguage(path)
		if err != nil {
			log.Printf("warning: skipping %s — unsupported language: %v", path, err)
			continue
		}

		chunks, err := chunker.ChunkFile(path, language)
		if err != nil {
			log.Printf("warning: failed to chunk %s: %v", path, err)
			continue
		}

		if err := store.DeleteChunksByPath(path); err != nil {
			log.Printf("warning: failed to clear previous chunks for %s: %v", path, err)
			continue
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
			})
			if err != nil {
				log.Printf("warning: failed to save chunk for %s: %v", path, err)
				continue
			}
			count++
		}

		indexedFiles++
		if indexedFiles == 1 || indexedFiles == totalFiles || indexedFiles%25 == 0 {
			fmt.Printf("Progress: %d/%d files, %d chunks indexed\n", indexedFiles, totalFiles, count)
		} else if i == totalFiles-1 {
			fmt.Printf("Progress: %d/%d files, %d chunks indexed\n", indexedFiles, totalFiles, count)
		}
	}

	fmt.Println("Cleaning excluded directories...")
	for _, root := range scope.Roots {
		for _, excludedDir := range collectExcludedDirs(root) {
			if err := store.DeleteChunksByPathPrefix(excludedDir); err != nil {
				log.Printf("warning: failed to clean excluded dir %s: %v", excludedDir, err)
			}
		}
	}

	for _, skippedPath := range skippedPaths {
		if err := store.DeleteChunksByPath(skippedPath); err != nil {
			log.Printf("warning: failed to clear skipped path %s: %v", skippedPath, err)
		}
	}

	fmt.Printf("Done: %d files, %d chunks indexed (project: %s)\n", indexedFiles, count, scope.Name)
}
