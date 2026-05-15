package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"vectos/internal/config"
	"vectos/internal/embeddings"
	"vectos/internal/storage"
	"vectos/internal/workspace"
)

var cliCodeSearch = executeSearch
var cliDocsSearch = executeSearchDocs

func executeSearchForCLI(store *storage.SQLiteStorage, embedConfig config.EmbeddingConfig, query string, docsOnly bool) (searchRun, error) {
	if docsOnly {
		return cliDocsSearch(store, embedConfig, query, 5)
	}
	return cliCodeSearch(store, embedConfig, query, 5)
}

func runSearch(projectBaseDir string, embedConfig config.EmbeddingConfig, query string, projectName string, full bool, docsOnly bool) {
	fmt.Printf("Searching: %q\n", query)

	if docsOnly {
		fmt.Printf("Mode: documentation search\n")
	}

	scope, err := resolveRuntimeScope(projectName)
	if err != nil {
		log.Fatalf("error resolving project scope: %v", err)
	}

	pm, err := storage.NewProjectManager(projectBaseDir)
	if err != nil {
		log.Fatalf("error initializing project manager: %v", err)
	}

	store, err := openStorageForScope(pm, scope, docsOnly)
	if err != nil {
		log.Fatalf("error opening database: %v", err)
	}
	defer store.Close()

	if _, loadedHash, _, _, err := store.LoadVectorIndex(); err != nil {
		log.Printf("warning: vector index not available, using linear scan: %v", err)
	} else {
		// Validate the loaded index matches current chunk content
		currentHash, hashErr := store.ChunkTableContentHash()
		if hashErr != nil || currentHash != loadedHash {
			store.SetVectorIndex(nil)
			log.Printf("warning: vector index is stale (chunks changed), using linear scan")
		}
	}

	searchRun, err := executeSearchForCLI(store, embedConfig, query, docsOnly)
	if err != nil {
		log.Fatalf("error running search: %v", err)
	}
	results := searchRun.Results

	if strings.TrimSpace(searchRun.Warning) != "" {
		fmt.Printf("Warning: %s\n", searchRun.Warning)
	}
	if len(results) == 0 {
		fmt.Println("No results found.")
		return
	}

	fmt.Print(formatSearchResults(query, results, searchRun.Mode, full))
}

func runStatus(projectBaseDir string, projectName string, docsOnly bool) {
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

	store, err := openStorageForScope(pm, scope, docsOnly)
	if err != nil {
		log.Fatalf("error opening database: %v", err)
	}
	defer store.Close()

	stats, err := store.Stats()
	if err != nil {
		log.Fatalf("error reading index stats: %v", err)
	}

	printIndexStats(&stats, scope, docsOnly)
	printProviderHealth(embedConfig)
	checkAndPrintReindexStatus(store, embedConfig)
}

func printIndexStats(stats *storage.IndexStats, scope *workspace.Scope, docsOnly bool) {
	fmt.Println("Vectos status")
	if docsOnly {
		fmt.Println("Mode: documentation index")
	}
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
}

func printProviderHealth(embedConfig config.EmbeddingConfig) {
	providerStatuses := embeddings.InspectProviders(embedConfig)
	if len(providerStatuses) == 0 {
		return
	}
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

func checkAndPrintReindexStatus(store *storage.SQLiteStorage, embedConfig config.EmbeddingConfig) {
	if _, providerInfo, err := embeddings.ResolveEmbedder(embedConfig); err == nil {
		requiresReindex, err := store.RequiresReindex(providerInfo.Provider, providerInfo.Model, providerInfo.Dimensions)
		if err == nil && requiresReindex {
			fmt.Println("Reindex required: current embedding provider configuration does not match stored index metadata")
		}
	}
}
