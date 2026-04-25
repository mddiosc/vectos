package indexer

import (
	"strings"
	"testing"
)

type fakeEmbedder struct{}

func (f fakeEmbedder) GetEmbedding(text string) ([]float32, error) {
	return []float32{1, 2, 3}, nil
}

func TestChunkStructuredTSXSeparatesPreludeAndBlocks(t *testing.T) {
	chunker := NewSimpleChunker(ChunkConfig{MaxLines: 40, MinLines: 5}, fakeEmbedder{})
	lines := strings.Split(`import { useMemo } from "react"
import { Button } from "./Button"

export function Hero() {
	return <Button />
}

export function useHeroData() {
	return useMemo(() => ({ title: "hi" }), [])
}

test("works", () => {
	expect(true).toBe(true)
})`, "\n")

	chunks := chunker.chunkStructuredFile("Hero.tsx", "tsx", lines)
	if len(chunks) < 4 {
		t.Fatalf("expected at least 4 chunks, got %d", len(chunks))
	}

	if !strings.Contains(chunks[0].Content, "import { useMemo }") {
		t.Fatalf("expected prelude chunk first, got %q", chunks[0].Content)
	}

	if !strings.Contains(chunks[1].Content, "export function Hero") {
		t.Fatalf("expected component chunk, got %q", chunks[1].Content)
	}

	if !strings.Contains(chunks[2].Content, "export function useHeroData") {
		t.Fatalf("expected hook chunk, got %q", chunks[2].Content)
	}

	if !strings.Contains(chunks[3].Content, "test(\"works\"") {
		t.Fatalf("expected test chunk, got %q", chunks[3].Content)
	}
}

func TestBuildSemanticContentAnnotatesChunkRole(t *testing.T) {
	semantic := buildSemanticContent("/tmp/Hero.tsx", "tsx", "export function useHeroData() {\n  return 1\n}")

	for _, expected := range []string{"Language: tsx", "Purpose: custom hook", "Signature: export function useHeroData() {"} {
		if !strings.Contains(semantic, expected) {
			t.Fatalf("expected semantic content to include %q, got %q", expected, semantic)
		}
	}
}
