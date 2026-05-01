package main

import (
	"strings"
	"testing"

	"vectos/internal/storage"
	"vectos/internal/workspace"
)

func TestBuildMCPMissingIndexPayloadIncludesGuidance(t *testing.T) {
	payload := buildMCPMissingIndexPayload(&workspace.Scope{Name: "vectos", PrimaryRoot: "/tmp/vectos", Roots: []string{"/tmp/vectos"}})
	if payload.Guidance != "IDX_MISSING" {
		t.Fatalf("unexpected guidance: %q", payload.Guidance)
	}
	if !strings.Contains(payload.NextAction, "index_project") {
		t.Fatalf("expected next action to mention index_project, got %q", payload.NextAction)
	}
}

func TestBuildMCPSearchPayloadIncludesFileLevelMetadata(t *testing.T) {
	fileResults := []storage.SearchFileResult{
		{
			FilePath:   "/tmp/vectos/cmd/vectos/main.go",
			FileName:   "main.go",
			Language:   "go",
			Category:   "source",
			Relevance:  0.87,
			LineRanges:  []storage.LineRange{{Start: 100, End: 140}},
			Signatures: []string{"func runMCP(projectBaseDir string, embedConfig config.EmbeddingConfig)"},
			Purpose:    "runs the MCP server",
		},
	}
	payload := buildMCPSearchPayload(&workspace.Scope{Name: "vectos"}, "search_code MCP tool handler", searchRun{
		Mode:       "semantic_hybrid",
		FileResults: fileResults,
	})

	if len(payload.Results) != 1 {
		t.Fatalf("expected one result, got %d", len(payload.Results))
	}
	result := payload.Results[0]
	if result.FilePath != "/tmp/vectos/cmd/vectos/main.go" {
		t.Fatalf("unexpected result metadata: %+v", result)
	}
	if len(result.Signatures) == 0 || result.Signatures[0] != "func runMCP(projectBaseDir string, embedConfig config.EmbeddingConfig)" {
		t.Fatalf("expected signatures, got %+v", result.Signatures)
	}
	if result.Relevance != 87 {
		t.Fatalf("expected relevance 87, got %d", result.Relevance)
	}
}

func TestBuildMCPSearchPayloadHighConfidenceNoHint(t *testing.T) {
	fileResults := []storage.SearchFileResult{
		{
			FilePath:   "/tmp/vectos/cmd/vectos/main.go",
			FileName:   "main.go",
			Language:   "go",
			Category:   "source",
			Relevance:  0.95,
			LineRanges:  []storage.LineRange{{Start: 100, End: 140}},
			Signatures: []string{"func runMCP(...)"},
			Purpose:    "runs MCP",
		},
	}
	payload := buildMCPSearchPayload(&workspace.Scope{Name: "vectos"}, "search_code", searchRun{
		Mode:       "semantic_hybrid",
		FileResults: fileResults,
	})

	if len(payload.Results) != 1 {
		t.Fatalf("expected one result, got %d", len(payload.Results))
	}
	result := payload.Results[0]
	if result.Relevance >= 90 && result.Hint != "" {
		t.Fatalf("expected no hint for high-confidence result, got %q", result.Hint)
	}
}

func TestBuildMCPSearchPayloadLowConfidenceHasHint(t *testing.T) {
	fileResults := []storage.SearchFileResult{
		{
			FilePath:   "/tmp/vectos/cmd/vectos/main.go",
			FileName:   "main.go",
			Language:   "go",
			Category:   "source",
			Relevance:  0.74,
			LineRanges:  []storage.LineRange{{Start: 100, End: 140}},
			Signatures: []string{"func runMCP(...)"},
			Purpose:    "runs MCP server",
		},
	}
	payload := buildMCPSearchPayload(&workspace.Scope{Name: "vectos"}, "how does MCP search work", searchRun{
		Mode:       "semantic_hybrid",
		FileResults: fileResults,
	})

	if len(payload.Results) != 1 {
		t.Fatalf("expected one result, got %d", len(payload.Results))
	}
	result := payload.Results[0]
	if result.Hint == "" {
		t.Fatalf("expected hint for low-confidence result (relevance=%d)", result.Relevance)
	}
}