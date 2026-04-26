package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"vectos/internal/config"
	"vectos/internal/embeddings"
	"vectos/internal/storage"
)

func runSearch(projectBaseDir string, embedConfig config.EmbeddingConfig, query string, projectName string) {
	fmt.Printf("Searching: %q\n", query)

	scope, err := resolveRuntimeScope(projectName)
	if err != nil {
		log.Fatalf("error resolving project scope: %v", err)
	}

	pm, err := storage.NewProjectManager(projectBaseDir)
	if err != nil {
		log.Fatalf("error initializing project manager: %v", err)
	}

	store, err := openStorageForScope(pm, scope)
	if err != nil {
		log.Fatalf("error opening database: %v", err)
	}
	defer store.Close()

	searchRun, err := executeSearch(store, embedConfig, query, 5)
	if err != nil {
		log.Fatalf("error running search: %v", err)
	}
	results := searchRun.Results

	if len(results) == 0 {
		fmt.Println("No results found.")
		return
	}

	if strings.TrimSpace(searchRun.Warning) != "" {
		fmt.Printf("Warning: %s\n", searchRun.Warning)
	}
	fmt.Printf("Search mode: %s\n", searchRun.Mode)

	fmt.Printf("Found %d result(s):\n\n", len(results))
	for _, r := range results {
		fmt.Printf("--- [%s:%d-%d] [%s/%s] ---\n", r.FilePath, r.StartLine, r.EndLine, r.Category, r.Language)
		fmt.Printf("%s\n\n", r.Content)
	}
}

func runStatus(projectBaseDir string, projectName string) {
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("error resolving home directory: %v", err)
	}
	embedConfig, err := config.LoadEmbeddingConfig(home)
	if err != nil {
		log.Fatalf("error loading embedding config: %v", err)
	}

	scope, err := resolveRuntimeScope(projectName)
	if err != nil {
		log.Fatalf("error resolving project scope: %v", err)
	}

	pm, err := storage.NewProjectManager(projectBaseDir)
	if err != nil {
		log.Fatalf("error initializing project manager: %v", err)
	}

	store, err := openStorageForScope(pm, scope)
	if err != nil {
		log.Fatalf("error opening database: %v", err)
	}
	defer store.Close()

	stats, err := store.Stats()
	if err != nil {
		log.Fatalf("error reading index stats: %v", err)
	}

	fmt.Println("Vectos status")
	if scope != nil {
		fmt.Printf("Project scope: %s\n", scope.Name)
		if scope.WorkspaceType != "" {
			fmt.Printf("Workspace type: %s\n", scope.WorkspaceType)
		}
		if len(scope.Roots) > 0 {
			fmt.Printf("Project roots: %s\n", strings.Join(scope.Roots, ", "))
		}
	}
	fmt.Printf("Project DB: %s\n", stats.DatabasePath)
	fmt.Printf("DB size: %d bytes\n", stats.DatabaseSize)
	fmt.Printf("Indexed files: %d\n", stats.FileCount)
	fmt.Printf("Indexed chunks: %d\n", stats.ChunkCount)
	fmt.Printf("Chunks with embeddings: %d\n", stats.EmbeddedCount)
	fmt.Printf("Chunks without embeddings: %d\n", stats.UnembeddedCount)
	if stats.Provider != "" {
		fmt.Printf("Embedding provider: %s\n", stats.Provider)
		fmt.Printf("Embedding model: %s\n", stats.Model)
		fmt.Printf("Embedding dimensions: %d\n", stats.Dimensions)
	}

	providerStatuses := embeddings.InspectProviders(embedConfig)
	if len(providerStatuses) > 0 {
		fmt.Println("Provider health:")
		for _, provider := range providerStatuses {
			state := "not ready"
			if provider.Ready {
				state = "ready"
			}
			fmt.Printf("- %s (%s): %s\n", provider.Provider, provider.Model, state)
			if provider.Message != "" {
				fmt.Printf("  %s\n", provider.Message)
			}
		}
	}

	if _, providerInfo, err := embeddings.ResolveEmbedder(embedConfig); err == nil {
		requiresReindex, err := store.RequiresReindex(providerInfo.Provider, providerInfo.Model, providerInfo.Dimensions)
		if err == nil && requiresReindex {
			fmt.Println("Reindex required: current embedding provider configuration does not match stored index metadata")
		}
	}
}
