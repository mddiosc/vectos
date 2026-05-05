package main

import (
	"path/filepath"
	"testing"

	"vectos/internal/storage"
)

func TestPrepareStoreForIndexingClearsChunksOnFullReindex(t *testing.T) {
	pm, err := storage.NewProjectManager(t.TempDir())
	if err != nil {
		t.Fatalf("new project manager: %v", err)
	}
	store, err := storage.NewSQLiteStorageForProjectName(pm, "vectos-test")
	if err != nil {
		t.Fatalf("new sqlite storage: %v", err)
	}
	defer store.Close()

	if _, err := store.SaveChunk(storage.CodeChunk{
		FilePath:  filepath.Join(t.TempDir(), "stale.md"),
		Content:   "stale docs chunk",
		StartLine: 1,
		EndLine:   1,
		Language:  "markdown",
		Category:  "docs",
	}); err != nil {
		t.Fatalf("save chunk: %v", err)
	}

	if err := prepareStoreForIndexing(store, nil); err != nil {
		t.Fatalf("prepareStoreForIndexing returned error: %v", err)
	}

	stats, err := store.Stats()
	if err != nil {
		t.Fatalf("stats: %v", err)
	}
	if stats.ChunkCount != 0 {
		t.Fatalf("expected empty index after full reindex preparation, got %d chunks", stats.ChunkCount)
	}
}

func TestPrepareStoreForIndexingKeepsChunksForIncrementalReindex(t *testing.T) {
	pm, err := storage.NewProjectManager(t.TempDir())
	if err != nil {
		t.Fatalf("new project manager: %v", err)
	}
	store, err := storage.NewSQLiteStorageForProjectName(pm, "vectos-test")
	if err != nil {
		t.Fatalf("new sqlite storage: %v", err)
	}
	defer store.Close()

	if _, err := store.SaveChunk(storage.CodeChunk{
		FilePath:  filepath.Join(t.TempDir(), "keep.md"),
		Content:   "keep docs chunk",
		StartLine: 1,
		EndLine:   1,
		Language:  "markdown",
		Category:  "docs",
	}); err != nil {
		t.Fatalf("save chunk: %v", err)
	}

	if err := prepareStoreForIndexing(store, []string{"README.md"}); err != nil {
		t.Fatalf("prepareStoreForIndexing returned error: %v", err)
	}

	stats, err := store.Stats()
	if err != nil {
		t.Fatalf("stats: %v", err)
	}
	if stats.ChunkCount != 1 {
		t.Fatalf("expected incremental preparation to preserve existing chunks, got %d", stats.ChunkCount)
	}
}
