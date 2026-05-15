package main

import (
	"testing"

	"vectos/internal/storage"
)

func TestRerankHybridResultsBoostsRelevantLocalSource(t *testing.T) {
	results := rerankHybridResults("filter changed file paths during indexing", []storage.CodeChunk{
		{
			FilePath: "/tmp/project/example-site/dist/assets/post.js",
			Content:  "changed file paths during indexing explained in generated blog output",
			Category: "source",
			Score:    0.61,
		},
		{
			FilePath:  "/tmp/project/vectos/cmd/vectos/main.go",
			Content:   "func filterChangedPaths(scope workspace.Scope, paths, skippedPaths, changedPaths []string) ([]string, []string, error) {",
			Category:  "source",
			StartLine: 903,
			EndLine:   951,
			Score:     0.58,
		},
	}, 5)

	if len(results) == 0 {
		t.Fatal("expected reranked results")
	}
	if results[0].FilePath != "/tmp/project/vectos/cmd/vectos/main.go" {
		t.Fatalf("expected local source result first, got %s", results[0].FilePath)
	}
}

func TestRerankHybridResultsDeduplicatesOverlappingChunks(t *testing.T) {
	results := rerankHybridResults("project path resolution", []storage.CodeChunk{
		{FilePath: "/tmp/project/internal/storage/project_manager.go", StartLine: 1, EndLine: 43, Category: "source", Content: "package storage", Score: 0.60},
		{FilePath: "/tmp/project/internal/storage/project_manager.go", StartLine: 25, EndLine: 41, Category: "source", Content: "func (pm *ProjectManager) BaseDir() string {", Score: 0.59},
		{FilePath: "/tmp/project/internal/storage/sqlite.go", StartLine: 21, EndLine: 59, Category: "source", Content: "type IndexStats struct {", Score: 0.58},
	}, 5)

	if len(results) != 2 {
		t.Fatalf("expected 2 deduplicated results, got %d", len(results))
	}
	if results[0].FilePath != "/tmp/project/internal/storage/project_manager.go" {
		t.Fatalf("expected project_manager result first, got %s", results[0].FilePath)
	}
}

func TestRerankHybridResultsPenalizesTestFiles(t *testing.T) {
	results := rerankHybridResults("structural chunking for TypeScript and React", []storage.CodeChunk{
		{FilePath: "/tmp/project/internal/indexer/chunker_test.go", Content: "TestChunkStructuredTSXSeparatesPreludeAndBlocks", Category: "source", Score: 0.78},
		{FilePath: "/tmp/project/internal/indexer/chunker.go", Content: "func (s *SimpleChunker) chunkStructuredFileImpl(filePath, language string, lines []string, embed bool) []ChunkResult {", Category: "source", Score: 0.77},
	}, 5)

	if len(results) == 0 {
		t.Fatal("expected reranked results")
	}
	if results[0].FilePath != "/tmp/project/internal/indexer/chunker.go" {
		t.Fatalf("expected implementation file first, got %s", results[0].FilePath)
	}
}

func TestRerankHybridResultsPenalizesHelpText(t *testing.T) {
	results := rerankHybridResults("how does Vectos fall back to text search when embeddings fail", []storage.CodeChunk{
		{FilePath: "/tmp/project/cmd/vectos/cli_help.go", Content: "fmt.Println(\"Search the index using semantic similarity. Falls back to keyword search when semantic search is unavailable or returns no results.\")", Category: "source", Score: 0.81},
		{FilePath: "/tmp/project/cmd/vectos/benchmark.go", Content: "func executeSearch(store *storage.SQLiteStorage, embedConfig config.EmbeddingConfig, query string, limit int) (searchRun, error) {", Category: "source", Score: 0.79},
	}, 5)

	if len(results) == 0 {
		t.Fatal("expected reranked results")
	}
	if results[0].FilePath != "/tmp/project/cmd/vectos/benchmark.go" {
		t.Fatalf("expected implementation file first, got %s", results[0].FilePath)
	}
}

func TestRerankHybridResultsPenalizesNonSourceFilesForBroadQueries(t *testing.T) {
	results := rerankHybridResults("where is routing defined", []storage.CodeChunk{
		{FilePath: "/tmp/project/openspec/config.yaml", Content: "routing definition for docs and config", Category: "infra_config", Score: 0.77},
		{FilePath: "/tmp/project/src/router/routes.tsx", Content: "export const router = createBrowserRouter([...])", Category: "source", Score: 0.75},
	}, 5)

	if len(results) == 0 {
		t.Fatal("expected reranked results")
	}
	if results[0].FilePath != "/tmp/project/src/router/routes.tsx" {
		t.Fatalf("expected source routing file first, got %s", results[0].FilePath)
	}
}

