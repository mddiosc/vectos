package main

import (
	"testing"

	"vectos/internal/storage"
)

func TestRerankHybridResultsBoostsRelevantLocalSource(t *testing.T) {
	results := rerankHybridResults("filter changed file paths during indexing", []storage.CodeChunk{
		{
			FilePath: "/tmp/project/mywebsite-2/dist/assets/post.js",
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
