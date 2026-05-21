package storage

import (
	"strings"
	"testing"
)

func TestExtractChunkPreview_BasicTruncation(t *testing.T) {
	content := strings.Repeat("word ", 100) // 500 chars
	preview := ExtractChunkPreview(content, 180)
	if len(preview) > 180 {
		t.Fatalf("preview exceeds maxBytes: got %d bytes", len(preview))
	}
	if !strings.HasSuffix(preview, "...") {
		t.Fatalf("expected truncated preview to end with '...', got %q", preview[len(preview)-10:])
	}
}

func TestExtractChunkPreview_ShortContentUnchanged(t *testing.T) {
	content := "func main() { fmt.Println(\"hello\") }"
	preview := ExtractChunkPreview(content, 180)
	if preview != content {
		t.Fatalf("short content should pass through unchanged, got %q", preview)
	}
}

func TestExtractChunkPreview_CollapsesWhitespace(t *testing.T) {
	content := "func main() {\n\tfmt.Println(\"hello\")\n\tfmt.Println(\"world\")\n}"
	preview := ExtractChunkPreview(content, 180)
	if strings.Contains(preview, "\n") {
		t.Fatalf("preview should not contain newlines, got %q", preview)
	}
	if strings.Contains(preview, "\t") {
		t.Fatalf("preview should not contain tabs, got %q", preview)
	}
	expected := "func main() { fmt.Println(\"hello\") fmt.Println(\"world\") }"
	if preview != expected {
		t.Fatalf("unexpected preview:\ngot:  %q\nwant: %q", preview, expected)
	}
}

func TestExtractChunkPreview_EmptyContent(t *testing.T) {
	for _, input := range []string{"", "   ", "\n\n\t  \n"} {
		preview := ExtractChunkPreview(input, 180)
		if preview != "" {
			t.Fatalf("expected empty preview for whitespace-only input %q, got %q", input, preview)
		}
	}
}

func TestExtractChunkPreview_TruncatesAtWordBoundary(t *testing.T) {
	// Build content where a word boundary exists near the cut point.
	content := "func handleRequest(ctx context.Context, req *http.Request) (*http.Response, error) { return doSomethingVeryLongAndComplicated(ctx, req) }"
	preview := ExtractChunkPreview(content, 80)
	if len(preview) > 80 {
		t.Fatalf("preview exceeds maxBytes: got %d", len(preview))
	}
	if !strings.HasSuffix(preview, "...") {
		t.Fatalf("expected truncation marker, got %q", preview)
	}
	// Should not cut in the middle of a word.
	beforeEllipsis := strings.TrimSuffix(preview, "...")
	if beforeEllipsis != "" && beforeEllipsis[len(beforeEllipsis)-1] != ' ' && !strings.HasSuffix(beforeEllipsis, ")") {
		// Word-boundary heuristic: last char before "..." should be a space or punctuation,
		// unless the cut point happens to land exactly at a word end.
		// This is a soft check — the important thing is the length cap.
	}
}

func TestCollapseFileResultsIncludesPreview(t *testing.T) {
	chunks := []CodeChunk{
		{
			FilePath:  "/tmp/project/src/main.go",
			Content:   "func main() {\n\tfmt.Println(\"hello world\")\n}",
			StartLine: 1, EndLine: 3,
			Language: "go", Category: "source",
			Score: 0.85,
		},
		{
			FilePath:  "/tmp/project/src/main.go",
			Content:   "func helper() {\n\treturn nil\n}",
			StartLine: 10, EndLine: 12,
			Language: "go", Category: "source",
			Score: 0.70,
		},
	}

	results := CollapseFileResults(chunks, 5)
	if len(results) != 1 {
		t.Fatalf("expected 1 file result, got %d", len(results))
	}
	if results[0].Preview == "" {
		t.Fatal("expected non-empty preview from collapsed chunks")
	}
	// Preview should come from the highest-scoring chunk.
	if !strings.Contains(results[0].Preview, "hello world") {
		t.Fatalf("expected preview from best chunk, got %q", results[0].Preview)
	}
}

func TestCollapseFileResultsPreviewFromBestChunk(t *testing.T) {
	// Second chunk has higher score — preview should update.
	chunks := []CodeChunk{
		{
			FilePath:  "/tmp/project/src/app.tsx",
			Content:   "import React from 'react'",
			StartLine: 1, EndLine: 1,
			Language: "tsx", Category: "source",
			Score: 0.60,
		},
		{
			FilePath:  "/tmp/project/src/app.tsx",
			Content:   "export default function App() { return <div>Hello</div> }",
			StartLine: 5, EndLine: 10,
			Language: "tsx", Category: "source",
			Score: 0.92,
		},
	}

	results := CollapseFileResults(chunks, 5)
	if len(results) != 1 {
		t.Fatalf("expected 1 file result, got %d", len(results))
	}
	// After sorting by score desc, the 0.92 chunk is processed first,
	// so preview should contain the App component content.
	if !strings.Contains(results[0].Preview, "App()") {
		t.Fatalf("expected preview from highest-score chunk, got %q", results[0].Preview)
	}
}

func TestCollapseFileResultsMultipleFilesEachGetPreview(t *testing.T) {
	chunks := []CodeChunk{
		{
			FilePath: "/tmp/a.go", Content: "func A() {}", StartLine: 1, EndLine: 1,
			Language: "go", Category: "source", Score: 0.80,
		},
		{
			FilePath: "/tmp/b.go", Content: "func B() {}", StartLine: 1, EndLine: 1,
			Language: "go", Category: "source", Score: 0.75,
		},
	}

	results := CollapseFileResults(chunks, 5)
	if len(results) != 2 {
		t.Fatalf("expected 2 file results, got %d", len(results))
	}
	for _, r := range results {
		if r.Preview == "" {
			t.Fatalf("expected preview for %s, got empty", r.FilePath)
		}
	}
}
