package main

import (
	"context"
	"encoding/json"
	"fmt"
	mcpSDK "github.com/modelcontextprotocol/go-sdk/mcp"
	"log"
	"os"
	"path/filepath"

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
