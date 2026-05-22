package main

import (
	"testing"

	"vectos/internal/storage"
)

// Test for RRF fusion of vector + keyword results (tasks 2.5-2.7)

func TestFuseResults_BasicFusion(t *testing.T) {
	vectorResults := []storage.CodeChunk{
		{ID: 1, FilePath: "/tmp/project/src/main.go", Content: "func main()", Score: 0.9},
		{ID: 2, FilePath: "/tmp/project/src/util.go", Content: "func helper()", Score: 0.7},
	}
	keywordResults := []storage.CodeChunk{
		{ID: 3, FilePath: "/tmp/project/src/auth.go", Content: "func login()", Score: 5.0},
		{ID: 1, FilePath: "/tmp/project/src/main.go", Content: "func main()", Score: 3.0},
	}

	fused := fuseResults(vectorResults, keywordResults, 60.0, 10)
	if len(fused) == 0 {
		t.Fatal("expected fused results")
	}
	// Chunk 1 appears in both lists → should get highest RRF score
	if fused[0].ID != 1 {
		t.Fatalf("expected chunk 1 (in both lists) first, got ID %d", fused[0].ID)
	}
	if len(fused) != 3 {
		t.Fatalf("expected 3 unique fused results, got %d", len(fused))
	}
}

func TestFuseResults_ChunkInBothLists(t *testing.T) {
	vectorResults := []storage.CodeChunk{
		{ID: 10, FilePath: "/tmp/a.go", Content: "x", Score: 0.8},
	}
	keywordResults := []storage.CodeChunk{
		{ID: 10, FilePath: "/tmp/a.go", Content: "x", Score: 4.0},
	}

	fused := fuseResults(vectorResults, keywordResults, 60.0, 10)
	if len(fused) != 1 {
		t.Fatalf("expected 1 result, got %d", len(fused))
	}
	if fused[0].ID != 10 {
		t.Fatalf("expected chunk 10, got %d", fused[0].ID)
	}
	// RRF score should be 1/(60+1) + 1/(60+1) ≈ 0.0328
	expectedRRF := 1.0/61.0 + 1.0/61.0
	if fused[0].Score < expectedRRF-0.001 || fused[0].Score > expectedRRF+0.001 {
		t.Fatalf("expected RRF score ~%.4f, got %.4f", expectedRRF, fused[0].Score)
	}
}

func TestFuseResults_EmptyKeywordList(t *testing.T) {
	vectorResults := []storage.CodeChunk{
		{ID: 1, FilePath: "/tmp/a.go", Content: "x", Score: 0.8},
	}

	fused := fuseResults(vectorResults, nil, 60.0, 10)
	if len(fused) != 1 {
		t.Fatalf("expected 1 result for vector-only, got %d", len(fused))
	}
	if fused[0].ID != 1 {
		t.Fatalf("expected chunk 1, got %d", fused[0].ID)
	}
}

func TestFuseResults_BothEmpty(t *testing.T) {
	fused := fuseResults(nil, nil, 60.0, 10)
	if len(fused) != 0 {
		t.Fatalf("expected 0 results for both empty, got %d", len(fused))
	}
}

// Test for fusion penalties (tasks 3.6-3.7)

func TestApplyFusionPenalties_TestFile(t *testing.T) {
	results := []storage.CodeChunk{
		{ID: 1, FilePath: "/tmp/project/internal/indexer/chunker_test.go", Content: "func TestX()", Score: 0.0328},
		{ID: 2, FilePath: "/tmp/project/internal/indexer/chunker.go", Content: "func chunk()", Score: 0.0320},
	}

	penalized := applyFusionPenalties(results)
	// Test file should be penalized (-0.08)
	if penalized[0].Score >= 0.0328-0.07 {
		t.Fatalf("expected test file score penalized, got %.4f", penalized[0].Score)
	}
	// Source file should NOT be penalized
	if penalized[1].Score != 0.0320 {
		t.Fatalf("expected source file score unchanged (0.0320), got %.4f", penalized[1].Score)
	}
}

func TestApplyFusionPenalties_SourceFileNoPenalty(t *testing.T) {
	results := []storage.CodeChunk{
		{ID: 1, FilePath: "/tmp/project/src/main.go", Content: "func main()", Score: 0.05},
	}

	penalized := applyFusionPenalties(results)
	if penalized[0].Score != 0.05 {
		t.Fatalf("expected source file score unchanged (0.05), got %.4f", penalized[0].Score)
	}
}

func TestApplyFusionPenalties_HelpText(t *testing.T) {
	results := []storage.CodeChunk{
		{ID: 1, FilePath: "/tmp/project/cmd/vectos/cli_help.go", Content: "fmt.Println(\"usage: vectos [command] [flags]\n\\nexamples:\\n  vectos search\")", Score: 0.05},
		{ID: 2, FilePath: "/tmp/project/cmd/vectos/search.go", Content: "func executeSearch()", Score: 0.04},
	}

	penalized := applyFusionPenalties(results)
	// Help text file should be penalized (score reduced by 0.10)
	if penalized[0].Score >= 0.0 {
		t.Fatalf("expected help text score to be penalized below 0, got %.4f", penalized[0].Score)
	}
	// Source file should NOT be penalized
	if penalized[1].Score != 0.04 {
		t.Fatalf("expected source file score unchanged (0.04), got %.4f", penalized[1].Score)
	}
}

// Test for dedupeByFile (task 4.4 verification)

func TestDedupeByFile_OverlappingChunks(t *testing.T) {
	results := []storage.CodeChunk{
		{ID: 1, FilePath: "/tmp/project/internal/storage/project_manager.go", StartLine: 1, EndLine: 43, Score: 0.05},
		{ID: 2, FilePath: "/tmp/project/internal/storage/project_manager.go", StartLine: 25, EndLine: 41, Score: 0.04},
		{ID: 3, FilePath: "/tmp/project/internal/storage/sqlite.go", StartLine: 21, EndLine: 59, Score: 0.03},
	}

	deduped := dedupeByFile(results, 10, 2)
	if len(deduped) != 2 {
		t.Fatalf("expected 2 deduplicated results (overlapping from same file removed), got %d", len(deduped))
	}
	if deduped[0].ID != 1 || deduped[1].ID != 3 {
		t.Fatalf("expected IDs 1 and 3, got %d and %d", deduped[0].ID, deduped[1].ID)
	}
}

func TestDedupeByFile_MaxPerFile(t *testing.T) {
	results := []storage.CodeChunk{
		{ID: 1, FilePath: "/tmp/a.go", StartLine: 1, EndLine: 10, Score: 0.05},
		{ID: 2, FilePath: "/tmp/a.go", StartLine: 20, EndLine: 30, Score: 0.04},
		{ID: 3, FilePath: "/tmp/a.go", StartLine: 40, EndLine: 50, Score: 0.03},
		{ID: 4, FilePath: "/tmp/b.go", StartLine: 1, EndLine: 10, Score: 0.02},
	}

	deduped := dedupeByFile(results, 10, 2)
	// Only 2 from a.go + 1 from b.go = 3
	if len(deduped) != 3 {
		t.Fatalf("expected 3 results (max 2 per file), got %d", len(deduped))
	}
}

func TestIsTestFilePath(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		// Go test files
		{"src/auth_test.go", true},
		{"src/auth.go", false},

		// TS/JS test files
		{"src/hooks_test.ts", true},
		{"src/hooks_test.tsx", true},
		{"src/hooks.ts", false},

		// Spec files (Playwright, Jest)
		{"e2e/contact.spec.ts", true},
		{"e2e/contact.spec.tsx", true},
		{"src/components/__tests__/button.spec.js", true},

		// e2e directories
		{"e2e/fixtures/index.ts", true},
		{"e2e/home.spec.ts", true},
		{"src/e2e/helpers.ts", true},

		// __tests__ directories
		{"src/components/__tests__/Button.test.tsx", true},

		// tests directories
		{"src/tests/setup.ts", true},

		// Cypress directories
		{"cypress/integration/login.ts", true},

		// Non-test files
		{"src/pages/Home/index.tsx", false},
		{"src/components/Navbar.tsx", false},
		{"internal/storage/sqlite.go", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := isTestFilePath(tt.path)
			if got != tt.expected {
				t.Errorf("isTestFilePath(%q) = %v, want %v", tt.path, got, tt.expected)
			}
		})
	}
}
