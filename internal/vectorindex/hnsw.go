// Package vectorindex provides a pure-Go HNSW (Hierarchical Navigable Small
// World) approximate nearest neighbor index for float32 vectors with cosine
// distance.
package vectorindex

import (
	"math"
	"math/rand/v2"
	"sort"
)

// HNSW is a Hierarchical Navigable Small World graph for approximate k-NN
// search. All vectors MUST have the same dimensionality.
type HNSW struct {
	nodes          []node
	entryPoint     int
	maxLevel       int
	dimension      int
	M              int // max connections per layer (layers > 0)
	Mmax0          int // max connections on layer 0 (default 2*M)
	efConstruction int // size of dynamic candidate list during construction
	efSearch       int // size of dynamic candidate list during query
	mL             float64
}

// Config holds construction-time parameters for the HNSW index.
type Config struct {
	M              int // max neighbors per node per layer; default 16
	EfConstruction int // candidate list size during construction; default 200
	EfSearch       int // candidate list size during query; default 100
}

func (c *Config) withDefaults() {
	if c.M <= 0 {
		c.M = 16
	}
	if c.EfConstruction <= 0 {
		c.EfConstruction = 200
	}
	if c.EfSearch <= 0 {
		c.EfSearch = 100
	}
}

type node struct {
	id       int
	vector   []float32
	// layers[i] contains the neighbor IDs at level i.
	layers [][]int
}

// NewHNSW creates an empty HNSW index. dimension is the length of vectors
// that will be inserted.
func NewHNSW(dimension int, cfg Config) *HNSW {
	cfg.withDefaults()
	return &HNSW{
		entryPoint:     -1,
		maxLevel:       -1,
		dimension:      dimension,
		M:              cfg.M,
		Mmax0:          cfg.M * 2,
		efConstruction: cfg.EfConstruction,
		efSearch:       cfg.EfSearch,
		mL:             1.0 / math.Log(float64(cfg.M)),
	}
}

// EfSearch returns the current ef_search parameter.
func (h *HNSW) EfSearch() int { return h.efSearch }

// SetEfSearch changes the query-time ef_search parameter.
func (h *HNSW) SetEfSearch(ef int) {
	if ef > 0 {
		h.efSearch = ef
	}
}

// Len returns the number of nodes in the index.
func (h *HNSW) Len() int { return len(h.nodes) }

// MaxLevel returns the highest layer in the index, or -1 when empty.
func (h *HNSW) MaxLevel() int { return h.maxLevel }

// Dimension returns the vector dimensionality expected by the index.
func (h *HNSW) Dimension() int { return h.dimension }

// Insert adds a vector to the index. id must be unique across all insertions.
func (h *HNSW) Insert(id int, vector []float32) {
	if len(vector) != h.dimension {
		panic("vectorindex: vector dimension mismatch")
	}

	// Normalize the vector so distance calculations reduce to 1 - dot(a,b).
	normalized := make([]float32, len(vector))
	copy(normalized, vector)
	normalizeVec(normalized)

	// Allocate node.
	n := node{id: id, vector: normalized}
	level := randomLevel(h.mL)
	n.layers = make([][]int, level+1)
	h.nodes = append(h.nodes, n)
	nodeIdx := len(h.nodes) - 1

	if h.entryPoint == -1 {
		// First node — no edges to build.
		h.entryPoint = nodeIdx
		h.maxLevel = level
		return
	}

	// Phase 1: traverse from top down to level+1 to find entry point for this level.
	curObj := h.entryPoint
	for lc := h.maxLevel; lc > level; lc-- {
		curObj = h.searchLayerLocal(normalized, curObj, 1, lc)[0]
	}

	// Phase 2: insert into layers level down to 0.
	for lc := min(level, h.maxLevel); lc >= 0; lc-- {
		candidates := h.searchLayerLocal(normalized, curObj, h.efConstruction, lc)

		// Select M neighbors and connect bidirectionally.
		mMax := h.M
		if lc == 0 {
			mMax = h.Mmax0
		}
		neighbors := selectSimple(candidates, mMax, h.nodes)

		h.nodes[nodeIdx].layers[lc] = make([]int, 0, len(neighbors))
		for _, e := range neighbors {
			// Connect nodeIdx <-> e
			h.connect(lc, nodeIdx, e)
			// Prune e if needed.
			h.prune(lc, e, mMax)
		}
	}

	// If new node has higher level, update global entry point.
	if level > h.maxLevel {
		h.maxLevel = level
		h.entryPoint = nodeIdx
	}
}

// ScoredNeighbor holds a node ID and its cosine distance from a query.
type ScoredNeighbor struct {
	ID       int
	Distance float64 // cosine distance (0 = identical, 2 = opposite)
}

// Search returns the IDs of the k nearest neighbors to the query vector.
func (h *HNSW) Search(query []float32, k int) []int {
	results := h.SearchScored(query, k)
	ids := make([]int, len(results))
	for i, r := range results {
		ids[i] = r.ID
	}
	return ids
}

// SearchScored returns the k nearest neighbors with their cosine distances.
func (h *HNSW) SearchScored(query []float32, k int) []ScoredNeighbor {
	if len(h.nodes) == 0 || k <= 0 {
		return nil
	}

	// Normalize query to match pre-normalized index vectors.
	normQuery := make([]float32, len(query))
	copy(normQuery, query)
	normalizeVec(normQuery)

	ef := h.efSearch
	if k > ef {
		ef = k
	}

	curObj := h.entryPoint
	for lc := h.maxLevel; lc > 0; lc-- {
		curObj = h.searchLayerLocal(normQuery, curObj, 1, lc)[0]
	}

	candidates := h.searchLayerLocal(normQuery, curObj, ef, 0)

	// Extract top-k scored results.
	if len(candidates) > k {
		candidates = candidates[:k]
	}

	out := make([]ScoredNeighbor, len(candidates))
	for i, idx := range candidates {
		out[i] = ScoredNeighbor{ID: h.nodes[idx].id, Distance: distance(normQuery, h.nodes[idx].vector)}
	}
	return out
}

// GetVector returns the vector for a node by its ID. Returns nil if not found.
func (h *HNSW) GetVector(id int) []float32 {
	for i := range h.nodes {
		if h.nodes[i].id == id {
			return h.nodes[i].vector
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

// normalizeVec normalizes a vector in-place to unit length.
func normalizeVec(v []float32) {
	var norm float64
	for _, x := range v {
		norm += float64(x) * float64(x)
	}
	if norm == 0 {
		return
	}
	scale := 1.0 / math.Sqrt(norm)
	for i := range v {
		v[i] = float32(float64(v[i]) * scale)
	}
}

// distance returns the cosine distance between two pre-normalized vectors.
// Since vectors are normalized at insertion time, this is simply 1 - dot(a,b).
func distance(a, b []float32) float64 {
	return 1.0 - dotProduct(a, b)
}

// dotProduct computes the dot product of two float32 vectors.
// Uses 8-way loop unrolling to maximize instruction-level parallelism
// on modern CPUs with wide pipelines (e.g., Apple M-series, AMD Zen).
func dotProduct(a, b []float32) float64 {
	n := len(a)
	var d0, d1, d2, d3, d4, d5, d6, d7 float64
	i := 0
	for ; i <= n-8; i += 8 {
		d0 += float64(a[i]) * float64(b[i])
		d1 += float64(a[i+1]) * float64(b[i+1])
		d2 += float64(a[i+2]) * float64(b[i+2])
		d3 += float64(a[i+3]) * float64(b[i+3])
		d4 += float64(a[i+4]) * float64(b[i+4])
		d5 += float64(a[i+5]) * float64(b[i+5])
		d6 += float64(a[i+6]) * float64(b[i+6])
		d7 += float64(a[i+7]) * float64(b[i+7])
	}
	for ; i < n; i++ {
		d0 += float64(a[i]) * float64(b[i])
	}
	return d0 + d1 + d2 + d3 + d4 + d5 + d6 + d7
}

// connect adds a bidirectional edge at the given layer.
func (h *HNSW) connect(layer, a, b int) {
	h.nodes[a].layers[layer] = append(h.nodes[a].layers[layer], b)
	h.nodes[b].layers[layer] = append(h.nodes[b].layers[layer], a)
}

// prune keeps at most maxN closest neighbors of node at the given layer.
func (h *HNSW) prune(layer, nodeIdx, maxN int) {
	neighbors := h.nodes[nodeIdx].layers[layer]
	if len(neighbors) <= maxN {
		return
	}

	// Pre-compute distances once instead of recalculating in every sort comparison.
	vec := h.nodes[nodeIdx].vector
	type distEntry struct {
		idx  int
		dist float64
	}
	entries := make([]distEntry, len(neighbors))
	for i, nb := range neighbors {
		entries[i] = distEntry{idx: nb, dist: distance(vec, h.nodes[nb].vector)}
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].dist < entries[j].dist
	})
	for i := range entries {
		neighbors[i] = entries[i].idx
	}
	h.nodes[nodeIdx].layers[layer] = neighbors[:maxN]
}

// searchLayerLocal performs a layer-local search returning the closest ef
// node indices, sorted by distance.
func (h *HNSW) searchLayerLocal(query []float32, entry int, ef, layer int) []int {
	visited := make(map[int]bool, 64)
	visited[entry] = true

	// candidates: min-heap of nodes to explore (closest first).
	// We implement a simple sorted-slice min-heap.
	candidates := &scoredHeap{scores: []scored{{idx: entry, dist: distance(query, h.nodes[entry].vector)}}}
	candidates.init()

	// results: max-heap of best ef nodes found so far.
	results := &scoredMaxHeap{limit: ef}
	results.init()

	results.push(scored{idx: entry, dist: distance(query, h.nodes[entry].vector)})

	for candidates.len() > 0 {
		c := candidates.pop() // closest candidate
		f := results.top()    // farthest in results

		// If c is farther than f and results is full, we're done.
		if f.dist >= 0 && c.dist > f.dist {
			break
		}

		for _, e := range h.nodes[c.idx].layers[layer] {
			if visited[e] {
				continue
			}
			visited[e] = true

			ed := distance(query, h.nodes[e].vector)
			f = results.top()

			if ed < f.dist || results.len() < ef {
				candidates.push(scored{idx: e, dist: ed})
				results.push(scored{idx: e, dist: ed})
			}
		}
	}

	// Return results sorted by distance.
	out := make([]int, results.len())
	for i := results.len() - 1; i >= 0; i-- {
		out[i] = results.pop().idx
	}
	return out
}

// selectSimple returns up to maxM closest entries from candidates to the
// query point (measured by distance to the first candidate's vector… but
// actually we measure distance to the inserted node's vector). For simplicity,
// we sort candidates by distance to the new node's vector and return the first
// maxM.
func selectSimple(candidates []int, maxM int, nodes []node) []int {
	if len(candidates) <= maxM {
		return candidates
	}
	// candidates already sorted by distance. Just take first maxM.
	return candidates[:maxM]
}

func randomLevel(mL float64) int {
	return int(-math.Log(rand.Float64()) * mL)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ---------------------------------------------------------------------------
// Distance helpers
// ---------------------------------------------------------------------------

// CosineDistance returns the cosine distance between two vectors.
func CosineDistance(a, b []float32) float64 {
	return 1.0 - cosineSimilarity(a, b)
}

func cosineSimilarity(a, b []float32) float64 {
	if len(a) == 0 || len(a) != len(b) {
		return 0
	}
	var dot, normA, normB float64
	for i := range a {
		av := float64(a[i])
		bv := float64(b[i])
		dot += av * bv
		normA += av * av
		normB += bv * bv
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}

// ---------------------------------------------------------------------------
// Priority queues (min-heap for candidates, max-heap for results)
// ---------------------------------------------------------------------------

type scored struct {
	idx  int
	dist float64
}

type scoredHeap struct {
	scores []scored
}

func (h *scoredHeap) init() {
	// Build min-heap.
	n := len(h.scores)
	for i := n/2 - 1; i >= 0; i-- {
		h.siftDown(i)
	}
}

func (h *scoredHeap) len() int           { return len(h.scores) }
func (h *scoredHeap) push(s scored)      { h.scores = append(h.scores, s); h.siftUp(len(h.scores) - 1) }
func (h *scoredHeap) less(i, j int) bool { return h.scores[i].dist < h.scores[j].dist }
func (h *scoredHeap) swap(i, j int)      { h.scores[i], h.scores[j] = h.scores[j], h.scores[i] }

func (h *scoredHeap) pop() scored {
	n := len(h.scores)
	s := h.scores[0]
	h.scores[0] = h.scores[n-1]
	h.scores = h.scores[:n-1]
	if len(h.scores) > 0 {
		h.siftDown(0)
	}
	return s
}

func (h *scoredHeap) siftUp(i int) {
	for i > 0 {
		p := (i - 1) / 2
		if h.less(p, i) {
			break
		}
		h.swap(p, i)
		i = p
	}
}

func (h *scoredHeap) siftDown(i int) {
	n := len(h.scores)
	for {
		smallest := i
		l, r := 2*i+1, 2*i+2
		if l < n && h.less(l, smallest) {
			smallest = l
		}
		if r < n && h.less(r, smallest) {
			smallest = r
		}
		if smallest == i {
			break
		}
		h.swap(i, smallest)
		i = smallest
	}
}

type scoredMaxHeap struct {
	scores []scored
	limit  int
}

func (h *scoredMaxHeap) init()   {}
func (h *scoredMaxHeap) len() int { return len(h.scores) }

func (h *scoredMaxHeap) push(s scored) {
	if len(h.scores) >= h.limit {
		// If new score is farther than current max, skip.
		if s.dist >= h.top().dist {
			return
		}
		// Replace the max.
		h.pop()
	}
	h.scores = append(h.scores, s)
	h.siftUp(len(h.scores) - 1)
}

func (h *scoredMaxHeap) top() scored {
	if len(h.scores) == 0 {
		return scored{dist: -1}
	}
	return h.scores[0]
}

func (h *scoredMaxHeap) pop() scored {
	n := len(h.scores)
	s := h.scores[0]
	h.scores[0] = h.scores[n-1]
	h.scores = h.scores[:n-1]
	if len(h.scores) > 0 {
		h.siftDown(0)
	}
	return s
}

func (h *scoredMaxHeap) less(i, j int) bool { return h.scores[i].dist > h.scores[j].dist }
func (h *scoredMaxHeap) swap(i, j int)      { h.scores[i], h.scores[j] = h.scores[j], h.scores[i] }

func (h *scoredMaxHeap) siftUp(i int) {
	for i > 0 {
		p := (i - 1) / 2
		if h.less(p, i) {
			break
		}
		h.swap(p, i)
		i = p
	}
}

func (h *scoredMaxHeap) siftDown(i int) {
	n := len(h.scores)
	for {
		largest := i
		l, r := 2*i+1, 2*i+2
		if l < n && h.less(l, largest) {
			largest = l
		}
		if r < n && h.less(r, largest) {
			largest = r
		}
		if largest == i {
			break
		}
		h.swap(i, largest)
		i = largest
	}
}
