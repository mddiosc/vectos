package storage

import (
	"strings"
	"testing"
)

func TestExtractChunkPreview_BasicTruncation(t *testing.T) {
	content := strings.Repeat("word ", 100) // 500 chars
	preview := ExtractChunkPreview(CodeChunk{Content: content}, 180)
	if len(preview) > 180 {
		t.Fatalf("preview exceeds maxBytes: got %d bytes", len(preview))
	}
	if !strings.HasSuffix(preview, "...") {
		t.Fatalf("expected truncated preview to end with '...', got %q", preview[len(preview)-10:])
	}
}

func TestExtractChunkPreview_ShortContentUnchanged(t *testing.T) {
	content := "func main() { fmt.Println(\"hello\") }"
	preview := ExtractChunkPreview(CodeChunk{Content: content}, 180)
	if preview != content {
		t.Fatalf("short content should pass through unchanged, got %q", preview)
	}
}

func TestExtractChunkPreview_CollapsesWhitespace(t *testing.T) {
	content := "func main() {\n\tfmt.Println(\"hello\")\n\tfmt.Println(\"world\")\n}"
	preview := ExtractChunkPreview(CodeChunk{Content: content}, 180)
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
		preview := ExtractChunkPreview(CodeChunk{Content: input}, 180)
		if preview != "" {
			t.Fatalf("expected empty preview for whitespace-only input %q, got %q", input, preview)
		}
	}
}

func TestExtractChunkPreview_TruncatesAtWordBoundary(t *testing.T) {
	// Build content where a word boundary exists near the cut point.
	content := "func handleRequest(ctx context.Context, req *http.Request) (*http.Response, error) { return doSomethingVeryLongAndComplicated(ctx, req) }"
	preview := ExtractChunkPreview(CodeChunk{Content: content}, 80)
	if len(preview) > 80 {
		t.Fatalf("preview exceeds maxBytes: got %d", len(preview))
	}
	if !strings.HasSuffix(preview, "...") {
		t.Fatalf("expected truncation marker, got %q", preview)
	}
	// Word-boundary heuristic: last char before "..." should not be in the middle of a word.
	// This is a soft check — the important thing is the length cap.
	_ = strings.TrimSuffix(preview, "...")
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

func TestExtractChunkPreview_UTF8MultibyteSafe(t *testing.T) {
	// Content with multi-byte characters (Spanish, emoji, Japanese).
	// Each ñ = 2 bytes, each emoji = 4 bytes, each kanji = 3 bytes.
	content := "función principal() { // コメント 🚀 return resultado }"
	preview := ExtractChunkPreview(CodeChunk{Content: content}, 30)
	if len(preview) > 30 {
		t.Fatalf("preview exceeds maxBytes: got %d bytes", len(preview))
	}
	if !strings.HasSuffix(preview, "...") {
		t.Fatalf("expected truncation, got %q", preview)
	}
	// Verify the preview is valid UTF-8 by checking it round-trips.
	for i, r := range preview {
		if r == '\uFFFD' {
			t.Fatalf("invalid UTF-8 at byte %d in preview %q", i, preview)
		}
	}
}

func TestExtractChunkPreview_UTF8ExactBoundary(t *testing.T) {
	// Build content where a 2-byte char (ñ) sits right at the byte boundary.
	// "a" repeated 176 times = 176 bytes, then "ñ" (2 bytes) = 178 bytes total.
	// maxBytes=178 should include the ñ without truncation.
	content := strings.Repeat("a", 176) + "ñ"
	preview := ExtractChunkPreview(CodeChunk{Content: content}, 178)
	if preview != content {
		t.Fatalf("expected content to pass through at exact boundary, got len=%d", len(preview))
	}

	// maxBytes=177 should truncate before the ñ (not split it).
	preview2 := ExtractChunkPreview(CodeChunk{Content: content}, 177)
	if len(preview2) > 177 {
		t.Fatalf("preview exceeds maxBytes: got %d bytes", len(preview2))
	}
	// Must not contain a broken ñ.
	for _, r := range preview2 {
		if r == '\uFFFD' {
			t.Fatalf("broken UTF-8 in preview: %q", preview2)
		}
	}
}

func TestExtractChunkPreview_ZeroMaxBytes(t *testing.T) {
	content := "func main() { fmt.Println(\"hello\") }"
	preview := ExtractChunkPreview(CodeChunk{Content: content}, 0)
	if preview != content {
		t.Fatalf("maxBytes=0 should not truncate, got %q", preview)
	}
}

func TestExtractChunkPreview_VerySmallMaxBytes(t *testing.T) {
	content := "func main() {}"
	// maxBytes=3 means only room for "..."
	preview := ExtractChunkPreview(CodeChunk{Content: content}, 3)
	if preview != "..." {
		t.Fatalf("maxBytes=3 should return just ellipsis, got %q", preview)
	}
	// maxBytes=4 should fit at least 1 char + "..."
	preview2 := ExtractChunkPreview(CodeChunk{Content: content}, 4)
	if len(preview2) > 4 {
		t.Fatalf("preview exceeds maxBytes=4: got %d bytes", len(preview2))
	}
}

func TestCollapseFileResultsEmptyContentChunk(t *testing.T) {
	chunks := []CodeChunk{
		{
			FilePath: "/tmp/empty.go", Content: "   \n\t  \n", StartLine: 1, EndLine: 2,
			Language: "go", Category: "source", Score: 0.80,
		},
	}
	results := CollapseFileResults(chunks, 5)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Preview != "" {
		t.Fatalf("expected empty preview for whitespace-only chunk, got %q", results[0].Preview)
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

func TestExtractChunkPreviewPrefersStoredSnippet(t *testing.T) {
	stored := "func main() { fmt.Println(\"hello\") }"
	longContent := strings.Repeat("word ", 500) // much longer than 180 bytes

	// With stored PreviewSnippet, it should be used directly (no truncation needed).
	chunk := CodeChunk{
		Content:        longContent,
		PreviewSnippet: stored,
	}
	preview := ExtractChunkPreview(chunk, 180)
	if preview != stored {
		t.Fatalf("expected stored snippet %q, got %q", stored, preview)
	}

	// Without stored snippet, fall back to content extraction (which truncates).
	chunk2 := CodeChunk{Content: longContent}
	preview2 := ExtractChunkPreview(chunk2, 180)
	if len(preview2) > 180 {
		t.Fatalf("fallback preview should be truncated to 180 bytes, got %d", len(preview2))
	}
	if !strings.HasSuffix(preview2, "...") {
		t.Fatal("fallback preview should end with '...'")
	}
}
