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

// printHelp prints global usage for all subcommands.
func printHelp() {
	fmt.Printf("vectos %s\n\n", buildinfo.Version)
	fmt.Println("Local-first code context engine for AI agents.")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  vectos <command> [arguments]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  index   <path>   Index a file or directory with semantic embeddings")
	fmt.Println("  search  <query>  Search the index using semantic or keyword search")
	fmt.Println("  status           Show index status for the current project")
	fmt.Println("  mcp              Start the MCP server for agent clients")
	fmt.Println("  setup   <agent>  Configure Vectos for a supported agent client")
	fmt.Println("  version          Show version, commit, and build date")
	fmt.Println("  help             Show this help message")
	fmt.Println()
	fmt.Println("Use 'vectos help <command>' or 'vectos <command> --help' for command details.")
}

// printSubcommandHelp prints help for a specific subcommand.
func printSubcommandHelp(cmd string) {
	switch cmd {
	case "index":
		fmt.Println("Usage:")
		fmt.Println("  vectos index <path> [flags]")
		fmt.Println()
		fmt.Println("Index a file or directory. Generates semantic embeddings for all supported")
		fmt.Println("files and stores them in a project-scoped SQLite database.")
		fmt.Println()
		fmt.Println("Flags:")
		fmt.Println("  --project <name>   Nx project name to scope the index (optional)")
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  vectos index .")
		fmt.Println("  vectos index ./src --project my-app")
	case "search":
		fmt.Println("Usage:")
		fmt.Println("  vectos search <query> [flags]")
		fmt.Println()
		fmt.Println("Search the index using semantic similarity. Falls back to keyword search")
		fmt.Println("when semantic search is unavailable or returns no results.")
		fmt.Println()
		fmt.Println("Flags:")
		fmt.Println("  --project <name>   Nx project name to scope the search (optional)")
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  vectos search \"checkout payment flow\"")
		fmt.Println("  vectos search \"auth middleware\" --project api")
	case "status":
		fmt.Println("Usage:")
		fmt.Println("  vectos status [flags]")
		fmt.Println()
		fmt.Println("Show index stats, embedding provider health, and reindex status for")
		fmt.Println("the current project.")
		fmt.Println()
		fmt.Println("Flags:")
		fmt.Println("  --project <name>   Nx project name to inspect (optional)")
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  vectos status")
		fmt.Println("  vectos status --project my-app")
	case "mcp":
		fmt.Println("Usage:")
		fmt.Println("  vectos mcp")
		fmt.Println()
		fmt.Println("Start the MCP server over stdio. Exposes vectos_index_project and")
		fmt.Println("vectos_search_code tools to compatible agent clients (e.g. OpenCode).")
		fmt.Println()
		fmt.Println("Logs are written to ~/.vectos/vectos-mcp.log to avoid polluting stdio.")
	case "setup":
		fmt.Println("Usage:")
		fmt.Println("  vectos setup <agent> [--uninstall]")
		fmt.Println()
		fmt.Println("Configure or remove Vectos MCP integration for a supported agent client.")
		fmt.Println("Optionally installs global guidance so the agent prefers Vectos")
		fmt.Println("search tools before falling back to generic file-search tools.")
		fmt.Println()
		fmt.Println("Supported agents:")
		fmt.Println("  opencode")
		fmt.Println("  claude")
		fmt.Println("  codex")
		fmt.Println()
		fmt.Println("Flags:")
		fmt.Println("  --uninstall   Remove the Vectos MCP entry and managed guidance for the selected agent")
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  vectos setup opencode")
		fmt.Println("  vectos setup claude")
		fmt.Println("  vectos setup codex")
		fmt.Println("  vectos setup opencode --uninstall")
	case "version":
		fmt.Println("Usage:")
		fmt.Println("  vectos version")
		fmt.Println()
		fmt.Println("Print the release version, git commit, and build date.")
	default:
		fmt.Printf("unknown command: %s\n", cmd)
		fmt.Println("Run 'vectos help' for a list of available commands.")
		os.Exit(1)
	}
}

func main() {
	// Subcommand flag sets.
	indexCmd := flag.NewFlagSet("index", flag.ExitOnError)
	searchCmd := flag.NewFlagSet("search", flag.ExitOnError)
	statusCmd := flag.NewFlagSet("status", flag.ExitOnError)
	mcpCmd := flag.NewFlagSet("mcp", flag.ExitOnError)
	setupCmd := flag.NewFlagSet("setup", flag.ExitOnError)
	indexProject := indexCmd.String("project", "", "Nx project name to index when inside an Nx workspace")
	searchProject := searchCmd.String("project", "", "Nx project name to search when inside an Nx workspace")
	statusProject := statusCmd.String("project", "", "Nx project name to inspect when inside an Nx workspace")
	setupUninstall := setupCmd.Bool("uninstall", false, "Remove the Vectos MCP setup for the selected agent")

	// Global --help / -h before any subcommand.
	if len(os.Args) >= 2 && (os.Args[1] == "--help" || os.Args[1] == "-h") {
		printHelp()
		os.Exit(0)
	}

	if len(os.Args) < 2 {
		printHelp()
		os.Exit(0)
	}

	// Base configuration.
	home, _ := os.UserHomeDir()
	projectBaseDir := fmt.Sprintf("%s/.vectos/projects", home)
	embedConfig, err := config.LoadEmbeddingConfig(home)
	if err != nil {
		log.Fatalf("error loading embedding config: %v", err)
	}

	switch os.Args[1] {
	case "help":
		if len(os.Args) >= 3 {
			printSubcommandHelp(os.Args[2])
		} else {
			printHelp()
		}

	case "index":
		if len(os.Args) >= 3 && (os.Args[2] == "--help" || os.Args[2] == "-h") {
			printSubcommandHelp("index")
			os.Exit(0)
		}
		if err := indexCmd.Parse(os.Args[2:]); err != nil {
			log.Fatal(err)
		}
		if indexCmd.NArg() < 1 {
			printSubcommandHelp("index")
			os.Exit(1)
		}
		runIndex(projectBaseDir, embedConfig, indexCmd.Arg(0), *indexProject)

	case "search":
		if len(os.Args) >= 3 && (os.Args[2] == "--help" || os.Args[2] == "-h") {
			printSubcommandHelp("search")
			os.Exit(0)
		}
		if err := searchCmd.Parse(os.Args[2:]); err != nil {
			log.Fatal(err)
		}
		if searchCmd.NArg() < 1 {
			printSubcommandHelp("search")
			os.Exit(1)
		}
		runSearch(projectBaseDir, embedConfig, searchCmd.Arg(0), *searchProject)

	case "status":
		if len(os.Args) >= 3 && (os.Args[2] == "--help" || os.Args[2] == "-h") {
			printSubcommandHelp("status")
			os.Exit(0)
		}
		if err := statusCmd.Parse(os.Args[2:]); err != nil {
			log.Fatal(err)
		}
		runStatus(projectBaseDir, *statusProject)

	case "mcp":
		if len(os.Args) >= 3 && (os.Args[2] == "--help" || os.Args[2] == "-h") {
			printSubcommandHelp("mcp")
			os.Exit(0)
		}
		if err := mcpCmd.Parse(os.Args[2:]); err != nil {
			log.Fatal(err)
		}
		runMCP(projectBaseDir, embedConfig)

	case "setup":
		setupArgs, showHelp := normalizeSetupArgs(os.Args[2:])
		if showHelp {
			printSubcommandHelp("setup")
			os.Exit(0)
		}
		if err := setupCmd.Parse(setupArgs); err != nil {
			log.Fatal(err)
		}
		if setupCmd.NArg() < 1 {
			printSubcommandHelp("setup")
			os.Exit(1)
		}
		runSetup(setupCmd.Arg(0), *setupUninstall)

	case "version":
		if len(os.Args) >= 3 && (os.Args[2] == "--help" || os.Args[2] == "-h") {
			printSubcommandHelp("version")
			os.Exit(0)
		}
		fmt.Printf("vectos %s\n", buildinfo.Version)
		fmt.Printf("commit: %s\n", buildinfo.Commit)
		fmt.Printf("built:  %s\n", buildinfo.Date)

	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		fmt.Fprintln(os.Stderr, "Run 'vectos help' for a list of available commands.")
		os.Exit(1)
	}
}

func runIndex(projectBaseDir string, embedConfig config.EmbeddingConfig, filePath string, projectName string) {
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

	totalFiles := len(paths)
	fmt.Printf("Found %d supported files\n", totalFiles)
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
	if embedClient, providerInfo, err := embeddings.ResolveEmbedder(embedConfig); err == nil {
		requiresReindex, reindexErr := store.RequiresReindex(providerInfo.Provider, providerInfo.Model, providerInfo.Dimensions)
		if reindexErr == nil && !requiresReindex {
			queryVector, embedErr := embedClient.GetEmbedding(query)
			if embedErr == nil {
				results, err = store.SearchSemantic(queryVector, 5)
				if err != nil {
					log.Fatalf("error running semantic search: %v", err)
				}
			}
		}
	}

	if len(results) == 0 {
		results, err = store.SearchText(query)
		if err != nil {
			log.Fatalf("error running keyword search: %v", err)
		}
	}

	if len(results) == 0 {
		fmt.Println("No results found.")
		return
	}

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
		paths, skippedPaths, err := collectIndexablePaths(scope.Roots)
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

		for _, skippedPath := range skippedPaths {
			if err := store.DeleteChunksByPath(skippedPath); err != nil {
				return nil, nil, err
			}
		}

		return &mcpSDK.CallToolResult{
			Content: []mcpSDK.Content{&mcpSDK.TextContent{Text: fmt.Sprintf("Successfully indexed %d files and %d chunks for %s", indexedFiles, count, scope.Name)}},
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

func normalizeSetupArgs(args []string) ([]string, bool) {
	flags := make([]string, 0, len(args))
	positionals := make([]string, 0, len(args))
	showHelp := false

	for _, arg := range args {
		switch arg {
		case "--help", "-h":
			showHelp = true
		case "--uninstall":
			flags = append(flags, arg)
		default:
			positionals = append(positionals, arg)
		}
	}

	return append(flags, positionals...), showHelp
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
