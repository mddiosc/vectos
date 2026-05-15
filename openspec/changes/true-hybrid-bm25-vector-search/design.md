## Context

The current search pipeline (`cmd/vectos/search_exec.go:42-62`):

1. Embed query → get query vector
2. `store.SearchSemantic()` → HNSW ANN returns top 10 by cosine similarity
3. `rerankHybridResults()` → apply small heuristic boosts (+0.04 to +0.18)
4. Dedup per file (max 2) → return results

The "hybrid" portion is only step 3 — post-retrieval boosts on ANN-only candidates. SQLite FTS5 text search (`SearchText`) exists but is only used as a **fallback** when embeddings are unavailable, never as a parallel signal.

This design replaces steps 2-4 with a true dual-search + fusion pipeline.

## Goals / Non-Goals

**Goals:**
- Run BM25 keyword search **independently** from vector search for every query
- Fuse vector and keyword results using Reciprocal Rank Fusion (RRF)
- Remove the current post-retrieval boosting heuristics (replaced by RRF)
- Keep the pure text-search fallback when embeddings are unavailable
- Minimize latency impact (one extra FTS5 query, ~1-5ms)

**Non-Goals:**
- Learning-to-rank or ML-based fusion
- Weighted score combination (RRF is simpler, no score normalization needed)
- Query expansion or rewriting (separate future change)
- Changing the chunking or embedding pipeline

## Decisions

### Decision 1: Reciprocal Rank Fusion (RRF) over weighted score combination

RRF formula: `RRF_score(d) = Σ 1/(k + rank_i(d))` where `k=60` (standard constant), summed across both rankings.

**Why RRF over weighted sum of normalized scores:**
- Vector scores (cosine similarity 0-1) and BM25 scores (unbounded) have different distributions
- Normalizing them to the same scale requires assumptions about score distributions that vary per query
- RRF is parameter-free (only `k`, which is standard) and robust to score distribution differences
- RRF naturally handles the case where a document appears in only one ranking (gets partial credit)

**Alternative considered**: Min-max normalization + weighted sum. Rejected because min/max vary wildly per query and BM25 scores can have extreme outliers for exact symbol matches (e.g., `mergeProjectData` appearing verbatim gets a BM25 score 10x other results).

### Decision 2: Run both searches in parallel, limit each to top 25

- Vector search: HNSW ANN, top 25 (increased from current 10 to give RRF more candidates)
- Keyword search: SQLite FTS5 with BM25 ranking, top 25
- RRF fuses both lists into a single ranking → top 10 results returned

**Why 25**: Gives RRF enough surface area to find overlapping candidates. 10 was too few when vector and keyword disagree. 25 keeps latency low (HNSW search is O(log N)).

### Decision 3: Replace `rerankHybridResults` with `fuseAndRank`

The current `rerankHybridResults` function (~150 lines of boosts) is replaced by a simpler `fuseAndRank` that:
1. Takes two result lists (vector + keyword)
2. Computes RRF scores
3. Applies only structural penalties (test file, build artifact, help text) — not content boosts
4. Deduplicates by file (max 2 per file, same as before)

The content-matching boosts (exact phrase, token overlap, file name) are no longer needed because BM25 keyword search already captures that signal natively in its own ranking.

### Decision 4: Add `SearchTextRanked` to storage layer

The existing `SearchText` returns results without scores. A new `SearchTextRanked` method returns BM25 scores alongside results. Implementation: SQLite FTS5 `bm25()` function or `rank` column.

### Decision 5: Search mode stays `"semantic_hybrid"`

The output mode is already `"semantic_hybrid"` — no change needed. The internal behavior changes but the API surface doesn't.

## Risks / Trade-offs

- **[BM25 favors exact matches]** Queries that are natural language only (e.g., "how does authentication work") may get noisy BM25 results → RRF's k=60 dampens this; keyword results with low overlap get low reciprocal rank weight
- **[RRF doesn't use score magnitudes]** A vector result with cosine similarity 0.95 gets the same RRF contribution as one with 0.70 at the same rank → acceptable because rank position already encodes relative quality within each list
- **[Extra latency]** One additional SQLite query → <5ms, negligible vs embedding generation (~50ms) and network round-trip for remote providers
