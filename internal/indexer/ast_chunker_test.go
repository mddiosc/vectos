package indexer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTreeSitterChunksTsxComponent(t *testing.T) {
	chunker := NewSimpleChunker(ChunkConfig{
		MaxLines:      10,
		TargetChars:   1200,
		MaxChars:      2500,
		MinChunkChars: 200,
		BatchSize:     5,
	}, nil)

	dir := t.TempDir()
	path := filepath.Join(dir, "Navbar.tsx")
	code := `import React from 'react'
import { useTranslation } from 'react-i18next'

export default function Navbar() {
  const [open, setOpen] = useState(false)

  useEffect(() => {
    window.addEventListener('scroll', handleScroll)
  }, [])

  return (
    <nav>
      <button onClick={() => setOpen(!open)}>Menu</button>
    </nav>
  )
}`

	if err := os.WriteFile(path, []byte(code), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	results, err := chunker.ChunkFileRaw(path, "tsx")
	if err != nil {
		t.Fatalf("ChunkFileRaw: %v", err)
	}

	// Should produce at least 2 chunks: import prelude + component.
	if len(results) < 2 {
		t.Fatalf("expected at least 2 chunks, got %d", len(results))
	}

	// The import prelude should contain import statements.
	foundImport := false
	foundComponent := false
	for _, r := range results {
		if strings.Contains(r.Content, "import React from 'react'") {
			foundImport = true
		}
		if strings.Contains(r.Content, "export default function Navbar") {
			foundComponent = true
		}
		// All chunks should be under MaxChars.
		if len(r.Content) > 2500 {
			t.Errorf("chunk exceeds MaxChars: %d bytes", len(r.Content))
		}
	}
	if !foundImport {
		t.Error("expected an import prelude chunk")
	}
	if !foundComponent {
		t.Error("expected a component chunk with export default function")
	}
}

func TestTreeSitterChunksPythonDefs(t *testing.T) {
	chunker := NewSimpleChunker(ChunkConfig{
		MaxLines:      10,
		TargetChars:   1200,
		MaxChars:      2500,
		MinChunkChars: 200,
		BatchSize:     5,
	}, nil)

	dir := t.TempDir()
	path := filepath.Join(dir, "auth.py")
	code := `import os
from typing import Optional

def authenticate(token: str) -> bool:
    if not token:
        raise ValueError("missing token")
    return True

def refresh_token(token: str) -> str:
    return token + "_refreshed"
`

	if err := os.WriteFile(path, []byte(code), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	results, err := chunker.ChunkFileRaw(path, "python")
	if err != nil {
		t.Fatalf("ChunkFileRaw: %v", err)
	}

	if len(results) < 2 {
		t.Fatalf("expected at least 2 chunks, got %d", len(results))
	}

	foundAuth := false
	foundRefresh := false
	for _, r := range results {
		if strings.Contains(r.Content, "def authenticate") {
			foundAuth = true
		}
		if strings.Contains(r.Content, "def refresh_token") {
			foundRefresh = true
		}
	}
	if !foundAuth {
		t.Error("expected authenticate function chunk")
	}
	if !foundRefresh {
		t.Error("expected refresh_token function chunk")
	}
}

func TestTreeSitterChunksGoFunctions(t *testing.T) {
	chunker := NewSimpleChunker(ChunkConfig{
		MaxLines:      10,
		TargetChars:   1200,
		MaxChars:      2500,
		MinChunkChars: 200,
		BatchSize:     5,
	}, nil)

	dir := t.TempDir()
	path := filepath.Join(dir, "main.go")
	code := `package main

import "fmt"

func main() {
	fmt.Println("hello")
}

func helper() string {
	return "helper"
}
`

	if err := os.WriteFile(path, []byte(code), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	results, err := chunker.ChunkFileRaw(path, "go")
	if err != nil {
		t.Fatalf("ChunkFileRaw: %v", err)
	}

	if len(results) < 2 {
		t.Fatalf("expected at least 2 chunks (main + helper), got %d", len(results))
	}

	for _, r := range results {
		if len(r.Content) > 2500 {
			t.Errorf("chunk exceeds MaxChars: %d bytes", len(r.Content))
		}
	}
}

func TestTreeSitterFallbackForUnsupportedLanguage(t *testing.T) {
	// JSON doesn't have tree-sitter support in our list — should fall back to line chunking.
	chunker := NewSimpleChunker(ChunkConfig{
		MaxLines:     5,
		BatchSize:    5,
		TargetChars:  1200,
		MaxChars:     2500,
	}, nil)

	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	code := `{
  "name": "test",
  "version": "1.0.0",
  "dependencies": {
    "react": "^19.0.0",
    "typescript": "^5.0.0"
  }
}`

	if err := os.WriteFile(path, []byte(code), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	results, err := chunker.ChunkFileRaw(path, "json")
	if err != nil {
		t.Fatalf("ChunkFileRaw: %v", err)
	}

	// JSON falls back to line chunking — should produce at least 1 chunk.
	if len(results) == 0 {
		t.Fatal("expected at least 1 chunk for JSON (line-chunked fallback)")
	}
}

func TestSupportsTreeSitter(t *testing.T) {
	supported := []string{"tsx", "typescript", "javascript", "jsx", "go", "python", "java", "shell"}
	for _, lang := range supported {
		if !supportsTreeSitter(lang) {
			t.Errorf("expected supportsTreeSitter(%q) = true", lang)
		}
	}
	unsupported := []string{"json", "yaml", "markdown", "dockerfile", "unknown"}
	for _, lang := range unsupported {
		if supportsTreeSitter(lang) {
			t.Errorf("expected supportsTreeSitter(%q) = false", lang)
		}
	}
}
