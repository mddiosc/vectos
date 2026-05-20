package main

import (
	"crypto/sha256"
	"os"
	"path/filepath"
	"testing"

	"vectos/internal/embeddings"
	"vectos/internal/storage"
	"vectos/internal/vectorindex"
)

func newIndexMetadataTestStore(t *testing.T) *storage.SQLiteStorage {
	t.Helper()
	pm, err := storage.NewProjectManager(t.TempDir())
	if err != nil {
		t.Fatalf("new project manager: %v", err)
	}
	store, err := storage.NewSQLiteStorageForProjectName(pm, "vectos-test")
	if err != nil {
		t.Fatalf("new sqlite storage: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store
}

func TestSyncIndexMetadataInvalidatesStaleEmbeddings(t *testing.T) {
	store := newIndexMetadataTestStore(t)

	if _, err := store.SaveChunk(storage.CodeChunk{
		FilePath:  filepath.Join(t.TempDir(), "stale.go"),
		Content:   "stale chunk",
		StartLine: 1,
		EndLine:   1,
		Language:  "go",
		Vector:    []float32{1, 0, 0, 0},
	}); err != nil {
		t.Fatalf("save chunk: %v", err)
	}
	if err := store.UpsertIndexedFile("stale.go", "hash-1"); err != nil {
		t.Fatalf("upsert indexed file: %v", err)
	}
	if err := store.SetIndexMetadata(storage.IndexMetadata{Provider: "embedded", Model: "bge-small-en-v1.5", Dimensions: 384}); err != nil {
		t.Fatalf("set initial metadata: %v", err)
	}

	idx := vectorindex.NewHNSW(4, vectorindex.Config{})
	if err := idx.Insert(1, []float32{1, 0, 0, 0}); err != nil {
		t.Fatalf("insert vector index node: %v", err)
	}
	if err := idx.Save(store.VectorIndexPath(), sha256.Sum256([]byte("hash")), "none", nil); err != nil {
		t.Fatalf("save vector index: %v", err)
	}

	fingerprint := currentIndexFingerprint(indexChunkerConfig(0))
	message, err := syncIndexMetadata(store, embeddings.ProviderInfo{Provider: "embedded", Model: "jina-embeddings-v3", Dimensions: 512}, fingerprint)
	if err != nil {
		t.Fatalf("syncIndexMetadata: %v", err)
	}
	if message.Message == "" {
		t.Fatal("expected invalidation message")
	}
	if _, err := os.Stat(store.VectorIndexPath()); !os.IsNotExist(err) {
		t.Fatalf("expected stale vector index to be removed, got err=%v", err)
	}

	stats, err := store.Stats()
	if err != nil {
		t.Fatalf("stats: %v", err)
	}
	if stats.EmbeddedCount != 0 {
		t.Fatalf("expected embeddings to be cleared, got %d embedded chunks", stats.EmbeddedCount)
	}

	meta, err := store.GetIndexMetadata()
	if err != nil {
		t.Fatalf("get index metadata: %v", err)
	}
	if meta.Model != "jina-embeddings-v3" || meta.Dimensions != 512 || meta.IndexFingerprint != fingerprint {
		t.Fatalf("unexpected metadata after sync: %+v", meta)
	}
}

func TestSyncIndexMetadataLegacyFingerprintForcesFullRebuild(t *testing.T) {
	store := newIndexMetadataTestStore(t)

	if _, err := store.SaveChunk(storage.CodeChunk{FilePath: "legacy.go", Content: "legacy chunk", StartLine: 1, EndLine: 1, Language: "go", Vector: []float32{1, 0, 0, 0}}); err != nil {
		t.Fatalf("save chunk: %v", err)
	}
	if err := store.UpsertIndexedFile("legacy.go", "hash-legacy"); err != nil {
		t.Fatalf("upsert indexed file: %v", err)
	}
	if err := store.SetIndexMetadata(storage.IndexMetadata{Provider: "embedded", Model: "jina-embeddings-v3", Dimensions: 512}); err != nil {
		t.Fatalf("set legacy metadata: %v", err)
	}

	fingerprint := currentIndexFingerprint(indexChunkerConfig(0))
	result, err := syncIndexMetadata(store, embeddings.ProviderInfo{Provider: "embedded", Model: "jina-embeddings-v3", Dimensions: 512}, fingerprint)
	if err != nil {
		t.Fatalf("syncIndexMetadata legacy: %v", err)
	}
	if !result.FullRebuild {
		t.Fatal("expected legacy fingerprint to force full rebuild")
	}

	stats, err := store.Stats()
	if err != nil {
		t.Fatalf("stats: %v", err)
	}
	if stats.ChunkCount != 0 || stats.FileCount != 0 {
		t.Fatalf("expected cleared index after rebuild trigger, got chunks=%d files=%d", stats.ChunkCount, stats.FileCount)
	}

	meta, err := store.GetIndexMetadata()
	if err != nil {
		t.Fatalf("get index metadata: %v", err)
	}
	if meta.IndexFingerprint != fingerprint {
		t.Fatalf("expected fingerprint %q, got %q", fingerprint, meta.IndexFingerprint)
	}
}
