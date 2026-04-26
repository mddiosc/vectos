package main

import (
	"context"
	"encoding/json"
	mcpSDK "github.com/modelcontextprotocol/go-sdk/mcp"
	"log"
	"os"
	"path/filepath"

	"vectos/internal/buildinfo"
	"vectos/internal/config"
)

type searchCodeInput struct {
	Query   string `json:"query" jsonschema:"search query for code context"`
	Path    string `json:"path,omitempty" jsonschema:"optional project path to scope the search"`
	Project string `json:"project,omitempty" jsonschema:"optional Nx project name when searching inside a workspace"`
}

type indexProjectInput struct {
	Path    string `json:"path" jsonschema:"path to a file or directory to index"`
	Project string `json:"project,omitempty" jsonschema:"optional Nx project name when indexing inside a workspace"`
	Changed string `json:"changed,omitempty" jsonschema:"optional comma-separated changed file paths for incremental refresh"`
}

func runMCP(projectBaseDir string, embedConfig config.EmbeddingConfig) {
	configureMCPLogging()
	server := newMCPServer()
	registerMCPTools(server, projectBaseDir, embedConfig)

	log.Printf("vectos mcp ready")

	if err := server.Run(context.Background(), &mcpSDK.StdioTransport{}); err != nil {
		log.Fatalf("MCP server error: %v", err)
	}
}

func configureMCPLogging() {
	home, _ := os.UserHomeDir()
	logPath := filepath.Join(home, ".vectos", "vectos-mcp.log")
	if err := os.MkdirAll(filepath.Dir(logPath), 0755); err == nil {
		if f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err == nil {
			log.SetOutput(f)
			log.Printf("starting vectos mcp")
			return
		}
	}
	log.SetOutput(os.Stderr)
}

func newMCPServer() *mcpSDK.Server {
	return mcpSDK.NewServer(&mcpSDK.Implementation{
		Name:    "vectos",
		Version: buildinfo.Version,
	}, &mcpSDK.ServerOptions{
		Capabilities: &mcpSDK.ServerCapabilities{
			Tools: &mcpSDK.ToolCapabilities{ListChanged: false},
		},
	})
}

func registerMCPTools(server *mcpSDK.Server, projectBaseDir string, embedConfig config.EmbeddingConfig) {
	mcpSDK.AddTool(server, &mcpSDK.Tool{
		Name:        "search_code",
		Description: "Search through the codebase using semantic search with keyword fallback",
	}, makeSearchCodeHandler(projectBaseDir, embedConfig))

	mcpSDK.AddTool(server, &mcpSDK.Tool{
		Name:        "index_project",
		Description: "Index a directory to make it searchable via semantic embeddings",
	}, makeIndexProjectHandler(projectBaseDir, embedConfig))
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
