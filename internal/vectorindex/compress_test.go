package vectorindex

import (
	"crypto/sha256"
	"math"
	"math/rand/v2"
	"path/filepath"
	"testing"
)

func TestSQ8_RoundTrip(t *testing.T) {
	vectors := make([][]float32, 100)
	for i := range vectors {
		v := make([]float32, 384)
		for j := range v {
			v[j] = rand.Float32()*2 - 1
		}
		vectors[i] = v
	}
	params, err := ComputeSQ8Params(vectors)
	if err != nil {
		t.Fatalf("ComputeSQ8Params: %v", err)
	}
	decoded := DecodeSQ8(EncodeSQ8(vectors, params), params)
	for i := range vectors {
		for j := range vectors[i] {
			if err := math.Abs(float64(vectors[i][j] - decoded[i][j])); err > 0.1 {
				t.Fatalf("error too high at %d,%d: %v", i, j, err)
			}
		}
	}
}

func TestSQ8_SingleVector(t *testing.T) {
	v := [][]float32{{1, 2, 3}}
	params, err := ComputeSQ8Params(v)
	if err != nil {
		t.Fatalf("ComputeSQ8Params: %v", err)
	}
	decoded := DecodeSQ8(EncodeSQ8(v, params), params)
	if len(decoded) != 1 || len(decoded[0]) != 3 {
		t.Fatal("bad roundtrip")
	}
}

func TestSQ8_EmptyVectors(t *testing.T) {
	if out := EncodeSQ8(nil, &SQ8Params{}); len(out) != 0 {
		t.Fatal("expected empty")
	}
}

func TestSQ8_ParamsReturnErrorOnDimensionMismatch(t *testing.T) {
	if _, err := ComputeSQ8Params([][]float32{{1, 2, 3}, {1, 2}}); err == nil {
		t.Fatal("expected dimension mismatch error")
	}
}

func TestSQ8_SaveLoadWithCompression(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sq8.vectorindex")
	h := NewHNSW(16, Config{})
	vectors := make([][]float32, 100)
	for i := range vectors {
		vectors[i] = randomVector(16)
		if err := h.Insert(i, vectors[i]); err != nil {
			t.Fatalf("Insert: %v", err)
		}
	}
	params, err := ComputeSQ8Params(vectors)
	if err != nil {
		t.Fatalf("ComputeSQ8Params: %v", err)
	}
	if err := h.Save(path, sha256.Sum256([]byte("h")), "sq8", params); err != nil {
		t.Fatalf("Save: %v", err)
	}
	loaded, _, compression, _, err := LoadIndex(path)
	if err != nil || compression != "sq8" || loaded.Len() != 100 {
		t.Fatalf("load failed: %v %s", err, compression)
	}
	if len(loaded.Search(vectors[0], 5)) == 0 {
		t.Fatal("search failed")
	}
}

func TestSQ8_ParamsPersistence(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "params.vectorindex")
	h := NewHNSW(8, Config{})
	vectors := [][]float32{{1, 2, 3, 4, 5, 6, 7, 8}}
	if err := h.Insert(1, vectors[0]); err != nil {
		t.Fatalf("Insert: %v", err)
	}
	params, err := ComputeSQ8Params(vectors)
	if err != nil {
		t.Fatalf("ComputeSQ8Params: %v", err)
	}
	if err := h.Save(path, sha256.Sum256([]byte("p")), "sq8", params); err != nil {
		t.Fatalf("Save: %v", err)
	}
	_, _, compression, loadedParams, err := LoadIndex(path)
	if err != nil || compression != "sq8" || loadedParams == nil || loadedParams.Dim != params.Dim {
		t.Fatalf("params not persisted: %v", err)
	}
}

// TestBatchCompressIndexEndToEnd tests the full pipeline:
// batched embeddings → SQ8 compression → save → load → search.
func TestBatchCompressIndexEndToEnd(t *testing.T) {
	dim := 384
	numVectors := 200
	vectors := randomVectors(numVectors, dim)

	cfg := Config{M: 16, EfConstruction: 200, EfSearch: 100}
	cfg.withDefaults()
	idx := NewHNSW(dim, cfg)
	for i, v := range vectors {
		if err := idx.Insert(i, v); err != nil {
			t.Fatalf("Insert: %v", err)
		}
	}

	params, err := ComputeSQ8Params(vectors)
	if err != nil {
		t.Fatalf("ComputeSQ8Params: %v", err)
	}
	if params == nil || params.Dim != dim {
		t.Fatalf("unexpected SQ8 params: %+v", params)
	}
	encoded := EncodeSQ8(vectors, params)
	if len(encoded) != numVectors*dim {
		t.Fatalf("unexpected encoded length: %d", len(encoded))
	}

	tmpFile := filepath.Join(t.TempDir(), "test.vectorindex")
	contentHash := sha256.Sum256([]byte("batch-compress-index"))
	if err := idx.Save(tmpFile, contentHash, "sq8", params); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loadedIdx, _, compression, sq8Params, err := LoadIndex(tmpFile)
	if err != nil {
		t.Fatal(err)
	}
	if compression != "sq8" {
		t.Fatalf("expected compression sq8, got %q", compression)
	}
	if sq8Params == nil || sq8Params.Dim != dim {
		t.Fatalf("unexpected loaded params: %+v", sq8Params)
	}
	if loadedIdx.Len() != numVectors {
		t.Fatalf("unexpected node count: %d", loadedIdx.Len())
	}

	query := randomVectors(1, dim)[0]
	results := loadedIdx.SearchScored(query, 10)
	if len(results) == 0 {
		t.Fatal("no results")
	}

	bruteResults := bruteForceSearch(vectors, query, 10)
	overlap := 0
	for _, r := range results {
		for _, br := range bruteResults {
			if r.ID == br.ID {
				overlap++
				break
			}
		}
	}
	recall := float64(overlap) / float64(len(results))
	if recall < 0.8 {
		t.Fatalf("SQ8 recall too low: %.2f", recall)
	}
}
