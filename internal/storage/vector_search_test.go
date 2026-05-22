package storage

import (
	"fmt"
	"math/rand/v2"
	"os"
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
	requires, err := store.RequiresReindex("embedded", "jina-embeddings-v3", 1024, "fp-v1")
	if err != nil {
		t.Fatalf("RequiresReindex (no metadata): %v", err)
	}
	if requires {
		t.Error("RequiresReindex should return false when no metadata exists")
	}

	// Set metadata for bge-small
	if err := store.SetIndexMetadata(IndexMetadata{
		Provider:         "embedded",
		Model:            "bge-small-en-v1.5",
		Dimensions:       384,
		IndexFingerprint: "fp-v1",
	}); err != nil {
		t.Fatalf("SetIndexMetadata: %v", err)
	}

	// Same model → should not require reindex
	requires, err = store.RequiresReindex("embedded", "bge-small-en-v1.5", 384, "fp-v1")
	if err != nil {
		t.Fatalf("RequiresReindex (same model): %v", err)
	}
	if requires {
		t.Error("RequiresReindex should return false when model matches")
	}

	// Different model → should require reindex
	requires, err = store.RequiresReindex("embedded", "jina-embeddings-v3", 1024, "fp-v1")
	if err != nil {
		t.Fatalf("RequiresReindex (different model): %v", err)
	}
	if !requires {
		t.Error("RequiresReindex should return true when model differs")
	}

	// Different dimensions → should require reindex
	requires, err = store.RequiresReindex("embedded", "bge-small-en-v1.5", 768, "fp-v1")
	if err != nil {
		t.Fatalf("RequiresReindex (different dims): %v", err)
	}
	if !requires {
		t.Error("RequiresReindex should return true when dimensions differ")
	}

	// Different provider → should require reindex
	requires, err = store.RequiresReindex("remote", "bge-small-en-v1.5", 384, "fp-v1")
	if err != nil {
		t.Fatalf("RequiresReindex (different provider): %v", err)
	}
	if !requires {
		t.Error("RequiresReindex should return true when provider differs")
	}

	requires, err = store.RequiresReindex("embedded", "bge-small-en-v1.5", 384, "fp-v2")
	if err != nil {
		t.Fatalf("RequiresReindex (different fingerprint): %v", err)
	}
	if !requires {
		t.Error("RequiresReindex should return true when fingerprint differs")
	}
}

func TestClearIndexedData(t *testing.T) {
	store, cleanup := newTestSQLiteStorage(t)
	defer cleanup()

	if _, err := store.SaveChunk(CodeChunk{FilePath: "wipe.go", Content: "chunk", StartLine: 1, EndLine: 1, Language: "go", Vector: randomVector(4)}); err != nil {
		t.Fatalf("SaveChunk: %v", err)
	}
	if err := store.UpsertIndexedFile("wipe.go", "hash-wipe"); err != nil {
		t.Fatalf("UpsertIndexedFile: %v", err)
	}

	if err := store.ClearIndexedData(); err != nil {
		t.Fatalf("ClearIndexedData: %v", err)
	}

	stats, err := store.Stats()
	if err != nil {
		t.Fatalf("Stats: %v", err)
	}
	if stats.ChunkCount != 0 || stats.FileCount != 0 || stats.EmbeddedCount != 0 {
		t.Fatalf("expected empty store after clear, got %+v", stats)
	}
	if hash, err := store.GetIndexedFileHash("wipe.go"); err != nil || hash != "" {
		t.Fatalf("expected cleared file hash, got %q, %v", hash, err)
	}
}

func TestInvalidateEmbeddings(t *testing.T) {
	store, cleanup := newTestSQLiteStorage(t)
	defer cleanup()

	// Seed the store with chunks that have embeddings and file hashes.
	const dim = 384
	for i := 0; i < 5; i++ {
		path := fmt.Sprintf("inv-%d.go", i)
		if _, err := store.SaveChunk(CodeChunk{
			FilePath:  path,
			Content:   fmt.Sprintf("chunk %d", i),
			StartLine: 1,
			EndLine:   2,
			Language:  "go",
			Vector:    randomVector(dim),
		}); err != nil {
			t.Fatalf("SaveChunk(%d): %v", i, err)
		}
		if err := store.UpsertIndexedFile(path, fmt.Sprintf("hash-%d", i)); err != nil {
			t.Fatalf("UpsertIndexedFile(%d): %v", i, err)
		}
	}

	// Build and set a vector index so we can verify it gets cleared.
	embeddings, err := store.GetAllEmbeddings()
	if err != nil {
		t.Fatalf("GetAllEmbeddings: %v", err)
	}
	if len(embeddings) != 5 {
		t.Fatalf("expected 5 embeddings, got %d", len(embeddings))
	}
	idx := vectorindex.NewHNSW(dim, vectorindex.Config{M: 16, EfConstruction: 200, EfSearch: 50})
	for id, vec := range embeddings {
		idx.Insert(id, vec)
	}
	store.SetVectorIndex(idx)
	if !store.HasVectorIndex() {
		t.Fatal("expected vector index to be set")
	}

	// Verify file hashes exist.
	for i := 0; i < 5; i++ {
		h, err := store.GetIndexedFileHash(fmt.Sprintf("inv-%d.go", i))
		if err != nil {
			t.Fatalf("GetIndexedFileHash(%d): %v", i, err)
		}
		if h == "" {
			t.Fatalf("expected hash for inv-%d.go", i)
		}
	}

	// Invalidate.
	if err := store.InvalidateEmbeddings(); err != nil {
		t.Fatalf("InvalidateEmbeddings: %v", err)
	}

	// Embeddings should be gone.
	embeddingsAfter, err := store.GetAllEmbeddings()
	if err != nil {
		t.Fatalf("GetAllEmbeddings after invalidation: %v", err)
	}
	if len(embeddingsAfter) != 0 {
		t.Fatalf("expected 0 embeddings after invalidation, got %d", len(embeddingsAfter))
	}

	// File hashes should be gone (forces re-processing).
	for i := 0; i < 5; i++ {
		h, err := store.GetIndexedFileHash(fmt.Sprintf("inv-%d.go", i))
		if err != nil {
			t.Fatalf("GetIndexedFileHash(%d) after invalidation: %v", i, err)
		}
		if h != "" {
			t.Fatalf("expected empty hash for inv-%d.go after invalidation, got %q", i, h)
		}
	}

	// In-memory vector index should be cleared.
	if store.HasVectorIndex() {
		t.Fatal("expected vector index to be cleared after invalidation")
	}

	// Chunk text should still exist (only embeddings were cleared).
	results, err := store.SearchText("chunk")
	if err != nil {
		t.Fatalf("SearchText after invalidation: %v", err)
	}
	if len(results) != 5 {
		t.Fatalf("expected 5 chunks to survive invalidation, got %d", len(results))
	}
}

func TestGetAllEmbeddings_MixedDimensions(t *testing.T) {
	store, cleanup := newTestSQLiteStorage(t)
	defer cleanup()

	// Insert chunks with two different embedding dimensions (simulates model change).
	dim384 := 384
	dim1024 := 1024
	for i := 0; i < 8; i++ {
		if _, err := store.SaveChunk(CodeChunk{
			FilePath:  fmt.Sprintf("file384-%d.go", i),
			Content:   fmt.Sprintf("chunk dim384 %d", i),
			StartLine: 1,
			EndLine:   2,
			Language:  "go",
			Vector:    randomVector(dim384),
		}); err != nil {
			t.Fatalf("SaveChunk dim384 (%d): %v", i, err)
		}
	}
	for i := 0; i < 3; i++ {
		if _, err := store.SaveChunk(CodeChunk{
			FilePath:  fmt.Sprintf("file1024-%d.go", i),
			Content:   fmt.Sprintf("chunk dim1024 %d", i),
			StartLine: 1,
			EndLine:   2,
			Language:  "go",
			Vector:    randomVector(dim1024),
		}); err != nil {
			t.Fatalf("SaveChunk dim1024 (%d): %v", i, err)
		}
	}

	embeddings, err := store.GetAllEmbeddings()
	if err != nil {
		t.Fatalf("GetAllEmbeddings: %v", err)
	}

	// Determine dominant dimension (majority vote).
	dimCounts := make(map[int]int)
	for _, vec := range embeddings {
		dimCounts[len(vec)]++
	}
	var dimension, maxCount int
	for dim, count := range dimCounts {
		if count > maxCount {
			dimension = dim
			maxCount = count
		}
	}

	if dimension != dim384 {
		t.Fatalf("expected dominant dimension %d, got %d", dim384, dimension)
	}

	// Filter and build HNSW — must not panic.
	var ids []int
	for id, vec := range embeddings {
		if len(vec) == dimension {
			ids = append(ids, id)
		}
	}

	idx := vectorindex.NewHNSW(dimension, vectorindex.Config{M: 16, EfConstruction: 200, EfSearch: 50})
	for _, id := range ids {
		idx.Insert(id, embeddings[id]) // Should not panic
	}

	if idx.Len() != 8 {
		t.Fatalf("expected 8 vectors in index, got %d", idx.Len())
	}
}

func TestRebuildVectorIndexFromStoredEmbeddings(t *testing.T) {
	store, cleanup := newTestSQLiteStorage(t)
	defer cleanup()

	const dim = 128
	// Insert chunks with embeddings so RebuildVectorIndex has data to work with.
	for i := 0; i < 50; i++ {
		v := randomVector(dim)
		if _, err := store.SaveChunk(CodeChunk{
			FilePath:  fmt.Sprintf("r-%d.go", i),
			Content:   fmt.Sprintf("rebuild %d", i),
			StartLine: 1,
			EndLine:   1,
			Language:  "go",
			Vector:    v,
		}); err != nil {
			t.Fatalf("SaveChunk: %v", err)
		}
	}

	// Delete any stale vector index file from a previous test.
	// RebuildVectorIndex should create it from scratch.
	_ = os.Remove(store.VectorIndexPath())

	store.SetVectorIndexParams(16, 200, 200)
	if err := store.RebuildVectorIndex(); err != nil {
		t.Fatalf("RebuildVectorIndex: %v", err)
	}

	if !store.HasVectorIndex() {
		t.Fatal("expected vector index to be loaded after rebuild")
	}

	// Verify the index file was persisted.
	if _, err := os.Stat(store.VectorIndexPath()); err != nil {
		t.Fatalf("vector index file not persisted: %v", err)
	}

	// Searching with the index should return results.
	results, err := store.SearchSemantic(randomVector(dim), 3, true)
	if err != nil {
		t.Fatalf("SearchSemantic after rebuild: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected search results after rebuild")
	}
}

func TestSearchSemanticAutoRebuildsOnMissingIndex(t *testing.T) {
	store, cleanup := newTestSQLiteStorage(t)
	defer cleanup()

	const dim = 128
	for i := 0; i < 50; i++ {
		v := randomVector(dim)
		if _, err := store.SaveChunk(CodeChunk{
			FilePath:  fmt.Sprintf("a-%d.go", i),
			Content:   fmt.Sprintf("auto %d", i),
			StartLine: 1,
			EndLine:   1,
			Language:  "go",
			Vector:    v,
		}); err != nil {
			t.Fatalf("SaveChunk: %v", err)
		}
	}

	// Remove any existing vector index — simulate the "index never built" case.
	_ = os.Remove(store.VectorIndexPath())

	// Configure HNSW params so auto-rebuild can proceed with explicit dimensions.
	store.SetVectorIndexParams(16, 200, 200)

	query := randomVector(dim)
	results, err := store.SearchSemantic(query, 3, true)
	if err != nil {
		t.Fatalf("SearchSemantic: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected search results after auto-rebuild")
	}

	// The index should now be loaded (auto-rebuilt by the first search).
	if !store.HasVectorIndex() {
		t.Fatal("expected vector index to be loaded after auto-rebuild")
	}

	// Subsequent searches should use the index without rebuilding.
	results2, err := store.SearchSemantic(query, 3, true)
	if err != nil {
		t.Fatalf("second SearchSemantic: %v", err)
	}
	if len(results2) == 0 {
		t.Fatal("expected results on second search")
	}
}

func TestSearchSemanticAutoRebuildsOnStaleIndex(t *testing.T) {
	store, cleanup := newTestSQLiteStorage(t)
	defer cleanup()

	const dim = 128
	for i := 0; i < 50; i++ {
		v := randomVector(dim)
		if _, err := store.SaveChunk(CodeChunk{
			FilePath:  fmt.Sprintf("s-%d.go", i),
			Content:   fmt.Sprintf("stale %d", i),
			StartLine: 1,
			EndLine:   1,
			Language:  "go",
			Vector:    v,
		}); err != nil {
			t.Fatalf("SaveChunk: %v", err)
		}
	}

	// Build and save a valid index.
	store.SetVectorIndexParams(16, 200, 200)
	if err := store.RebuildVectorIndex(); err != nil {
		t.Fatalf("RebuildVectorIndex: %v", err)
	}

	// Now mutate the chunk table (add a new chunk) so the content hash diverges.
	if _, err := store.SaveChunk(CodeChunk{
		FilePath:  "s-extra.go",
		Content:   "extra chunk to break hash",
		StartLine: 1,
		EndLine:   1,
		Language:  "go",
		Vector:    randomVector(dim),
	}); err != nil {
		t.Fatalf("SaveChunk(extra): %v", err)
	}

	// The index is now stale (content hash changed). The first search should
	// detect the staleness and trigger a rebuild.
	query := randomVector(dim)
	results, err := store.SearchSemantic(query, 3, true)
	if err != nil {
		t.Fatalf("SearchSemantic on stale index: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected search results after stale-index rebuild")
	}

	if !store.HasVectorIndex() {
		t.Fatal("expected vector index to be reloaded after stale detection")
	}
}
