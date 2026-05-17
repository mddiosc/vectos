package indexer

import (
	"path/filepath"
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

	chunks := chunker.chunkStructuredFileImpl("Hero.tsx", "tsx", lines, true)
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

func TestChunkFileRawWrapsReadErrors(t *testing.T) {
	chunker := NewSimpleChunker(ChunkConfig{MaxLines: 10}, fakeEmbedder{})
	missing := filepath.Join(t.TempDir(), "missing.ts")

	_, err := chunker.ChunkFileRaw(missing, "typescript")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "cannot read file "+missing) {
		t.Fatalf("unexpected error: %v", err)
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

// 3.1: interface chunk receives "type definition" tag
func TestInferPurpose_InterfaceChunk(t *testing.T) {
	purpose := inferNonGoPurpose("typescript", "interface UserProps {\n  name: string\n  email: string\n}")
	if !strings.Contains(purpose, "type definition") {
		t.Fatalf("expected 'type definition' in purpose, got %q", purpose)
	}
}

// 3.2: type alias chunk receives "type definition" tag
func TestInferPurpose_TypeAliasChunk(t *testing.T) {
	purpose := inferNonGoPurpose("typescript", "type UserRole = \"admin\" | \"editor\" | \"viewer\"")
	if !strings.Contains(purpose, "type definition") {
		t.Fatalf("expected 'type definition' in purpose, got %q", purpose)
	}
}

// 3.3: enum chunk receives "enumeration" tag
func TestInferPurpose_EnumChunk(t *testing.T) {
	purpose := inferNonGoPurpose("typescript", "enum Status {\n  Active,\n  Inactive,\n  Pending\n}")
	if !strings.Contains(purpose, "enumeration") {
		t.Fatalf("expected 'enumeration' in purpose, got %q", purpose)
	}
}

// 3.4: async function chunk receives "async function" tag
func TestInferPurpose_AsyncFunction(t *testing.T) {
	purpose := inferNonGoPurpose("typescript", "async function fetchUserData(id: string): Promise<User> {\n  return await fetch(`/api/users/${id}`)\n}")
	if !strings.Contains(purpose, "async function") {
		t.Fatalf("expected 'async function' in purpose, got %q", purpose)
	}
}

// 3.5: chunk with both component and async gets both tags (tags are additive)
func TestInferPurpose_ComponentAndAsync(t *testing.T) {
	// Use a component that contains an async arrow function, so both detections fire
	purpose := inferNonGoPurpose("tsx", "export function UserProfile({ id }: Props) {\n  const fetch = async () => { return await api.get(id) }\n  return <div>Profile</div>\n}")
	if !strings.Contains(purpose, "react component") {
		t.Fatalf("expected 'react component' in purpose, got %q", purpose)
	}
	if !strings.Contains(purpose, "async function") {
		t.Fatalf("expected 'async function' in purpose, got %q", purpose)
	}
}

// 3.6: regular function without async does NOT get "async function" tag
func TestInferPurpose_RegularFunctionNoAsync(t *testing.T) {
	purpose := inferNonGoPurpose("typescript", "function formatDate(date: Date): string {\n  return date.toISOString()\n}")
	if strings.Contains(purpose, "async function") {
		t.Fatalf("expected NO 'async function' tag for regular function, got %q", purpose)
	}
	if !strings.Contains(purpose, "function or callable block") {
		t.Fatalf("expected 'function or callable block' in purpose, got %q", purpose)
	}
}

// 3.7: export interface gets both "type definition" and "exported api"
func TestInferPurpose_ExportInterface(t *testing.T) {
	purpose := inferNonGoPurpose("typescript", "export interface ApiResponse<T> {\n  data: T\n  error?: string\n}")
	if !strings.Contains(purpose, "type definition") {
		t.Fatalf("expected 'type definition' in purpose, got %q", purpose)
	}
	if !strings.Contains(purpose, "exported api") {
		t.Fatalf("expected 'exported api' in purpose, got %q", purpose)
	}
}

// 3.8: interface/enum are recognized as structural boundaries in isStructuredBoundary()
func TestIsStructuredBoundary_TypescriptStructures(t *testing.T) {
	tests := []struct {
		line     string
		language string
		want     bool
	}{
		{"interface Foo {}", "typescript", true},
		{"export interface Bar<T> {", "typescript", true},
		{"enum Colors {", "typescript", true},
		{"const enum Direction {", "typescript", true},
		{"export enum Status {", "typescript", true},
		{"interface Foo {}", "javascript", true},
		{"enum Colors {", "javascript", true},
		{"type Alias = string", "typescript", false},
		{"import { Foo } from './foo'", "typescript", false},
		{"const x = 1;", "typescript", false},
	}

	for _, tt := range tests {
		got := isStructuredBoundary(tt.language, tt.line)
		if got != tt.want {
			t.Errorf("isStructuredBoundary(%q, %q) = %v, want %v", tt.language, tt.line, got, tt.want)
		}
	}
}

// 3.9: table-driven test for all new patterns (valid and invalid inputs)
func TestInferPurpose_TableDriven(t *testing.T) {
	tests := []struct {
		name     string
		language string
		content  string
		contains []string
		excludes []string
	}{
		{
			name:     "interface gets type definition",
			language: "typescript",
			content:  "interface UserProps {\n  name: string\n}",
			contains: []string{"type definition"},
			excludes: []string{"enumeration", "async function"},
		},
		{
			name:     "exported interface gets type definition and exported api",
			language: "typescript",
			content:  "export interface ApiResponse<T> {\n  data: T\n}",
			contains: []string{"type definition", "exported api"},
		},
		{
			name:     "type alias gets type definition",
			language: "typescript",
			content:  "type UserRole = \"admin\" | \"editor\"",
			contains: []string{"type definition"},
			excludes: []string{"enumeration", "async function"},
		},
		{
			name:     "generic type alias gets type definition",
			language: "typescript",
			content:  "type ApiResponse<T> = { data: T; error?: string }",
			contains: []string{"type definition"},
		},
		{
			name:     "enum gets enumeration",
			language: "typescript",
			content:  "enum Status {\n  Active,\n  Inactive\n}",
			contains: []string{"enumeration"},
			excludes: []string{"type definition", "async function"},
		},
		{
			name:     "const enum gets enumeration",
			language: "typescript",
			content:  "const enum Direction {\n  Up,\n  Down\n}",
			contains: []string{"enumeration"},
		},
		{
			name:     "exported enum gets enumeration and exported api",
			language: "typescript",
			content:  "export enum LogLevel {\n  Debug,\n  Error\n}",
			contains: []string{"enumeration", "exported api"},
		},
		{
			name:     "async function gets async function tag",
			language: "typescript",
			content:  "async function fetchData(url: string) {\n  return await fetch(url)\n}",
			contains: []string{"async function"},
		},
		{
			name:     "async arrow gets async function tag",
			language: "typescript",
			content:  "const fetchData = async (url: string) => {\n  return await fetch(url)\n}",
			contains: []string{"async function"},
		},
		{
			name:     "export async gets async function and exported api",
			language: "typescript",
			content:  "export async function loadData() {\n  return await fetch('/api')\n}",
			contains: []string{"async function", "exported api"},
		},
		{
			name:     "regular function without async does NOT get async function",
			language: "typescript",
			content:  "function formatDate(date: Date): string {\n  return date.toISOString()\n}",
			contains: []string{"function or callable block"},
			excludes: []string{"async function"},
		},
		{
			name:     "non-JS/TS language ignores new patterns",
			language: "python",
			content:  "interface = {}\ndef foo():\n  pass",
			contains: []string{"function or callable block"},
			excludes: []string{"type definition", "enumeration", "async function"},
		},
		{
			name:     "non-JS/TS language with enum-like content does not get enumeration",
			language: "python",
			content:  "from enum import Enum\nclass Color(Enum):\n  RED = 1",
			contains: []string{"class definition"},
			excludes: []string{"enumeration"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			purpose := inferNonGoPurpose(tt.language, tt.content)
			for _, c := range tt.contains {
				if !strings.Contains(purpose, c) {
					t.Errorf("purpose %q should contain %q", purpose, c)
				}
			}
			for _, e := range tt.excludes {
				if strings.Contains(purpose, e) {
					t.Errorf("purpose %q should NOT contain %q", purpose, e)
				}
			}
		})
	}
}
