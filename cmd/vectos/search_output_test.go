package main

import (
	"strings"
	"testing"

	"vectos/internal/storage"
)

func TestFormatSearchResultsUsesCompactSnippetsByDefault(t *testing.T) {
	content := strings.Repeat("line content that should be truncated ", 8)
	output := formatSearchResults("structured chunking tsx", []storage.CodeChunk{{
		FilePath:  "/tmp/project/internal/storage/sqlite.go",
		StartLine: 191,
		EndLine:   214,
		Language:  "go",
		Category:  "source",
		Score:     0.92,
		Content:   content,
	}}, "semantic_hybrid", false)

	if strings.Contains(output, content) {
		t.Fatalf("expected compact output, got %q", output)
	}
	if !strings.Contains(output, "reason:") {
		t.Fatalf("expected relevance reason in compact output, got %q", output)
	}
}

func TestFormatSearchResultsCanPrintFullContent(t *testing.T) {
	output := formatSearchResults("structured chunking tsx", []storage.CodeChunk{{
		FilePath: "/tmp/project/internal/storage/sqlite.go",
		Content:  "line 1\nline 2",
	}}, "semantic_hybrid", true)

	if !strings.Contains(output, "line 2") {
		t.Fatalf("expected full output, got %q", output)
	}
}

func TestAdaptivePreviewLimitShortensHighConfidenceQueries(t *testing.T) {
	limit := adaptivePreviewLimit("fallback text search", []storage.CodeChunk{{Score: 0.92}, {Score: 0.70}})
	if limit != searchPreviewShort {
		t.Fatalf("expected short preview for high confidence query, got %d", limit)
	}
}

func TestAdaptivePreviewLimitExpandsAmbiguousLongQueries(t *testing.T) {
	limit := adaptivePreviewLimit("how does Nx scope resolution expand logical roots from dependencies", []storage.CodeChunk{{Score: 0.71}, {Score: 0.70}})
	if limit != searchPreviewLong {
		t.Fatalf("expected long preview for ambiguous long query, got %d", limit)
	}
}
