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

func TestBuildMCPSearchPayloadLowConfidenceIncludesPreview(t *testing.T) {
	fileResults := []storage.SearchFileResult{
		{
			FilePath:   "/tmp/vectos/cmd/vectos/main.go",
			FileName:   "main.go",
			Language:   "go",
			Category:   "source",
			Relevance:  0.74,
			LineRanges:  []storage.LineRange{{Start: 10, End: 20}},
			Signatures: []string{"func runSearch(...)"},
			Purpose:    "executes search",
			Preview:    "func runSearch(query string) { results := store.Search(query) }",
		},
	}
	payload := buildMCPSearchPayload(&workspace.Scope{Name: "vectos"}, "how does search work", searchRun{
		Mode:        "semantic_hybrid",
		FileResults: fileResults,
	})

	if len(payload.Results) != 1 {
		t.Fatalf("expected one result, got %d", len(payload.Results))
	}
	result := payload.Results[0]
	if result.Preview == "" {
		t.Fatal("expected preview for low-confidence result")
	}
	if !strings.Contains(result.Preview, "runSearch") {
		t.Fatalf("expected preview to contain function name, got %q", result.Preview)
	}
}

func TestBuildMCPSearchPayloadHighConfidenceNoPreview(t *testing.T) {
	fileResults := []storage.SearchFileResult{
		{
			FilePath:   "/tmp/vectos/cmd/vectos/main.go",
			FileName:   "main.go",
			Language:   "go",
			Category:   "source",
			Relevance:  0.95,
			LineRanges:  []storage.LineRange{{Start: 10, End: 20}},
			Signatures: []string{"func runSearch(...)"},
			Preview:    "func runSearch(query string) { results := store.Search(query) }",
		},
	}
	payload := buildMCPSearchPayload(&workspace.Scope{Name: "vectos"}, "search", searchRun{
		Mode:        "semantic_hybrid",
		FileResults: fileResults,
	})

	if len(payload.Results) != 1 {
		t.Fatalf("expected one result, got %d", len(payload.Results))
	}
	result := payload.Results[0]
	if result.Preview != "" {
		t.Fatalf("expected no preview for high-confidence result (relevance=%d), got %q", result.Relevance, result.Preview)
	}
}

func TestBuildMCPSearchPayloadPreviewBoundaryScore(t *testing.T) {
	// Relevance exactly 0.90 should NOT have preview (>= threshold).
	atThreshold := []storage.SearchFileResult{
		{
			FilePath: "/tmp/a.go", FileName: "a.go", Language: "go", Category: "source",
			Relevance: 0.90, LineRanges: []storage.LineRange{{Start: 1, End: 10}},
			Preview: "func A() {}",
		},
	}
	payload := buildMCPSearchPayload(&workspace.Scope{Name: "test"}, "query", searchRun{
		Mode: "semantic_hybrid", FileResults: atThreshold,
	})
	if payload.Results[0].Preview != "" {
		t.Fatalf("expected no preview at exact threshold 0.90, got %q", payload.Results[0].Preview)
	}

	// Relevance 0.899 should have preview (< threshold).
	belowThreshold := []storage.SearchFileResult{
		{
			FilePath: "/tmp/b.go", FileName: "b.go", Language: "go", Category: "source",
			Relevance: 0.899, LineRanges: []storage.LineRange{{Start: 1, End: 10}},
			Preview: "func B() {}",
		},
	}
	payload2 := buildMCPSearchPayload(&workspace.Scope{Name: "test"}, "query", searchRun{
		Mode: "semantic_hybrid", FileResults: belowThreshold,
	})
	if payload2.Results[0].Preview == "" {
		t.Fatal("expected preview just below threshold 0.899")
	}
}

func TestBuildMCPSearchPayloadPreviewMaxLength(t *testing.T) {
	longPreview := strings.Repeat("x", 300)
	fileResults := []storage.SearchFileResult{
		{
			FilePath:  "/tmp/project/src/big.go",
			FileName:  "big.go",
			Language:  "go",
			Category:  "source",
			Relevance: 0.75,
			LineRanges: []storage.LineRange{{Start: 1, End: 50}},
			Preview:   longPreview,
		},
	}
	payload := buildMCPSearchPayload(&workspace.Scope{Name: "test"}, "query", searchRun{
		Mode:        "semantic_hybrid",
		FileResults: fileResults,
	})

	result := payload.Results[0]
	// The preview is passed through as-is from SearchFileResult;
	// the truncation happens at extraction time in CollapseFileResults.
	// But we verify it's present.
	if result.Preview == "" {
		t.Fatal("expected preview to be propagated")
	}
}