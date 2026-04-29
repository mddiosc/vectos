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
	payload := buildMCPSearchPayload(&workspace.Scope{Name: "vectos"}, "search_code MCP tool handler", searchRun{
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

func TestBuildMCPSearchPayloadUsesAdaptivePreview(t *testing.T) {
	query := "how does Nx scope resolution expand logical roots from dependencies"
	payload := buildMCPSearchPayload(&workspace.Scope{Name: "vectos"}, query, searchRun{
		Mode: "semantic_hybrid",
		Results: []storage.CodeChunk{{
			FilePath:  "/tmp/vectos/cmd/vectos/benchmark.go",
			StartLine: 48,
			EndLine:   92,
			Language:  "go",
			Category:  "source",
			Score:     0.74,
			Content:   strings.Repeat("fallback and text search content ", 12),
		}, {
			FilePath:  "/tmp/vectos/internal/config/embedding.go",
			StartLine: 66,
			EndLine:   87,
			Language:  "go",
			Category:  "source",
			Score:     0.73,
			Content:   strings.Repeat("second result content ", 12),
		}},
	})

	if len(payload.Results) != 2 {
		t.Fatalf("expected two results, got %d", len(payload.Results))
	}
	if got, want := len(payload.Results[0].Preview), searchPreviewLong; got > want {
		t.Fatalf("expected long preview for top result, got %d > %d", got, want)
	}
	if got, want := len(payload.Results[1].Preview), searchPreviewMedium; got > want {
		t.Fatalf("expected medium preview for second result, got %d > %d", got, want)
	}
}
