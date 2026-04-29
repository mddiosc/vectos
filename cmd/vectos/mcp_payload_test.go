package main

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"vectos/internal/storage"
	"vectos/internal/workspace"
)

func TestMCPSearchPayloadSizes(t *testing.T) {
	outPath := os.Getenv("MCP_BENCHMARK_OUT")
	if outPath == "" {
		t.Skip("MCP_BENCHMARK_OUT not set")
	}

	queries := []struct {
		name  string
		query string
	}{
		{name: "Fallback to text search", query: "how does Vectos fall back to text search when embeddings fail"},
		{name: "Docs and metadata exclusion", query: "where are docs and dependency metadata excluded from indexing"},
		{name: "TypeScript and React chunking", query: "how are TypeScript and React files chunked structurally"},
		{name: "Hybrid ranking logic", query: "how does hybrid ranking rerank search results and remove duplicates"},
		{name: "Nx logical scope roots", query: "how does Nx scope resolution expand logical roots from dependencies"},
	}

	results := []storage.CodeChunk{
		{FilePath: "/Users/mddiosc/develop/personal/vectos/cmd/vectos/benchmark.go", StartLine: 48, EndLine: 92, Language: "go", Category: "source", Score: 0.8426, Content: strings.Repeat("fallback and text search content ", 12)},
		{FilePath: "/Users/mddiosc/develop/personal/vectos/internal/config/embedding.go", StartLine: 66, EndLine: 87, Language: "go", Category: "source", Score: 0.7632, Content: strings.Repeat("second result content ", 12)},
		{FilePath: "/Users/mddiosc/develop/personal/vectos/internal/setup/claude.go", StartLine: 82, EndLine: 94, Language: "go", Category: "source", Score: 0.7584, Content: strings.Repeat("third result content ", 12)},
		{FilePath: "/Users/mddiosc/develop/personal/vectos/internal/setup/opencode.go", StartLine: 105, EndLine: 117, Language: "go", Category: "source", Score: 0.7549, Content: strings.Repeat("fourth result content ", 12)},
		{FilePath: "/Users/mddiosc/develop/personal/vectos/internal/setup/codex.go", StartLine: 83, EndLine: 95, Language: "go", Category: "source", Score: 0.7525, Content: strings.Repeat("fifth result content ", 12)},
		{FilePath: "/Users/mddiosc/develop/personal/vectos/internal/embeddings/client.go", StartLine: 98, EndLine: 107, Language: "go", Category: "source", Score: 0.7410, Content: strings.Repeat("sixth result content ", 12)},
		{FilePath: "/Users/mddiosc/develop/personal/vectos/cmd/vectos/commands_search.go", StartLine: 14, EndLine: 48, Language: "go", Category: "source", Score: 0.7320, Content: strings.Repeat("seventh result content ", 12)},
		{FilePath: "/Users/mddiosc/develop/personal/vectos/cmd/vectos/search_ranking.go", StartLine: 34, EndLine: 104, Language: "go", Category: "source", Score: 0.7240, Content: strings.Repeat("eighth result content ", 12)},
		{FilePath: "/Users/mddiosc/develop/personal/vectos/internal/workspace/workspace.go", StartLine: 66, EndLine: 117, Language: "go", Category: "source", Score: 0.7180, Content: strings.Repeat("ninth result content ", 12)},
		{FilePath: "/Users/mddiosc/develop/personal/vectos/cmd/vectos/runtime_paths.go", StartLine: 182, EndLine: 203, Language: "go", Category: "source", Score: 0.7120, Content: strings.Repeat("tenth result content ", 12)},
	}
	counts := []int{3, 5, 10}
	rows := make([]map[string]any, 0, len(queries)*len(counts))

	for _, tc := range queries {
		for _, count := range counts {
			payload := buildMCPSearchPayload(&workspace.Scope{Name: "vectos"}, tc.query, searchRun{Mode: "semantic_hybrid", Results: results[:count]})
			encoded, err := json.Marshal(payload)
			if err != nil {
				t.Fatalf("%s: marshal payload: %v", tc.name, err)
			}
			rows = append(rows, map[string]any{"query": tc.name, "results": count, "bytes": len(encoded)})
		}
	}

	encoded, err := json.MarshalIndent(rows, "", "  ")
	if err != nil {
		t.Fatalf("marshal results: %v", err)
	}
	if err := os.WriteFile(outPath, encoded, 0644); err != nil {
		t.Fatalf("write results: %v", err)
	}
}

func TestMCPSearchPayloadPreviewHeuristic(t *testing.T) {
	shortContent := strings.Repeat("short query preview content ", 12)
	longContent := strings.Repeat("long query preview content ", 12)
	short := buildMCPSearchPayload(&workspace.Scope{Name: "vectos"}, "fallback text search", searchRun{Mode: "semantic_hybrid", Results: []storage.CodeChunk{{Score: 0.92, Content: shortContent}, {Score: 0.60, Content: shortContent}}})
	long := buildMCPSearchPayload(&workspace.Scope{Name: "vectos"}, "how does Nx scope resolution expand logical roots from dependencies", searchRun{Mode: "semantic_hybrid", Results: []storage.CodeChunk{{Score: 0.71, Content: longContent}, {Score: 0.70, Content: longContent}}})

	if len(short.Results) == 0 || len(long.Results) == 0 {
		t.Fatal("expected results")
	}
	if got := len(short.Results[0].Preview); got != searchPreviewShort {
		t.Fatalf("expected short query preview to use short limit, got %d", got)
	}
	if got := len(long.Results[0].Preview); got != searchPreviewLong {
		t.Fatalf("expected long query preview to use long limit, got %d", got)
	}
}
