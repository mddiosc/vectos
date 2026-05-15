package vectorindex

import (
	"math"
	"math/rand"
	"sort"
	"testing"
)

// randomVectors generates n random L2-normalized vectors of dimension dim.
func randomVectors(n, dim int) [][]float32 {
	rng := rand.New(rand.NewSource(42))
	vectors := make([][]float32, n)
	for i := 0; i < n; i++ {
		v := make([]float32, dim)
		var norm float64
		for j := 0; j < dim; j++ {
			x := rng.Float64()*2 - 1
			v[j] = float32(x)
			norm += x * x
		}
		norm = math.Sqrt(norm)
		if norm == 0 {
			norm = 1
		}
		for j := range v {
			v[j] = float32(float64(v[j]) / norm)
		}
		vectors[i] = v
	}
	return vectors
}

// bruteForceSearch performs an exact cosine-distance search.
func bruteForceSearch(vectors [][]float32, query []float32, k int) []ScoredNeighbor {
	results := make([]ScoredNeighbor, 0, len(vectors))
	for i, v := range vectors {
		results = append(results, ScoredNeighbor{ID: i, Distance: CosineDistance(query, v)})
	}
	sort.Slice(results, func(i, j int) bool { return results[i].Distance < results[j].Distance })
	if len(results) > k {
		results = results[:k]
	}
	return results
}

func BenchmarkHNSWSearch_1K(b *testing.B)  { benchmarkHNSWSearch(b, 1000) }
func BenchmarkHNSWSearch_10K(b *testing.B) { benchmarkHNSWSearch(b, 10000) }
func BenchmarkLinearSearch_1K(b *testing.B) { benchmarkLinearSearch(b, 1000) }
func BenchmarkLinearSearch_10K(b *testing.B) { benchmarkLinearSearch(b, 10000) }
func BenchmarkHNSWBuild_1K(b *testing.B)   { benchmarkHNSWBuild(b, 1000) }
func BenchmarkHNSWBuild_10K(b *testing.B)  { benchmarkHNSWBuild(b, 10000) }

func benchmarkHNSWSearch(b *testing.B, n int) {
	const dim = 384
	vectors := randomVectors(n, dim)
	idx := NewHNSW(dim, Config{M: 16, EfConstruction: 200, EfSearch: 100})
	for i, v := range vectors {
		idx.Insert(i, v)
	}
	queries := randomVectors(b.N+1, dim)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = idx.SearchScored(queries[i], 10)
	}
}

func benchmarkLinearSearch(b *testing.B, n int) {
	const dim = 384
	vectors := randomVectors(n, dim)
	queries := randomVectors(b.N+1, dim)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = bruteForceSearch(vectors, queries[i], 10)
	}
}

func benchmarkHNSWBuild(b *testing.B, n int) {
	const dim = 384
	vectors := randomVectors(n, dim)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		idx := NewHNSW(dim, Config{M: 16, EfConstruction: 200, EfSearch: 100})
		for j, v := range vectors {
			idx.Insert(j, v)
		}
	}
}
