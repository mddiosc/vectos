package main

import (
	"context"
	"encoding/json"
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
	home, _ := os.UserHomeDir()
	projectBaseDir := fmt.Sprintf("%s/.vectos/projects", home)
	embedConfig, err := config.LoadEmbeddingConfig(home)
	if err != nil {
		log.Fatalf("error loading embedding config: %v", err)
	}

	runCLI(appContext{
		projectBaseDir: projectBaseDir,
		embedConfig:    embedConfig,
		flags:          newCLIFlags(),
	}, os.Args[1:])
}

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

	chunker := indexer.NewSimpleChunker(indexer.ChunkConfig{
		MaxLines: 10,
	}, embedClient)

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

	results := []storage.CodeChunk(nil)
	searchRun, err := executeSearch(store, embedConfig, query, 5)
	if err != nil {
		log.Fatalf("error running search: %v", err)
	}
	results = searchRun.Results

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

func runMCP(projectBaseDir string, embedConfig config.EmbeddingConfig) {
	// In MCP mode, write logs to a file to avoid polluting the stdio protocol stream.
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

		payload := buildMCPSearchPayload(scope, searchRun)

		text, err := stringifyMCPResult(payload)
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
		Changed string `json:"changed,omitempty" jsonschema:"optional comma-separated changed file paths for incremental refresh"`
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
		paths, skippedPaths, err := collectIndexablePaths(scope.Roots)
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

		for _, skippedPath := range skippedPaths {
			if err := store.DeleteChunksByPath(skippedPath); err != nil {
				return nil, nil, err
			}
		}

		label := "files"
		if len(changedPaths) > 0 {
			label = "changed files"
		}
		_ = label

		payload := buildMCPIndexPayload(scope, changedPaths, indexedFiles, count, len(skippedPaths))
		text, err := stringifyMCPResult(payload)
		if err != nil {
			return nil, nil, err
		}

		return &mcpSDK.CallToolResult{
			Content: []mcpSDK.Content{&mcpSDK.TextContent{Text: text}},
		}, nil, nil
	})

	log.Printf("vectos mcp ready")

	if err := server.Run(context.Background(), &mcpSDK.StdioTransport{}); err != nil {
		log.Fatalf("MCP server error: %v", err)
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

func runSetup(agent string, uninstall bool) {
	action := "setting up"
	if uninstall {
		action = "removing"
	}

	if err := setupinternal.Run(agent, uninstall); err != nil {
		log.Fatalf("error %s %s: %v", action, agent, err)
	}
	if uninstall {
		fmt.Printf("Vectos setup removed for %s.\n", agent)
		return
	}

	fmt.Printf("Vectos configured for %s.\n", agent)
}

func collectIndexablePaths(inputPaths []string) ([]string, []string, error) {
	var paths []string
	var skippedPaths []string
	seen := map[string]struct{}{}
	skippedSeen := map[string]struct{}{}
	for _, path := range inputPaths {
		absPath, err := filepath.Abs(path)
		if err != nil {
			return nil, nil, err
		}

		info, err := os.Stat(absPath)
		if err != nil {
			return nil, nil, err
		}

		if !info.IsDir() {
			if language, err := detectLanguage(absPath); err == nil {
				if !shouldIndexLanguage(language) {
					if _, ok := skippedSeen[absPath]; !ok {
						skippedPaths = append(skippedPaths, absPath)
						skippedSeen[absPath] = struct{}{}
					}
					continue
				}
				if _, ok := seen[absPath]; !ok {
					paths = append(paths, absPath)
					seen[absPath] = struct{}{}
				}
				continue
			}
			return nil, nil, fmt.Errorf("unsupported file type: %s", absPath)
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

			if language, err := detectLanguage(current); err == nil {
				if !shouldIndexLanguage(language) {
					if _, ok := skippedSeen[current]; !ok {
						skippedPaths = append(skippedPaths, current)
						skippedSeen[current] = struct{}{}
					}
					return nil
				}
				if _, ok := seen[current]; !ok {
					paths = append(paths, current)
					seen[current] = struct{}{}
				}
			}
			return nil
		})
		if err != nil {
			return nil, nil, err
		}
	}

	if len(paths) == 0 {
		return nil, nil, fmt.Errorf("no supported files found in selected scope")
	}

	return paths, skippedPaths, nil
}

func shouldSkipDir(name string) bool {
	switch name {
	case ".git", "node_modules", ".opencode", ".vectos", "coverage", "playwright-report", "test-results", "dist", ".next", "build":
		return true
	default:
		return false
	}
}

func shouldIndexLanguage(language string) bool {
	category := classifyCategory(language)
	return category != "docs" && category != "dependency_metadata"
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

func parseChangedPaths(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	changed := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		changed = append(changed, trimmed)
	}
	return changed
}

func filterChangedPaths(scope workspace.Scope, paths, skippedPaths, changedPaths []string) ([]string, []string, error) {
	allowedRoots := make([]string, 0, len(scope.Roots))
	for _, root := range scope.Roots {
		absRoot, err := filepath.Abs(root)
		if err != nil {
			return nil, nil, err
		}
		allowedRoots = append(allowedRoots, absRoot)
	}

	pathSet := make(map[string]struct{}, len(paths))
	for _, path := range paths {
		pathSet[path] = struct{}{}
	}

	skippedSet := make(map[string]struct{}, len(skippedPaths))
	for _, path := range skippedPaths {
		skippedSet[path] = struct{}{}
	}

	var filteredPaths []string
	var filteredSkipped []string
	seenPaths := map[string]struct{}{}
	seenSkipped := map[string]struct{}{}

	for _, changed := range changedPaths {
		resolved, err := resolveChangedPath(scope.PrimaryRoot, changed)
		if err != nil {
			return nil, nil, err
		}
		if !isWithinRoots(resolved, allowedRoots) {
			continue
		}
		if _, ok := pathSet[resolved]; ok {
			if _, seen := seenPaths[resolved]; !seen {
				filteredPaths = append(filteredPaths, resolved)
				seenPaths[resolved] = struct{}{}
			}
			continue
		}
		if _, ok := skippedSet[resolved]; ok || !fileExists(resolved) {
			if _, seen := seenSkipped[resolved]; !seen {
				filteredSkipped = append(filteredSkipped, resolved)
				seenSkipped[resolved] = struct{}{}
			}
		}
	}

	return filteredPaths, filteredSkipped, nil
}

func resolveChangedPath(baseRoot, changed string) (string, error) {
	if filepath.IsAbs(changed) {
		return filepath.Clean(changed), nil
	}
	return filepath.Abs(filepath.Join(baseRoot, changed))
}

func isWithinRoots(path string, roots []string) bool {
	for _, root := range roots {
		if path == root || strings.HasPrefix(path, root+string(os.PathSeparator)) {
			return true
		}
	}
	return false
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
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
	case baseName == ".editorconfig":
		return "ini", nil
	case baseName == ".gitignore":
		return "gitignore", nil
	case baseName == ".prettierignore" || baseName == ".eslintignore":
		return "gitignore", nil
	case baseName == ".npmrc" || baseName == ".yarnrc" || baseName == ".nvmrc" || baseName == ".prettierrc" || baseName == ".tool-versions":
		return "config", nil
	case baseName == "gradlew" || baseName == "mvnw":
		return "shell", nil
	case strings.HasSuffix(baseName, ".gradle.kts"):
		return "gradle", nil
	case strings.HasSuffix(baseName, ".lock") || baseName == "bun.lockb":
		return "lockfile", nil
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
	case ".mjs", ".cjs":
		return "javascript", nil
	case ".jsx":
		return "jsx", nil
	case ".ts":
		return "typescript", nil
	case ".mts", ".cts":
		return "typescript", nil
	case ".tsx":
		return "tsx", nil
	case ".py":
		return "python", nil
	case ".java":
		return "java", nil
	case ".kt":
		return "kotlin", nil
	case ".kts":
		return "kotlin", nil
	case ".json":
		return "json", nil
	case ".sh":
		return "shell", nil
	case ".md":
		return "markdown", nil
	case ".mdx":
		return "markdown", nil
	case ".toml":
		return "toml", nil
	case ".ini":
		return "ini", nil
	case ".conf":
		return "config", nil
	case ".xml":
		return "xml", nil
	case ".properties":
		return "properties", nil
	case ".gradle":
		return "gradle", nil
	case ".sql":
		return "sql", nil
	case ".proto":
		return "proto", nil
	case ".graphql", ".gql":
		return "graphql", nil
	case ".css":
		return "css", nil
	case ".scss":
		return "scss", nil
	case ".sass":
		return "sass", nil
	case ".less":
		return "less", nil
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
	case language == "json", language == "toml", language == "ini", language == "xml", language == "properties", language == "makefile", language == "gradle", language == "lockfile", language == "config":
		return classifyMetadataCategory(language)
	case language == "dockerfile", strings.HasPrefix(language, "yaml"), strings.HasPrefix(language, "bazel"):
		return "infra_config"
	default:
		return "source"
	}
}

func classifyMetadataCategory(language string) string {
	switch language {
	case "json", "toml", "properties", "xml", "makefile", "gradle", "lockfile":
		return "dependency_metadata"
	default:
		return "infra_config"
	}
}
