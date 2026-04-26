package main

import (
	"testing"

	"vectos/internal/storage"
)

func TestValidateBenchmarkFileRejectsMissingExpectedTargets(t *testing.T) {
	err := validateBenchmarkFile(retrievalBenchmarkFile{
		Queries: []retrievalBenchmarkQuery{{Name: "q1", Query: "auth flow"}},
	})
	if err == nil {
		t.Fatal("expected validation error for missing expected targets")
	}
}

func TestValidateBenchmarkFileAcceptsExpectedFilesAndChunks(t *testing.T) {
	err := validateBenchmarkFile(retrievalBenchmarkFile{
		Queries: []retrievalBenchmarkQuery{{
			Name:          "q1",
			Query:         "auth flow",
			ExpectedFiles: []string{"cmd/vectos/main.go"},
			ExpectedChunks: []retrievalExpectedChunk{{
				File:      "internal/storage/sqlite.go",
				StartLine: 191,
				EndLine:   214,
			}},
		}},
	})
	if err != nil {
		t.Fatalf("expected valid benchmark file, got %v", err)
	}
}

func TestEvaluateTopHitsMatchesExpectedFileAndChunk(t *testing.T) {
	query := retrievalBenchmarkQuery{
		Name:          "q1",
		Query:         "search text fallback",
		ExpectedFiles: []string{"cmd/vectos/main.go"},
		ExpectedChunks: []retrievalExpectedChunk{{
			File:      "internal/storage/sqlite.go",
			StartLine: 191,
			EndLine:   214,
		}},
	}
	results := []storage.CodeChunk{
		{FilePath: "/tmp/project/other.go", StartLine: 1, EndLine: 10},
		{FilePath: "/tmp/project/internal/storage/sqlite.go", StartLine: 200, EndLine: 220},
		{FilePath: "/tmp/project/cmd/vectos/main.go", StartLine: 376, EndLine: 425},
	}

	hits := evaluateTopHits(query, results, []int{1, 2, 3})
	if hits[1] {
		t.Fatal("did not expect a top-1 hit")
	}
	if !hits[2] {
		t.Fatal("expected a top-2 hit from expected chunk overlap")
	}
	if !hits[3] {
		t.Fatal("expected a top-3 hit")
	}
}
