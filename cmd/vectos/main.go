package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	mcpSDK "github.com/modelcontextprotocol/go-sdk/mcp"
	"log"
	"os"
	"path/filepath"
	"strings"
	"vectos/internal/buildinfo"
	"vectos/internal/config"
	"vectos/internal/embeddings"
	"vectos/internal/indexer"
	setupinternal "vectos/internal/setup"
	"vectos/internal/storage"
	"vectos/internal/workspace"
)

func main() {
	// Definición de subcomandos
	indexCmd := flag.NewFlagSet("index", flag.ExitOnError)
	searchCmd := flag.NewFlagSet("search", flag.ExitOnError)
	statusCmd := flag.NewFlagSet("status", flag.ExitOnError)
	mcpCmd := flag.NewFlagSet("mcp", flag.ExitOnError)
	setupCmd := flag.NewFlagSet("setup", flag.ExitOnError)
	indexProject := indexCmd.String("project", "", "Nx project name to index when inside an Nx workspace")
	searchProject := searchCmd.String("project", "", "Nx project name to search when inside an Nx workspace")
	statusProject := statusCmd.String("project", "", "Nx project name to inspect when inside an Nx workspace")

	if len(os.Args) < 2 {
		fmt.Println("Uso: vectos <comando> [argumentos]")
		fmt.Println("Comandos disponibles:")
		fmt.Println("  index <ruta_archivo>  - Indexa un archivo con embeddings")
		fmt.Println("  search <query>        - Busca texto en la base de datos")
		fmt.Println("  status                - Muestra el estado del índice del proyecto actual")
		fmt.Println("  mcp                   - Inicia el servidor MCP para agentes")
		fmt.Println("  setup <agent>         - Configura Vectos para un agente compatible")
		fmt.Println("  version               - Muestra la versión, commit y fecha de build")
		os.Exit(1)
	}

	// Configuración base
	home, _ := os.UserHomeDir()
	projectBaseDir := fmt.Sprintf("%s/.vectos/projects", home)
	embedConfig, err := config.LoadEmbeddingConfig(home)
	if err != nil {
		log.Fatalf("❌ Error cargando configuración de embeddings: %v", err)
	}

	switch os.Args[1] {
	case "index":
		if err := indexCmd.Parse(os.Args[2:]); err != nil {
			log.Fatal(err)
		}
		if indexCmd.NArg() < 1 {
			fmt.Println("Uso: vectos index <ruta_archivo>")
			os.Exit(1)
		}
		filePath := indexCmd.Arg(0)
		runIndex(projectBaseDir, embedConfig, filePath, *indexProject)

	case "search":
		if err := searchCmd.Parse(os.Args[2:]); err != nil {
			log.Fatal(err)
		}
		if searchCmd.NArg() < 1 {
			fmt.Println("Uso: vectos search <query>")
			os.Exit(1)
		}
		query := searchCmd.Arg(0)
		runSearch(projectBaseDir, embedConfig, query, *searchProject)

	case "status":
		if err := statusCmd.Parse(os.Args[2:]); err != nil {
			log.Fatal(err)
		}
		runStatus(projectBaseDir, *statusProject)

	case "mcp":
		if err := mcpCmd.Parse(os.Args[2:]); err != nil {
			log.Fatal(err)
		}
		runMCP(projectBaseDir, embedConfig)

	case "setup":
		if err := setupCmd.Parse(os.Args[2:]); err != nil {
			log.Fatal(err)
		}
		if setupCmd.NArg() < 1 {
			fmt.Println("Uso: vectos setup <agent>")
			os.Exit(1)
		}
		runSetup(setupCmd.Arg(0))

	case "version":
		fmt.Printf("vectos %s\n", buildinfo.Version)
		fmt.Printf("commit: %s\n", buildinfo.Commit)
		fmt.Printf("built:  %s\n", buildinfo.Date)

	default:
		fmt.Printf("Comando desconocido: %s\n", os.Args[1])
		os.Exit(1)
	}
}

func runIndex(projectBaseDir string, embedConfig config.EmbeddingConfig, filePath string, projectName string) {
	fmt.Printf("🚀 Indexando con IA: %s\n", filePath)

	absolutePath, err := filepath.Abs(filePath)
	if err != nil {
		log.Fatalf("❌ Error resolviendo ruta: %v", err)
	}

	scope, err := workspace.ResolveScope(absolutePath, projectName)
	if err != nil {
		log.Fatalf("❌ Error resolviendo proyecto: %v", err)
	}

	pm, err := storage.NewProjectManager(projectBaseDir)
	if err != nil {
		log.Fatalf("❌ Error PM: %v", err)
	}

	embedClient, providerInfo, err := embeddings.ResolveEmbedder(embedConfig)
	if err != nil {
		log.Fatalf("❌ Error embeddings: %v", err)
	}
	store, err := storage.NewSQLiteStorageForProjectName(pm, scope.Name)
	if err != nil {
		log.Fatalf("❌ Error DB: %v", err)
	}
	defer store.Close()

	if err := store.SetIndexMetadata(storage.IndexMetadata{
		Provider:   providerInfo.Provider,
		Model:      providerInfo.Model,
		Dimensions: providerInfo.Dimensions,
	}); err != nil {
		log.Fatalf("❌ Error guardando metadata del índice: %v", err)
	}

	chunker := indexer.NewSimpleChunker(indexer.ChunkConfig{
		MaxLines: 10,
	}, embedClient)

	paths, err := collectIndexablePaths(scope.Roots)
	if err != nil {
		log.Fatalf("❌ Error indexando: %v", err)
	}

	indexedFiles := 0
	count := 0
	for _, path := range paths {
		language, err := detectLanguage(path)
		if err != nil {
			log.Printf("⚠️ Error lenguaje en %s: %v", path, err)
			continue
		}

		chunks, err := chunker.ChunkFile(path, language)
		if err != nil {
			log.Printf("⚠️ Error Chunker en %s: %v", path, err)
			continue
		}

		if err := store.DeleteChunksByPath(path); err != nil {
			log.Printf("⚠️ Error limpiando chunks previos en %s: %v", path, err)
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
				log.Printf("⚠️ Error guardando trozo en %s: %v", path, err)
				continue
			}
			count++
		}

		indexedFiles++
	}

	for _, root := range scope.Roots {
		for _, excludedDir := range collectExcludedDirs(root) {
			if err := store.DeleteChunksByPathPrefix(excludedDir); err != nil {
				log.Printf("⚠️ Error limpiando directorio excluido %s: %v", excludedDir, err)
			}
		}
	}

	fmt.Printf("✅ Éxito: %d archivos y %d trozos indexados con vectores en %s (%s)\n", indexedFiles, count, projectBaseDir, scope.Name)
}

func runSearch(projectBaseDir string, embedConfig config.EmbeddingConfig, query string, projectName string) {
	fmt.Printf("🔍 Buscando: '%s'\n", query)

	scope, err := resolveRuntimeScope(projectName)
	if err != nil {
		log.Fatalf("❌ Error resolviendo proyecto: %v", err)
	}

	pm, err := storage.NewProjectManager(projectBaseDir)
	if err != nil {
		log.Fatalf("❌ Error PM: %v", err)
	}

	store, err := openStorageForScope(pm, scope)
	if err != nil {
		log.Fatalf("❌ Error DB: %v", err)
	}
	defer store.Close()

	results := []storage.CodeChunk(nil)
	if embedClient, providerInfo, err := embeddings.ResolveEmbedder(embedConfig); err == nil {
		requiresReindex, reindexErr := store.RequiresReindex(providerInfo.Provider, providerInfo.Model, providerInfo.Dimensions)
		if reindexErr == nil && !requiresReindex {
			queryVector, embedErr := embedClient.GetEmbedding(query)
			if embedErr == nil {
				results, err = store.SearchSemantic(queryVector, 5)
				if err != nil {
					log.Fatalf("❌ Error búsqueda semántica: %v", err)
				}
			}
		}
	}

	if len(results) == 0 {
		results, err = store.SearchText(query)
		if err != nil {
			log.Fatalf("❌ Error búsqueda keyword: %v", err)
		}
	}

	if len(results) == 0 {
		fmt.Println("No se encontraron resultados.")
		return
	}

	fmt.Printf("Encontrados %d resultados:\n\n", len(results))
	for _, r := range results {
		fmt.Printf("--- [%s:%d-%d] [%s/%s] ---\n", r.FilePath, r.StartLine, r.EndLine, r.Category, r.Language)
		fmt.Printf("%s\n\n", r.Content)
	}
}

func runStatus(projectBaseDir string, projectName string) {
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("❌ Error home: %v", err)
	}
	embedConfig, err := config.LoadEmbeddingConfig(home)
	if err != nil {
		log.Fatalf("❌ Error cargando configuración de embeddings: %v", err)
	}

	scope, err := resolveRuntimeScope(projectName)
	if err != nil {
		log.Fatalf("❌ Error resolviendo proyecto: %v", err)
	}

	pm, err := storage.NewProjectManager(projectBaseDir)
	if err != nil {
		log.Fatalf("❌ Error PM: %v", err)
	}

	store, err := openStorageForScope(pm, scope)
	if err != nil {
		log.Fatalf("❌ Error DB: %v", err)
	}
	defer store.Close()

	stats, err := store.Stats()
	if err != nil {
		log.Fatalf("❌ Error obteniendo estado: %v", err)
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

func runMCP(projectBaseDir string, embedConfig config.EmbeddingConfig) {
	// En modo MCP escribimos logs a archivo para depurar sin contaminar stdout/stderr del protocolo.
	home, _ := os.UserHomeDir()
	logPath := filepath.Join(home, ".vectos", "vectos-mcp.log")
	if err := os.MkdirAll(filepath.Dir(logPath), 0755); err == nil {
		if f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err == nil {
			log.SetOutput(f)
			log.Printf("starting vectos mcp")
		} else {
			log.SetOutput(os.Stderr)
		}
	} else {
		log.SetOutput(os.Stderr)
	}

	server := mcpSDK.NewServer(&mcpSDK.Implementation{
		Name:    "vectos",
		Version: buildinfo.Version,
	}, &mcpSDK.ServerOptions{
		Capabilities: &mcpSDK.ServerCapabilities{
			Tools: &mcpSDK.ToolCapabilities{ListChanged: false},
		},
	})

	type searchCodeInput struct {
		Query   string `json:"query" jsonschema:"search query for code context"`
		Path    string `json:"path,omitempty" jsonschema:"optional project path to scope the search"`
		Project string `json:"project,omitempty" jsonschema:"optional Nx project name when searching inside a workspace"`
	}

	mcpSDK.AddTool(server, &mcpSDK.Tool{
		Name:        "search_code",
		Description: "Search through the codebase using semantic search with keyword fallback",
	}, func(ctx context.Context, req *mcpSDK.CallToolRequest, input searchCodeInput) (*mcpSDK.CallToolResult, any, error) {
		scope, err := resolveToolScope(input.Path, input.Project)
		if err != nil {
			return nil, nil, err
		}

		pm, err := storage.NewProjectManager(projectBaseDir)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to initialize project manager: %w", err)
		}

		store, err := openStorageForScope(pm, scope)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to open storage: %w", err)
		}
		defer store.Close()

		embedClient, providerInfo, err := embeddings.ResolveEmbedder(embedConfig)
		if err != nil {
			results, textErr := store.SearchText(input.Query)
			if textErr != nil {
				return nil, nil, err
			}
			text, textErr := stringifyMCPResult(results)
			if textErr != nil {
				return nil, nil, textErr
			}
			return &mcpSDK.CallToolResult{Content: []mcpSDK.Content{&mcpSDK.TextContent{Text: text}}}, nil, nil
		}

		requiresReindex, err := store.RequiresReindex(providerInfo.Provider, providerInfo.Model, providerInfo.Dimensions)
		if err == nil && requiresReindex {
			results, textErr := store.SearchText(input.Query)
			if textErr != nil {
				return nil, nil, textErr
			}
			text, textErr := stringifyMCPResult(map[string]any{
				"warning": "index metadata does not match current embedding provider; semantic results may be stale until reindex",
				"results": results,
			})
			if textErr != nil {
				return nil, nil, textErr
			}
			return &mcpSDK.CallToolResult{Content: []mcpSDK.Content{&mcpSDK.TextContent{Text: text}}}, nil, nil
		}
		queryVector, err := embedClient.GetEmbedding(input.Query)
		results := []storage.CodeChunk(nil)
		if err == nil {
			results, err = store.SearchSemantic(queryVector, 5)
			if err != nil {
				return nil, nil, err
			}
		}

		if len(results) == 0 {
			results, err = store.SearchText(input.Query)
			if err != nil {
				return nil, nil, err
			}
		}

		text, err := stringifyMCPResult(results)
		if err != nil {
			return nil, nil, err
		}

		return &mcpSDK.CallToolResult{
			Content: []mcpSDK.Content{&mcpSDK.TextContent{Text: text}},
		}, nil, nil
	})

	type indexProjectInput struct {
		Path    string `json:"path" jsonschema:"path to a file or directory to index"`
		Project string `json:"project,omitempty" jsonschema:"optional Nx project name when indexing inside a workspace"`
	}

	mcpSDK.AddTool(server, &mcpSDK.Tool{
		Name:        "index_project",
		Description: "Index a directory to make it searchable via semantic embeddings",
	}, func(ctx context.Context, req *mcpSDK.CallToolRequest, input indexProjectInput) (*mcpSDK.CallToolResult, any, error) {
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
		store, err := storage.NewSQLiteStorageForProjectName(pm, scope.Name)
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
		paths, err := collectIndexablePaths(scope.Roots)
		if err != nil {
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
				})
				if err != nil {
					return nil, nil, err
				}
				count++
			}
			indexedFiles++
		}

		return &mcpSDK.CallToolResult{
			Content: []mcpSDK.Content{&mcpSDK.TextContent{Text: fmt.Sprintf("Successfully indexed %d files and %d chunks for %s", indexedFiles, count, scope.Name)}},
		}, nil, nil
	})

	log.Printf("vectos mcp ready")

	if err := server.Run(context.Background(), &mcpSDK.StdioTransport{}); err != nil {
		log.Fatalf("❌ MCP Server Error: %v", err)
	}
}

func stringifyMCPResult(result interface{}) (string, error) {
	if text, ok := result.(string); ok {
		return text, nil
	}

	encoded, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", err
	}

	return string(encoded), nil
}

func runSetup(agent string) {
	if err := setupinternal.Run(agent); err != nil {
		log.Fatalf("❌ Error configurando %s: %v", agent, err)
	}
	fmt.Printf("✅ Vectos configurado para %s.\n", agent)
}


func collectIndexablePaths(inputPaths []string) ([]string, error) {
	var paths []string
	seen := map[string]struct{}{}
	for _, path := range inputPaths {
		absPath, err := filepath.Abs(path)
		if err != nil {
			return nil, err
		}

		info, err := os.Stat(absPath)
		if err != nil {
			return nil, err
		}

		if !info.IsDir() {
			if _, err := detectLanguage(absPath); err == nil {
				if _, ok := seen[absPath]; !ok {
					paths = append(paths, absPath)
					seen[absPath] = struct{}{}
				}
				continue
			}
			return nil, fmt.Errorf("unsupported file type: %s", absPath)
		}

		err = filepath.Walk(absPath, func(current string, info os.FileInfo, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}

			if info.IsDir() {
				if shouldSkipDir(info.Name()) {
					return filepath.SkipDir
				}
				return nil
			}

			if _, err := detectLanguage(current); err == nil {
				if _, ok := seen[current]; !ok {
					paths = append(paths, current)
					seen[current] = struct{}{}
				}
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	if len(paths) == 0 {
		return nil, fmt.Errorf("no supported files found in selected scope")
	}

	return paths, nil
}

func shouldSkipDir(name string) bool {
	switch name {
	case ".git", "node_modules", ".opencode", ".vectos", "coverage", "playwright-report", "test-results", "dist", ".next", "build":
		return true
	default:
		return false
	}
}

func collectExcludedDirs(root string) []string {
	var excluded []string
	_ = filepath.Walk(root, func(current string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() && shouldSkipDir(info.Name()) {
			excluded = append(excluded, current)
			return filepath.SkipDir
		}
		return nil
	})
	return excluded
}

func openStorageForScope(pm *storage.ProjectManager, scope *workspace.Scope) (*storage.SQLiteStorage, error) {
	if scope == nil || strings.TrimSpace(scope.Name) == "" {
		return storage.NewSQLiteStorage(pm)
	}

	return storage.NewSQLiteStorageForProjectName(pm, scope.Name)
}

func resolveRuntimeScope(projectName string) (*workspace.Scope, error) {
	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	scope, err := workspace.ResolveScope(wd, projectName)
	if err != nil {
		if strings.TrimSpace(projectName) == "" {
			return nil, nil
		}
		return nil, err
	}

	return &scope, nil
}

func resolveToolScope(path string, projectName string) (*workspace.Scope, error) {
	if strings.TrimSpace(path) == "" {
		if strings.TrimSpace(projectName) != "" {
			return &workspace.Scope{Name: projectName}, nil
		}
		return resolveRuntimeScope(projectName)
	}

	scope, err := workspace.ResolveScope(path, projectName)
	if err != nil {
		return nil, err
	}
	return &scope, nil
}

func detectLanguage(path string) (string, error) {
	baseName := filepath.Base(path)
	lowerBase := strings.ToLower(baseName)
	switch {
	case baseName == "Dockerfile" || strings.HasPrefix(baseName, "Dockerfile."):
		return "dockerfile", nil
	case baseName == "Makefile":
		return "makefile", nil
	case baseName == ".gitignore":
		return "gitignore", nil
	case strings.HasPrefix(lowerBase, "docker-compose") && (strings.HasSuffix(lowerBase, ".yml") || strings.HasSuffix(lowerBase, ".yaml")):
		return "yaml.compose", nil
	case baseName == "BUILD":
		return "bazel.build", nil
	case baseName == "BUILD.bazel":
		return "bazel.build", nil
	case baseName == "WORKSPACE":
		return "bazel.workspace", nil
	case baseName == "MODULE.bazel":
		return "bazel.module", nil
	}

	switch strings.ToLower(filepath.Ext(path)) {
	case ".go":
		return "go", nil
	case ".js":
		return "javascript", nil
	case ".jsx":
		return "jsx", nil
	case ".ts":
		return "typescript", nil
	case ".tsx":
		return "tsx", nil
	case ".py":
		return "python", nil
	case ".java":
		return "java", nil
	case ".json":
		return "json", nil
	case ".sh":
		return "shell", nil
	case ".md":
		return "markdown", nil
	case ".toml":
		return "toml", nil
	case ".ini":
		return "ini", nil
	case ".xml":
		return "xml", nil
	case ".properties":
		return "properties", nil
	case ".yml":
		return "yaml", nil
	case ".yaml":
		return "yaml", nil
	case ".bzl":
		return "bazel.bzl", nil
	default:
		return "", fmt.Errorf("unsupported file type: %s", path)
	}
}

func classifyCategory(language string) string {
	switch {
	case language == "shell":
		return "scripts"
	case language == "markdown", language == "gitignore":
		return "docs"
	case language == "json", language == "toml", language == "ini", language == "xml", language == "properties", language == "makefile":
		return classifyMetadataCategory(language)
	case language == "dockerfile", strings.HasPrefix(language, "yaml"), strings.HasPrefix(language, "bazel"):
		return "infra_config"
	default:
		return "source"
	}
}

func classifyMetadataCategory(language string) string {
	switch language {
	case "json", "toml", "properties", "xml", "makefile":
		return "dependency_metadata"
	default:
		return "infra_config"
	}
}
