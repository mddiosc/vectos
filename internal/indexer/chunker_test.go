package indexer

import (
	"strings"
	"testing"
)

type fakeEmbedder struct{}

func (f fakeEmbedder) GetEmbedding(text string) ([]float32, error) {
	return []float32{1, 2, 3}, nil
}

func (f fakeEmbedder) GetEmbeddings(texts []string) ([][]float32, error) {
	vecs := make([][]float32, len(texts))
	for i := range texts {
		vecs[i] = []float32{1, 2, 3}
	}
	return vecs, nil
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

func TestBatchEmbedChunks_FillsVectors(t *testing.T) {
	chunker := NewSimpleChunker(ChunkConfig{MaxLines: 10}, fakeEmbedder{})

	chunks := []ChunkResult{
		{SemanticText: "text a", Vector: nil},
		{SemanticText: "text b", Vector: nil},
		{SemanticText: "text c", Vector: nil},
	}

	if err := chunker.BatchEmbedChunks(chunks, 2); err != nil {
		t.Fatalf("BatchEmbedChunks failed: %v", err)
	}

	for i, c := range chunks {
		if c.Vector == nil {
			t.Errorf("chunks[%d].Vector is nil after batch embed", i)
		} else if len(c.Vector) != 3 || c.Vector[0] != 1 || c.Vector[1] != 2 || c.Vector[2] != 3 {
			t.Errorf("chunks[%d].Vector = %v, want [1 2 3]", i, c.Vector)
		}
	}
}

func TestBatchEmbedChunks_SkipsAlreadyFilled(t *testing.T) {
	chunker := NewSimpleChunker(ChunkConfig{MaxLines: 10}, fakeEmbedder{})

	existing := []float32{9, 9, 9}
	chunks := []ChunkResult{
		{SemanticText: "a", Vector: existing},
		{SemanticText: "b", Vector: nil},
	}

	if err := chunker.BatchEmbedChunks(chunks, 32); err != nil {
		t.Fatalf("BatchEmbedChunks failed: %v", err)
	}

	if chunks[0].Vector[0] != 9 || chunks[0].Vector[1] != 9 || chunks[0].Vector[2] != 9 {
		t.Errorf("existing vector was overwritten: %v", chunks[0].Vector)
	}
	if chunks[1].Vector == nil || chunks[1].Vector[0] != 1 {
		t.Errorf("nil vector was not filled: %v", chunks[1].Vector)
	}
}

func TestBatchEmbedChunks_EmptyChunks(t *testing.T) {
	chunker := NewSimpleChunker(ChunkConfig{MaxLines: 10}, fakeEmbedder{})
	if err := chunker.BatchEmbedChunks(nil, 32); err != nil {
		t.Fatalf("BatchEmbedChunks on nil failed: %v", err)
	}
	if err := chunker.BatchEmbedChunks([]ChunkResult{}, 32); err != nil {
		t.Fatalf("BatchEmbedChunks on empty failed: %v", err)
	}
}

func TestBuildSemanticContentAnnotatesChunkRole(t *testing.T) {
	semantic := buildSemanticContent("/tmp/Hero.tsx", "tsx", "export function useHeroData() {\n  return 1\n}")

	for _, expected := range []string{"File: Hero.tsx", "Purpose: custom hook", "Signature: export function useHeroData() {"} {
		if !strings.Contains(semantic, expected) {
			t.Fatalf("expected semantic content to include %q, got %q", expected, semantic)
		}
	}
	if strings.Contains(semantic, "Language:") {
		t.Fatalf("expected semantic content NOT to include Language:, got %q", semantic)
	}
}
