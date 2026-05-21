package main

import (
	"encoding/json"
	"os"
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

	fileResults := []storage.SearchFileResult{
		{FilePath: "/tmp/project/cmd/vectos/benchmark.go", FileName: "benchmark.go", Language: "go", Category: "source", Relevance: 0.8426, LineRanges: []storage.LineRange{{Start: 48, End: 92}}, Signatures: []string{"func executeSearch(...)"}, Preview: "func executeSearch(store *SQLiteStorage, query string, limit int) (searchRun, error)"},
		{FilePath: "/tmp/project/internal/config/embedding.go", FileName: "embedding.go", Language: "go", Category: "source", Relevance: 0.7632, LineRanges: []storage.LineRange{{Start: 66, End: 87}}, Signatures: []string{"type EmbeddingConfig"}, Preview: "type EmbeddingConfig struct { Provider string Model string Dimensions int }"},
		{FilePath: "/tmp/project/internal/setup/claude.go", FileName: "claude.go", Language: "go", Category: "source", Relevance: 0.7584, LineRanges: []storage.LineRange{{Start: 82, End: 94}}, Signatures: []string{"func setupClaude(...)"}, Preview: "func setupClaude(projectDir string) error { return writeConfig(claudeConfigPath) }"},
		{FilePath: "/tmp/project/internal/setup/opencode.go", FileName: "opencode.go", Language: "go", Category: "source", Relevance: 0.7549, LineRanges: []storage.LineRange{{Start: 105, End: 117}}, Signatures: []string{"func setupOpenCode(...)"}, Preview: "func setupOpenCode(projectDir string) error { return patchOpenCodeConfig(projectDir) }"},
		{FilePath: "/tmp/project/internal/setup/codex.go", FileName: "codex.go", Language: "go", Category: "source", Relevance: 0.7525, LineRanges: []storage.LineRange{{Start: 83, End: 95}}, Signatures: []string{"func setupCodex(...)"}, Preview: "func setupCodex(projectDir string) error { return writeCodexInstructions(projectDir) }"},
		{FilePath: "/tmp/project/internal/embeddings/client.go", FileName: "client.go", Language: "go", Category: "source", Relevance: 0.7410, LineRanges: []storage.LineRange{{Start: 98, End: 107}}, Signatures: []string{"type EmbeddingClient"}, Preview: "type EmbeddingClient struct { provider string model string httpClient *http.Client }"},
		{FilePath: "/tmp/project/cmd/vectos/commands_search.go", FileName: "commands_search.go", Language: "go", Category: "source", Relevance: 0.7320, LineRanges: []storage.LineRange{{Start: 14, End: 48}}, Signatures: []string{"func runSearch(...)"}, Preview: "func runSearch(projectBaseDir string, query string, docsMode bool, verbose bool)"},
		{FilePath: "/tmp/project/cmd/vectos/search_ranking.go", FileName: "search_ranking.go", Language: "go", Category: "source", Relevance: 0.7240, LineRanges: []storage.LineRange{{Start: 34, End: 104}}, Signatures: []string{"func rerankHybridResults(...)"}, Preview: "func rerankHybridResults(results []CodeChunk, query string) []CodeChunk"},
		{FilePath: "/tmp/project/internal/workspace/workspace.go", FileName: "workspace.go", Language: "go", Category: "source", Relevance: 0.7180, LineRanges: []storage.LineRange{{Start: 66, End: 117}}, Signatures: []string{"type Scope"}, Preview: "type Scope struct { Name string WorkspaceRoot string PrimaryRoot string Roots []string }"},
		{FilePath: "/tmp/project/cmd/vectos/runtime_paths.go", FileName: "runtime_paths.go", Language: "go", Category: "source", Relevance: 0.7120, LineRanges: []storage.LineRange{{Start: 182, End: 203}}, Signatures: []string{"func resolveRuntimePaths(...)"}, Preview: "func resolveRuntimePaths(projectName string) (*workspace.Scope, error)"},
	}
	counts := []int{3, 5, 10}
	rows := make([]map[string]any, 0, len(queries)*len(counts))

	for _, tc := range queries {
		for _, count := range counts {
			payload := buildMCPSearchPayload(&workspace.Scope{Name: "vectos"}, tc.query, searchRun{Mode: "semantic_hybrid", FileResults: fileResults[:count]})
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

func TestMCPSearchPayloadHintForLowConfidence(t *testing.T) {
	shortQuery := "fallback text search"
	longQuery := "how does Nx scope resolution expand logical roots from dependencies"

	highConfFile := storage.SearchFileResult{
		FilePath: "/tmp/vectos/cmd/vectos/main.go", FileName: "main.go", Language: "go", Category: "source",
		Relevance: 0.95, LineRanges: []storage.LineRange{{Start: 100, End: 140}},
		Signatures: []string{"func runMCP(...)"}, Purpose: "runs MCP server",
	}
	lowConfFile := storage.SearchFileResult{
		FilePath: "/tmp/vectos/cmd/vectos/main.go", FileName: "main.go", Language: "go", Category: "source",
		Relevance: 0.74, LineRanges: []storage.LineRange{{Start: 100, End: 140}},
		Signatures: []string{"func runMCP(...)"}, Purpose: "runs MCP server",
	}

	highConfPayload := buildMCPSearchPayload(&workspace.Scope{Name: "vectos"}, shortQuery, searchRun{Mode: "semantic_hybrid", FileResults: []storage.SearchFileResult{highConfFile}})
	lowConfPayload := buildMCPSearchPayload(&workspace.Scope{Name: "vectos"}, longQuery, searchRun{Mode: "semantic_hybrid", FileResults: []storage.SearchFileResult{lowConfFile}})

	if len(highConfPayload.Results) == 0 || len(lowConfPayload.Results) == 0 {
		t.Fatal("expected results")
	}
	if highConfPayload.Results[0].Hint != "" {
		t.Fatalf("expected no hint for high-confidence result (relevance=%d), got %q", int(highConfFile.Relevance*100), highConfPayload.Results[0].Hint)
	}
	if lowConfPayload.Results[0].Hint == "" {
		t.Fatalf("expected hint for low-confidence result (relevance=%d)", int(lowConfFile.Relevance*100))
	}
}

func TestMCPSearchPayloadBudgetWithPreviews(t *testing.T) {
	// Verify that a typical 5-result payload with previews stays under 3KB.
	fileResults := []storage.SearchFileResult{
		{FilePath: "cmd/vectos/search_exec.go", FileName: "search_exec.go", Language: "go", Category: "source", Relevance: 0.85, LineRanges: []storage.LineRange{{Start: 17, End: 41}}, Signatures: []string{"func executeSearch(...)"}, Preview: "func executeSearch(store *SQLiteStorage, embedConfig EmbeddingConfig, query string, limit int) (searchRun, error)"},
		{FilePath: "internal/storage/sqlite.go", FileName: "sqlite.go", Language: "go", Category: "source", Relevance: 0.78, LineRanges: []storage.LineRange{{Start: 461, End: 598}}, Signatures: []string{"func SearchTextRanked(...)"}, Preview: "func (s *SQLiteStorage) SearchTextRanked(query string, limit int) ([]CodeChunk, error)"},
		{FilePath: "cmd/vectos/search_fusion.go", FileName: "search_fusion.go", Language: "go", Category: "source", Relevance: 0.76, LineRanges: []storage.LineRange{{Start: 1, End: 45}}, Signatures: []string{"func fuseResults(...)"}, Preview: "func fuseResults(vectorResults, keywordResults []CodeChunk, k float64, offset int) []CodeChunk"},
		{FilePath: "cmd/vectos/mcp_format.go", FileName: "mcp_format.go", Language: "go", Category: "source", Relevance: 0.72, LineRanges: []storage.LineRange{{Start: 44, End: 77}}, Signatures: []string{"func buildMCPSearchPayload(...)"}, Preview: "func buildMCPSearchPayload(scope *Scope, query string, searchRun searchRun) mcpSearchPayload"},
		{FilePath: "internal/storage/search_result.go", FileName: "search_result.go", Language: "go", Category: "source", Relevance: 0.70, LineRanges: []storage.LineRange{{Start: 33, End: 66}}, Signatures: []string{"func CollapseFileResults(...)"}, Preview: "func CollapseFileResults(chunks []CodeChunk, lineWindow int) []SearchFileResult"},
	}

	payload := buildMCPSearchPayload(&workspace.Scope{Name: "vectos"}, "how does search work", searchRun{
		Mode:        "semantic_hybrid",
		FileResults: fileResults,
	})

	encoded, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	const maxPayloadBytes = 3072 // 3KB budget for 5 results with previews
	if len(encoded) > maxPayloadBytes {
		t.Fatalf("payload exceeds budget: %d bytes > %d bytes", len(encoded), maxPayloadBytes)
	}

	// Verify previews are present for low-confidence results.
	previewCount := 0
	for _, r := range payload.Results {
		if r.Preview != "" {
			previewCount++
		}
	}
	if previewCount == 0 {
		t.Fatal("expected at least one preview in payload")
	}
}