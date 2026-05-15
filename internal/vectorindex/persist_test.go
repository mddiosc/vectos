package vectorindex

import (
	"crypto/sha256"
	"math/rand/v2"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func randomVector384() []float32 {
	v := make([]float32, 384)
	for i := range v {
		v[i] = rand.Float32()
	}
	return v
}

func normalize(v []float32) {
	var norm float64
	for _, x := range v {
		norm += float64(x) * float64(x)
	}
	scale := float32(1.0 / float64(norm))
	if norm > 0 {
		for i := range v {
			v[i] *= scale
		}
	}
}

func TestSaveLoad_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.vectorindex")

	// Build index with 100 random vectors.
	cfg := Config{M: 16, EfConstruction: 200, EfSearch: 50}
	idx := NewHNSW(384, cfg)
	const N = 100
	vectors := make([][]float32, N)
	for i := 0; i < N; i++ {
		v := randomVector384()
		normalize(v)
		vectors[i] = v
		idx.Insert(i, v)
	}

	contentHash := sha256.Sum256([]byte("test-hash"))
	if err := idx.Save(path, contentHash, "none", nil); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, loadedHash, _, _, err := LoadIndex(path)
	if err != nil {
		t.Fatalf("LoadIndex: %v", err)
	}
	if loadedHash != contentHash {
		t.Errorf("content hash mismatch: got %x, want %x", loadedHash, contentHash)
	}
	if loaded.Len() != N {
		t.Errorf("node count: got %d, want %d", loaded.Len(), N)
	}
	if loaded.dimension != 384 {
		t.Errorf("dimension: got %d, want 384", loaded.dimension)
	}
	if loaded.M != 16 {
		t.Errorf("M: got %d, want 16", loaded.M)
	}
	if loaded.efConstruction != 200 {
		t.Errorf("efConstruction: got %d, want 200", loaded.efConstruction)
	}
	if loaded.efSearch != 50 {
		t.Errorf("efSearch: got %d, want 50", loaded.efSearch)
	}

	// Query the loaded index and verify search works.
	results := loaded.Search(vectors[0], 5)
	if len(results) == 0 {
		t.Fatal("search on loaded index returned no results")
	}
	// The query vector itself should be the top result (distance ~0).
	if results[0] != 0 {
		t.Errorf("top result: got %d, want 0 (self-match)", results[0])
	}
}

func TestSaveLoad_EmptyIndex(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.vectorindex")

	idx := NewHNSW(384, Config{})
	contentHash := [sha256.Size]byte{1, 2, 3}
	if err := idx.Save(path, contentHash, "none", nil); err != nil {
		t.Fatalf("Save empty: %v", err)
	}

	loaded, loadedHash, _, _, err := LoadIndex(path)
	if err != nil {
		t.Fatalf("LoadIndex empty: %v", err)
	}
	if loadedHash != contentHash {
		t.Errorf("content hash mismatch")
	}
	if loaded.Len() != 0 {
		t.Errorf("expected empty index, got %d nodes", loaded.Len())
	}
}

func TestLoadIndex_MissingFile(t *testing.T) {
	_, _, _, _, err := LoadIndex("/nonexistent/path.vectorindex")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestLoadIndex_BadMagic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.vectorindex")
	if err := os.WriteFile(path, []byte("BAD MAGIC FILE"), 0644); err != nil {
		t.Fatal(err)
	}
	_, _, _, _, err := LoadIndex(path)
	if err == nil {
		t.Fatal("expected error for bad magic")
	}
}

func TestLoadIndex_VersionMismatch(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "version.vectorindex")
	// Write magic + wrong version.
	data := make([]byte, 8)
	// Magic
	data[0] = 0x48 // H
	data[1] = 0x53 // S
	data[2] = 0x4E // N
	data[3] = 0x57 // W
	// Version = 99
	data[4] = 99
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
	_, _, _, _, err := LoadIndex(path)
	if err == nil {
		t.Fatal("expected error for version mismatch")
	}
}

func TestSaveLoad_StalenessDetection(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "stale.vectorindex")

	idx := NewHNSW(384, Config{})
	v := randomVector384()
	normalize(v)
	idx.Insert(0, v)

	hash1 := sha256.Sum256([]byte("state-v1"))
	if err := idx.Save(path, hash1, "none", nil); err != nil {
		t.Fatalf("Save: %v", err)
	}

	_, loadedHash, _, _, err := LoadIndex(path)
	if err != nil {
		t.Fatalf("LoadIndex: %v", err)
	}
	if loadedHash != hash1 {
		t.Errorf("hash mismatch: got %x, want %x", loadedHash, hash1)
	}

	// Different hash would indicate stale state.
	hash2 := sha256.Sum256([]byte("state-v2"))
	if hash1 == hash2 {
		t.Fatal("hashes should differ for different inputs")
	}
	// Caller would compare loadedHash != hash2 to detect staleness.
	// loadedHash == hash1 != hash2, so index is stale if hash2 is current.
	if loadedHash == hash2 {
		t.Error("loaded hash matches new hash — should be different")
	}
}

func TestSaveLoad_ContentHashZeroPreserved(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "zero.vectorindex")

	idx := NewHNSW(384, Config{})
	v := randomVector384()
	normalize(v)
	idx.Insert(0, v)

	var zeroHash [sha256.Size]byte // all zeros
	if err := idx.Save(path, zeroHash, "none", nil); err != nil {
		t.Fatalf("Save: %v", err)
	}

	_, loadedHash, _, _, err := LoadIndex(path)
	if err != nil {
		t.Fatalf("LoadIndex: %v", err)
	}
	if loadedHash != zeroHash {
		t.Errorf("zero hash not preserved: got %x", loadedHash)
	}
}

func TestIndexRebuild(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "rebuild.vectorindex")

	build := func(n int) *HNSW {
		idx := NewHNSW(384, Config{M: 16, EfConstruction: 100, EfSearch: 50})
		for i := 0; i < n; i++ {
			v := randomVector384()
			normalize(v)
			idx.Insert(i, v)
		}
		return idx
	}

	idx1 := build(50)
	hash1 := sha256.Sum256([]byte("rebuild-1"))
	if err := idx1.Save(path, hash1, "none", nil); err != nil {
		t.Fatalf("Save first index: %v", err)
	}
	info1, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat first index: %v", err)
	}

	time.Sleep(10 * time.Millisecond)

	idx2 := build(100)
	hash2 := sha256.Sum256([]byte("rebuild-2"))
	if err := idx2.Save(path, hash2, "none", nil); err != nil {
		t.Fatalf("Save second index: %v", err)
	}
	info2, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat second index: %v", err)
	}
	if info2.Size() == info1.Size() && !info2.ModTime().After(info1.ModTime()) {
		t.Fatal("expected index file to be overwritten")
	}

	loaded, loadedHash, _, _, err := LoadIndex(path)
	if err != nil {
		t.Fatalf("LoadIndex rebuilt: %v", err)
	}
	if loadedHash != hash2 {
		t.Fatalf("loaded hash mismatch: got %x want %x", loadedHash, hash2)
	}
	if loaded.Len() != 100 {
		t.Fatalf("node count mismatch: got %d want 100", loaded.Len())
	}
	if len(loaded.Search(loaded.GetVector(0), 5)) == 0 {
		t.Fatal("expected rebuilt index search to return results")
	}
}
