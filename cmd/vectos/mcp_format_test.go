package main

import (
	"strings"
	"testing"

	"vectos/internal/storage"
	"vectos/internal/workspace"
)

func TestBuildMCPMissingIndexPayloadIncludesGuidance(t *testing.T) {
	payload := buildMCPMissingIndexPayload(&workspace.Scope{Name: "vectos", PrimaryRoot: "/tmp/vectos", Roots: []string{"/tmp/vectos"}})
	if !strings.Contains(payload.Guidance, "does not have a usable Vectos index") {
		t.Fatalf("unexpected guidance: %q", payload.Guidance)
	}
	if !strings.Contains(payload.NextAction, "index_project") {
		t.Fatalf("expected next action to mention index_project, got %q", payload.NextAction)
	}
}

func TestBuildMCPSearchPayloadIncludesMetadata(t *testing.T) {
	payload := buildMCPSearchPayload(&workspace.Scope{Name: "vectos"}, searchRun{
		Mode: "semantic_hybrid",
		Results: []storage.CodeChunk{{
			FilePath:  "/tmp/vectos/cmd/vectos/main.go",
			StartLine: 100,
			EndLine:   140,
			Language:  "go",
			Category:  "source",
			Score:     0.87,
			Content:   "func runMCP(projectBaseDir string, embedConfig config.EmbeddingConfig) {\n  // start MCP\n}",
		}},
	})

	if len(payload.Results) != 1 {
		t.Fatalf("expected one result, got %d", len(payload.Results))
	}
	result := payload.Results[0]
	if result.Rank != 1 || result.FileName != "main.go" {
		t.Fatalf("unexpected result metadata: %+v", result)
	}
	if result.Preview == "" || result.Reason == "" {
		t.Fatalf("expected preview and reason, got %+v", result)
	}
}
