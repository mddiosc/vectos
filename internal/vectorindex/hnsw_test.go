package vectorindex

import (
	"math"
	"math/rand/v2"
	"testing"
)

func TestHNSWInsert_FirstNode(t *testing.T) {
	cfg := Config{M: 16, EfConstruction: 100, EfSearch: 50}
	h := NewHNSW(3, cfg)
	h.Insert(1, []float32{0.1, 0.2, 0.3})

	if h.Len() != 1 {
		t.Fatalf("expected 1 node, got %d", h.Len())
	}
	if h.entryPoint != 0 {
		t.Fatalf("expected entryPoint 0, got %d", h.entryPoint)
	}
}

func TestHNSWInsert_MultipleNodes(t *testing.T) {
	cfg := Config{M: 8, EfConstruction: 100, EfSearch: 50}
	h := NewHNSW(4, cfg)
	for i := 0; i < 100; i++ {
		h.Insert(i, randomVector(4))
	}
	if h.Len() != 100 {
		t.Fatalf("expected 100 nodes, got %d", h.Len())
	}
}

func TestHNSWSearch_Empty(t *testing.T) {
	h := NewHNSW(4, Config{})
	result := h.Search([]float32{1, 0, 0, 0}, 10)
	if len(result) != 0 {
		t.Errorf("expected empty result, got %v", result)
	}
}

func TestHNSWSearch_ReturnTopK(t *testing.T) {
	cfg := Config{M: 8, EfConstruction: 100, EfSearch: 50}
	h := NewHNSW(4, cfg)

	// Insert known vectors.
	vectors := [][]float32{
		{1.0, 0.0, 0.0, 0.0}, // id 0
		{1.0, 0.1, 0.0, 0.0}, // id 1 — very close to 0
		{0.0, 1.0, 0.0, 0.0}, // id 2
		{0.0, 0.1, 1.0, 0.0}, // id 3
		{0.0, 0.0, 0.1, 1.0}, // id 4
	}
	for i, v := range vectors {
		h.Insert(i, v)
	}

	// Query = [1, 0, 0, 0] — closest should be id 0 then id 1.
	query := []float32{1.0, 0.0, 0.0, 0.0}
	result := h.Search(query, 2)
	if len(result) != 2 {
		t.Fatalf("expected 2 results, got %d", len(result))
	}
	if result[0] != 0 {
		t.Errorf("expected id 0 as closest, got %d", result[0])
	}
	if result[1] != 1 {
		t.Errorf("expected id 1 as second closest, got %d", result[1])
	}
}

func TestHNSWSearch_CosineDistanceOrdering(t *testing.T) {
	cfg := Config{M: 8, EfConstruction: 100, EfSearch: 50}
	h := NewHNSW(4, cfg)

	vectors := [][]float32{
		{1.0, 0.0, 0.0, 0.0}, // id 0 — exact match
		{0.0, 1.0, 0.0, 0.0}, // id 1 — orthogonal
		{0.0, 0.0, 1.0, 0.0}, // id 2 — orthogonal
	}
	for i, v := range vectors {
		h.Insert(i, v)
	}

	query := []float32{1.0, 0.0, 0.0, 0.0}
	result := h.Search(query, 3)
	if len(result) != 3 {
		t.Fatalf("expected 3 results, got %d", len(result))
	}
	// First result should be id 0 (cosine distance = 0).
	if result[0] != 0 {
		t.Errorf("expected id 0 as closest, got %d", result[0])
	}
	// Verify distances are monotonically increasing.
	prev := CosineDistance(query, vectors[result[0]])
	for i := 1; i < len(result); i++ {
		d := CosineDistance(query, vectors[result[i]])
		if d < prev-1e-9 {
			t.Errorf("distance not monotonic at position %d: %v < %v", i, d, prev)
		}
		prev = d
	}
}

func TestHNSWSearch_KGreaterThanNodes(t *testing.T) {
	cfg := Config{M: 8, EfConstruction: 100, EfSearch: 50}
	h := NewHNSW(3, cfg)
	for i := 0; i < 5; i++ {
		h.Insert(i, randomVector(3))
	}
	result := h.Search(randomVector(3), 100)
	if len(result) > 5 {
		t.Errorf("expected at most 5 results, got %d", len(result))
	}
}

func TestHNSWSearch_RecallOnRandomVectors(t *testing.T) {
	cfg := Config{M: 16, EfConstruction: 200, EfSearch: 100}
	h := NewHNSW(384, cfg)
	n := 500
	rng := rand.New(rand.NewPCG(42, 42))
	vectors := make([][]float32, n)
	for i := 0; i < n; i++ {
		v := make([]float32, 384)
		for j := range v {
			v[j] = float32(rng.NormFloat64())
		}
		// L2-normalize.
		var norm float64
		for _, f := range v {
			norm += float64(f) * float64(f)
		}
		norm = math.Sqrt(norm)
		for j := range v {
			v[j] = float32(float64(v[j]) / norm)
		}
		vectors[i] = v
		h.Insert(i, v)
	}

	// Test 50 queries, check recall@10 vs brute-force.
	totalRecall := 0.0
	queryCount := 50
	k := 10
	for q := 0; q < queryCount; q++ {
		query := make([]float32, 384)
		for j := range query {
			query[j] = float32(rng.NormFloat64())
		}
		var norm float64
		for _, f := range query {
			norm += float64(f) * float64(f)
		}
		norm = math.Sqrt(norm)
		for j := range query {
			query[j] = float32(float64(query[j]) / norm)
		}

		// Brute-force top-k.
		all := make([]pair, n)
		for i := 0; i < n; i++ {
			all[i] = pair{id: i, dist: CosineDistance(query, vectors[i])}
		}
		sortPairs(all)
		groundTruth := make(map[int]bool)
		for i := 0; i < k; i++ {
			groundTruth[all[i].id] = true
		}

		// HNSW result.
		result := h.Search(query, k)
		hits := 0
		for _, id := range result {
			if groundTruth[id] {
				hits++
			}
		}
		totalRecall += float64(hits) / float64(k)
	}

	avgRecall := totalRecall / float64(queryCount)
	t.Logf("Average recall@%d over %d queries: %.2f%%", k, queryCount, avgRecall*100)
	if avgRecall < 0.95 {
		t.Errorf("recall %.4f below 0.95 threshold", avgRecall)
	}
}

func TestHNSW_EfSearch(t *testing.T) {
	cfg := Config{M: 8, EfConstruction: 100, EfSearch: 50}
	h := NewHNSW(3, cfg)
	if h.EfSearch() != 50 {
		t.Errorf("expected efSearch=50, got %d", h.EfSearch())
	}
	h.SetEfSearch(200)
	if h.EfSearch() != 200 {
		t.Errorf("expected efSearch=200, got %d", h.EfSearch())
	}
}

func TestHNSWInsert_PanicsOnDimensionMismatch(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic on dimension mismatch")
		}
	}()
	h := NewHNSW(3, Config{})
	h.Insert(1, []float32{1, 2})
}

func TestCosineDistance_Identical(t *testing.T) {
	v := []float32{1, 2, 3}
	if d := CosineDistance(v, v); d > 1e-9 {
		t.Errorf("identical vectors should have distance 0, got %v", d)
	}
}

func TestCosineDistance_Opposite(t *testing.T) {
	a := []float32{1, 0, 0}
	b := []float32{0, 1, 0}
	d := CosineDistance(a, b)
	if d < 0.99 || d > 1.01 {
		t.Errorf("orthogonal vectors should have distance ~1, got %v", d)
	}
}

func TestConfigDefaults(t *testing.T) {
	var c Config
	c.withDefaults()
	if c.M != 16 {
		t.Errorf("expected M=16, got %d", c.M)
	}
	if c.EfConstruction != 200 {
		t.Errorf("expected EfConstruction=200, got %d", c.EfConstruction)
	}
	if c.EfSearch != 100 {
		t.Errorf("expected EfSearch=100, got %d", c.EfSearch)
	}
}

func randomVector(dim int) []float32 {
	v := make([]float32, dim)
	for i := range v {
		v[i] = rand.Float32()
	}
	return v
}

type pair struct {
	id   int
	dist float64
}

func sortPairs(pairs []pair) {
	for i := 1; i < len(pairs); i++ {
		for j := i; j > 0 && pairs[j].dist < pairs[j-1].dist; j-- {
			pairs[j], pairs[j-1] = pairs[j-1], pairs[j]
		}
	}
}

func TestHNSWSearchScored_NonContiguousExternalIDs(t *testing.T) {
	cfg := Config{M: 8, EfConstruction: 100, EfSearch: 50}
	h := NewHNSW(4, cfg)

	// Insert with non-contiguous, non-zero-based IDs to verify
	// SearchScored returns the external chunk IDs, not internal indices.
	insertIDs := []int{10, 25, 47, 99}
	insertVectors := [][]float32{
		{1.0, 0.0, 0.0, 0.0}, // internal index 0
		{1.0, 0.1, 0.0, 0.0}, // internal index 1
		{0.0, 1.0, 0.0, 0.0}, // internal index 2
		{0.0, 0.1, 1.0, 0.0}, // internal index 3
	}
	for i, v := range insertVectors {
		h.Insert(insertIDs[i], v)
	}

	query := []float32{1.0, 0.0, 0.0, 0.0}
	results := h.SearchScored(query, 2)
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	// IDs should be the external ones (10, 25), NOT internal indices (0, 1).
	if results[0].ID != 10 {
		t.Errorf("expected external ID 10 as closest, got %d", results[0].ID)
	}
	if results[1].ID != 25 {
		t.Errorf("expected external ID 25 as second closest, got %d", results[1].ID)
	}
}
