package storage

import (
	"fmt"
	"math/rand/v2"
	"path/filepath"
	"testing"

	"vectos/internal/vectorindex"
)

func newTestSQLiteStorage(t *testing.T) (*SQLiteStorage, func()) {
	t.Helper()

	baseDir := t.TempDir()
	pm, err := NewProjectManager(baseDir)
	if err != nil {
		t.Fatalf("NewProjectManager: %v", err)
	}

	projectDir := filepath.Join(baseDir, "project")
	storage, err := NewSQLiteStorageForProject(pm, projectDir)
	if err != nil {
		t.Fatalf("NewSQLiteStorageForProject: %v", err)
	}

	cleanup := func() { _ = storage.Close() }
	return storage, cleanup
}

func randomVector(dim int) []float32 {
	v := make([]float32, dim)
	for i := range v {
		v[i] = rand.Float32()
	}
	return v
}

func TestVectorSearchEndToEnd(t *testing.T) {
	store, cleanup := newTestSQLiteStorage(t)
	defer cleanup()

	const dim = 384
	const n = 100
	vectors := make([][]float32, n)
	for i := 0; i < n; i++ {
		v := randomVector(dim)
		vectors[i] = v
		if _, err := store.SaveChunk(CodeChunk{
			FilePath:  fmt.Sprintf("file-%d.go", i),
			Content:   fmt.Sprintf("chunk %d", i),
			StartLine: 1,
			EndLine:   2,
			Language:  "go",
			Vector:    v,
		}); err != nil {
			t.Fatalf("SaveChunk(%d): %v", i, err)
		}
	}

	idx := vectorindex.NewHNSW(dim, vectorindex.Config{M: 16, EfConstruction: 200, EfSearch: 50})
	embeddings, err := store.GetAllEmbeddings()
	if err != nil {
		t.Fatalf("GetAllEmbeddings: %v", err)
	}
	for id, vec := range embeddings {
		idx.Insert(id, vec)
	}

	hash, err := store.ChunkTableContentHash()
	if err != nil {
		t.Fatalf("ChunkTableContentHash: %v", err)
	}
	if err := idx.Save(store.VectorIndexPath(), hash, "none", nil); err != nil {
		t.Fatalf("Save index: %v", err)
	}

	loaded, loadedHash, _, _, err := store.LoadVectorIndex()
	if err != nil {
		t.Fatalf("LoadVectorIndex: %v", err)
	}
	if loadedHash != hash {
		t.Fatalf("loaded hash mismatch")
	}
	if !store.HasVectorIndex() {
		t.Fatal("expected vector index to be loaded")
	}

	results, err := store.SearchSemantic(vectors[0], 5, true)
	if err != nil {
		t.Fatalf("SearchSemantic: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected search results")
	}
	_ = loaded
}

func TestVectorSearchFallbackWhenNoIndex(t *testing.T) {
	store, cleanup := newTestSQLiteStorage(t)
	defer cleanup()

	query := randomVector(384)
	if _, err := store.SaveChunk(CodeChunk{
		FilePath:  "fallback.go",
		Content:   "semantic fallback",
		StartLine: 1,
		EndLine:   1,
		Language:  "go",
		Vector:    query,
	}); err != nil {
		t.Fatalf("SaveChunk: %v", err)
	}

	results, err := store.SearchSemantic(query, 5, true)
	if err != nil {
		t.Fatalf("SearchSemantic fallback: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected fallback search results")
	}
}

func TestIndexStalenessDetection(t *testing.T) {
	store, cleanup := newTestSQLiteStorage(t)
	defer cleanup()

	for i := 0; i < 10; i++ {
		if _, err := store.SaveChunk(CodeChunk{FilePath: fmt.Sprintf("stale-%d.go", i), Content: "x", StartLine: 1, EndLine: 1, Language: "go", Vector: randomVector(384)}); err != nil {
			t.Fatalf("SaveChunk(%d): %v", i, err)
		}
	}

	idx := vectorindex.NewHNSW(384, vectorindex.Config{})
	embeddings, err := store.GetAllEmbeddings()
	if err != nil {
		t.Fatalf("GetAllEmbeddings: %v", err)
	}
	for id, vec := range embeddings {
		idx.Insert(id, vec)
	}

	hash1, err := store.ChunkTableContentHash()
	if err != nil {
		t.Fatalf("ChunkTableContentHash: %v", err)
	}
	if err := idx.Save(store.VectorIndexPath(), hash1, "none", nil); err != nil {
		t.Fatalf("Save index: %v", err)
	}
	if hash2, err := store.ChunkTableContentHash(); err != nil || hash1 != hash2 {
		t.Fatalf("hash should remain stable before modification: %v", err)
	}

	if _, err := store.SaveChunk(CodeChunk{FilePath: "stale-new.go", Content: "new", StartLine: 1, EndLine: 1, Language: "go", Vector: randomVector(384)}); err != nil {
		t.Fatalf("SaveChunk new: %v", err)
	}
	hash3, err := store.ChunkTableContentHash()
	if err != nil {
		t.Fatalf("ChunkTableContentHash modified: %v", err)
	}
	if hash3 == hash1 {
		t.Fatal("expected hash to change after modification")
	}

	// LoadVectorIndex should now FAIL because chunks changed (hash mismatch).
	// The returned hash is still valid for comparison.
	_, savedHash, _, _, err := store.LoadVectorIndex()
	if err == nil {
		t.Fatal("LoadVectorIndex should return error when chunks changed after index was built")
	}
	if savedHash != hash1 {
		t.Fatalf("saved hash mismatch: got %x, want %x", savedHash, hash1)
	}
	if savedHash == hash3 {
		t.Fatal("expected loaded hash to differ from current hash")
	}
}

func TestRequiresReindex_ModelMismatch(t *testing.T) {
	store, cleanup := newTestSQLiteStorage(t)
	defer cleanup()

	// No metadata → should not require reindex
	requires, err := store.RequiresReindex("embedded", "jina-embeddings-v3", 1024)
	if err != nil {
		t.Fatalf("RequiresReindex (no metadata): %v", err)
	}
	if requires {
		t.Error("RequiresReindex should return false when no metadata exists")
	}

	// Set metadata for bge-small
	if err := store.SetIndexMetadata(IndexMetadata{
		Provider:   "embedded",
		Model:      "bge-small-en-v1.5",
		Dimensions: 384,
	}); err != nil {
		t.Fatalf("SetIndexMetadata: %v", err)
	}

	// Same model → should not require reindex
	requires, err = store.RequiresReindex("embedded", "bge-small-en-v1.5", 384)
	if err != nil {
		t.Fatalf("RequiresReindex (same model): %v", err)
	}
	if requires {
		t.Error("RequiresReindex should return false when model matches")
	}

	// Different model → should require reindex
	requires, err = store.RequiresReindex("embedded", "jina-embeddings-v3", 1024)
	if err != nil {
		t.Fatalf("RequiresReindex (different model): %v", err)
	}
	if !requires {
		t.Error("RequiresReindex should return true when model differs")
	}

	// Different dimensions → should require reindex
	requires, err = store.RequiresReindex("embedded", "bge-small-en-v1.5", 768)
	if err != nil {
		t.Fatalf("RequiresReindex (different dims): %v", err)
	}
	if !requires {
		t.Error("RequiresReindex should return true when dimensions differ")
	}

	// Different provider → should require reindex
	requires, err = store.RequiresReindex("remote", "bge-small-en-v1.5", 384)
	if err != nil {
		t.Fatalf("RequiresReindex (different provider): %v", err)
	}
	if !requires {
		t.Error("RequiresReindex should return true when provider differs")
	}
}
