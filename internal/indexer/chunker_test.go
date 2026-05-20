package indexer

import (
	"fmt"
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

// --- Smart splitting tests ---

// Helper to generate a large React component for testing.
func makeLargeComponent(totalLines int) string {
	var lines []string
	lines = append(lines,
		`import { useState, useEffect, useMemo } from 'react'`,
		`import { useTranslation } from 'react-i18next'`,
		``,
		`export default function LargeComponent() {`,
		`  const [open, setOpen] = useState(false)`,
		`  const [count, setCount] = useState(0)`,
		`  const { t } = useTranslation()`,
		``,
		`  useEffect(() => {`,
		`    document.title = t('title')`,
		`    return () => { document.title = '' }`,
		`  }, [t])`,
		``,
		`  useEffect(() => {`,
		`    const handler = () => setCount(c => c + 1)`,
		`    window.addEventListener('scroll', handler)`,
		`    return () => window.removeEventListener('scroll', handler)`,
		`  }, [])`,
		``,
		`  const handleClick = () => {`,
		`    setOpen(!open)`,
		`    console.log('clicked')`,
		`  }`,
		``,
		`  const handleSubmit = (e: React.FormEvent) => {`,
		`    e.preventDefault()`,
		`    console.log('submitted')`,
		`  }`,
		``,
	)

	// Pad with handler-like code to reach desired size
	for len(lines) < totalLines-20 {
		lines = append(lines,
			`  const value`+fmt.Sprintf("%d", len(lines))+` = useMemo(() => {`,
			`    return count * `+fmt.Sprintf("%d", len(lines)),
			`  }, [count])`,
			``,
		)
	}

	lines = append(lines,
		`  return (`,
		`    <div className="container">`,
		`      <h1>{t('heading')}</h1>`,
		`      <button onClick={handleClick}>Toggle</button>`,
		`      <form onSubmit={handleSubmit}>`,
		`        <input value={count} />`,
		`      </form>`,
		`      {open && (`,
		`        <div className="panel">`,
		`          <p>{t('content')}</p>`,
		`        </div>`,
		`      )}`,
		`    </div>`,
		`  )`,
		`}`,
	)

	return strings.Join(lines, "\n")
}

func TestSplitOversizedChunk_SplitsLargeComponent(t *testing.T) {
	chunker := NewSimpleChunker(ChunkConfig{
		MaxLines:      40,
		TargetChars:   1200,
		MaxChars:      2500,
		MinChunkChars: 200,
	}, fakeEmbedder{})

	// Use 300 lines to ensure the component body exceeds 2500 chars
	component := makeLargeComponent(300)
	lines := strings.Split(component, "\n")

	chunks := chunker.chunkBraceStructuredFileImpl("LargeComponent.tsx", "tsx", lines, false)

	// Should produce multiple chunks, not just 1 giant one
	if len(chunks) < 3 {
		t.Fatalf("expected at least 3 chunks from large component, got %d (total chars: %d)", len(chunks), len(component))
	}

	// No chunk should exceed MaxChars
	for i, c := range chunks {
		if len(c.Content) > 2500 {
			t.Errorf("chunk %d exceeds MaxChars: %d chars", i, len(c.Content))
		}
	}

	// First chunk should be the prelude (imports)
	if !strings.Contains(chunks[0].Content, "import") {
		t.Errorf("expected first chunk to be prelude with imports, got: %s", chunks[0].Content[:min(100, len(chunks[0].Content))])
	}
}

func TestSplitOversizedChunk_PreservesSmallChunks(t *testing.T) {
	chunker := NewSimpleChunker(ChunkConfig{
		MaxLines:      40,
		TargetChars:   1200,
		MaxChars:      2500,
		MinChunkChars: 200,
	}, fakeEmbedder{})

	// A small component should NOT be split
	small := `export function SmallComponent() {
  const [x, setX] = useState(0)
  return <div>{x}</div>
}`
	lines := strings.Split("import React from 'react'\n\n"+small, "\n")

	chunks := chunker.chunkBraceStructuredFileImpl("Small.tsx", "tsx", lines, false)

	// Should have prelude + 1 component chunk (not split further)
	if len(chunks) != 2 {
		t.Fatalf("expected 2 chunks (prelude + component), got %d", len(chunks))
	}
}

func TestSplitOversizedChunk_SplitsAtReturn(t *testing.T) {
	chunker := NewSimpleChunker(ChunkConfig{
		MaxLines:      40,
		TargetChars:   150,
		MaxChars:      400,
		MinChunkChars: 50,
	}, fakeEmbedder{})

	component := `export function MyComponent() {
  const [state, setState] = useState(false)
  const [count, setCount] = useState(0)

  useEffect(() => {
    document.title = 'test'
    return () => { document.title = '' }
  }, [])

  const handleClick = () => {
    setState(!state)
    setCount(count + 1)
  }

  return (
    <div className="wrapper">
      <h1>Title</h1>
      <button onClick={handleClick}>Click</button>
      <span>{count}</span>
      <p>Some content here</p>
    </div>
  )
}`
	lines := strings.Split(component, "\n")
	contentLen := len(strings.Join(lines, "\n"))

	subChunks := chunker.splitOversizedChunk(lines, "tsx")

	if len(subChunks) < 2 {
		t.Fatalf("expected at least 2 sub-chunks, got %d (content: %d chars, maxChars: %d)", len(subChunks), contentLen, chunker.config.MaxChars)
	}

	// Check that the return statement creates a split boundary
	foundReturnSplit := false
	for _, sub := range subChunks {
		content := strings.Join(sub, "\n")
		if strings.HasPrefix(strings.TrimSpace(content), "return") || strings.Contains(content, "\n  return") {
			foundReturnSplit = true
		}
	}
	if !foundReturnSplit {
		t.Error("expected return statement to be at the start of a sub-chunk")
	}
}

func TestSplitOversizedChunk_SplitsAtHookWithBlock(t *testing.T) {
	chunker := NewSimpleChunker(ChunkConfig{
		MaxLines:      40,
		TargetChars:   100,
		MaxChars:      120,
		MinChunkChars: 40,
	}, fakeEmbedder{})

	component := `export function MyComponent() {
  const [state, setState] = useState(false)

  useEffect(() => {
    const a = 1
    const b = 2
    console.log(a, b)
  }, [])

  return <div>{state}</div>
}`
	lines := strings.Split(component, "\n")
	subChunks := chunker.splitOversizedChunk(lines, "tsx")

	// Expect one sub-chunk to start at the useEffect line.
	foundHookSplit := false
	for _, sub := range subChunks {
		content := strings.Join(sub, "\n")
		if strings.HasPrefix(strings.TrimSpace(content), "useEffect") {
			foundHookSplit = true
		}
	}
	if !foundHookSplit {
		t.Error("expected useEffect to be at the start of a sub-chunk")
	}
}

func TestSplitOversizedChunk_SplitsAtLatestCandidateOfSamePriority(t *testing.T) {
	chunker := NewSimpleChunker(ChunkConfig{
		MaxLines:      40,
		TargetChars:   100,
		MaxChars:      130,
		MinChunkChars: 10,
	}, fakeEmbedder{})

	lines := []string{
		"const a = 'some long string';", // 30 chars
		"",                              // blank line index 1 - total: 31 chars
		"const b = 'some long string';", // 30 chars
		"",                              // blank line index 3 - total: 62 chars
		"const c = 'some long string';", // 30 chars
		"const d = 'some long string';", // 30 chars - total: 123 chars (exceeds TargetChars=100)
		"const e = 'some long string';",
	}

	subChunks := chunker.splitOversizedChunk(lines, "tsx")

	if len(subChunks) < 2 {
		t.Fatalf("expected split, got %d chunks", len(subChunks))
	}

	// We expect the first chunk to contain up to "const b = ...", i.e. 3 lines.
	// (split at index 3).
	// If it used the first candidate (index 1), the first chunk would only have 1 line ("const a = ...").
	firstChunkLen := len(subChunks[0])
	if firstChunkLen != 3 {
		t.Errorf("expected first chunk to have 3 lines (split at blank line index 3), got %d lines: %v", firstChunkLen, subChunks[0])
	}
}

func TestSplitOversizedChunk_MergesTinyFragments(t *testing.T) {
	chunker := NewSimpleChunker(ChunkConfig{
		MaxLines:      40,
		TargetChars:   200,
		MaxChars:      400,
		MinChunkChars: 150,
	}, fakeEmbedder{})

	// Create lines where some splits would produce tiny fragments
	lines := []string{
		"const a = 1",
		"",
		"const b = 2",
		"",
		"const c = 3",
	}

	subChunks := chunker.splitOversizedChunk(lines, "tsx")

	// No fragment should be smaller than MinChunkChars (unless it's the only one)
	for i, sub := range subChunks {
		content := strings.Join(sub, "\n")
		if len(content) < 150 && len(subChunks) > 1 {
			t.Errorf("sub-chunk %d is too small (%d chars): %q", i, len(content), content)
		}
	}
}

func TestSplitOversizedChunk_SplitsAtBareJSXReturn(t *testing.T) {
	chunker := NewSimpleChunker(ChunkConfig{
		MaxLines:      40,
		TargetChars:   180,
		MaxChars:      220,
		MinChunkChars: 20,
	}, fakeEmbedder{})

	component := `export function MyComponent() {
	  doSomethingWithVeryLongNameAndExtraSuffix(state, props, context)
	  renderSomethingWithVeryLongNameAndExtraSuffix(state, props, context)
	  trackSomethingWithVeryLongNameAndExtraSuffix(state, props, context)
	  return <div>{value}{other}</div>
}`
	lines := strings.Split(component, "\n")
	subChunks := chunker.splitOversizedChunk(lines, "tsx")

	foundReturnSplit := false
	for _, sub := range subChunks {
		content := strings.Join(sub, "\n")
		if strings.HasPrefix(strings.TrimSpace(content), "return <") {
			foundReturnSplit = true
		}
	}
	if !foundReturnSplit {
		t.Error("expected bare JSX return to be at the start of a sub-chunk")
	}
}

func TestMergeTinyFragments_RespectsMaxChars(t *testing.T) {
	chunker := NewSimpleChunker(ChunkConfig{
		MaxLines:      40,
		TargetChars:   100,
		MaxChars:      20,
		MinChunkChars: 10,
	}, fakeEmbedder{})

	segments := [][]string{
		{"tiny"},
		{"123456789012345678"},
	}

	merged := chunker.mergeTinyFragments(segments)
	if len(merged) != 2 {
		t.Fatalf("expected segments to remain separate when merge would exceed MaxChars, got %d", len(merged))
	}
	for i, seg := range merged {
		if got := len(strings.Join(seg, "\n")); got > chunker.config.MaxChars {
			t.Fatalf("segment %d exceeds MaxChars after mergeTinyFragments: %d > %d", i, got, chunker.config.MaxChars)
		}
	}
}

func TestSplitOversizedChunk_NoSplitUnderMaxChars(t *testing.T) {
	chunker := NewSimpleChunker(ChunkConfig{
		MaxLines:      40,
		TargetChars:   1200,
		MaxChars:      2500,
		MinChunkChars: 200,
	}, fakeEmbedder{})

	// Content under MaxChars should not be split
	lines := []string{
		"const x = 1",
		"const y = 2",
		"return x + y",
	}

	subChunks := chunker.splitOversizedChunk(lines, "tsx")

	if len(subChunks) != 1 {
		t.Fatalf("expected 1 chunk for small content, got %d", len(subChunks))
	}
}

func TestChunkBraceStructured_GoFunctionSplitting(t *testing.T) {
	chunker := NewSimpleChunker(ChunkConfig{
		MaxLines:      40,
		TargetChars:   200,
		MaxChars:      500,
		MinChunkChars: 50,
	}, fakeEmbedder{})

	// Build a large Go function with enough content to exceed MaxChars
	var funcLines []string
	funcLines = append(funcLines, "func bigFunction() {")
	for i := 0; i < 80; i++ {
		funcLines = append(funcLines, fmt.Sprintf("\tresult%d := processItem(%d, \"some-long-argument-string-%d\")", i, i, i))
	}
	funcLines = append(funcLines, "}")

	lines := append([]string{"package main", "", "import \"fmt\"", ""}, funcLines...)

	chunks, err := chunker.chunkGoFileImpl("big.go", "go", lines, false)
	if err != nil {
		t.Fatalf("chunkGoFileImpl failed: %v", err)
	}

	// The big function should be split into multiple chunks
	funcContent := strings.Join(funcLines, "\n")
	if len(chunks) < 3 {
		t.Fatalf("expected at least 3 chunks (prelude + split function), got %d (func chars: %d)", len(chunks), len(funcContent))
	}

	// No chunk should exceed MaxChars
	for i, c := range chunks {
		if len(c.Content) > 500 {
			t.Errorf("chunk %d exceeds MaxChars: %d chars", i, len(c.Content))
		}
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
