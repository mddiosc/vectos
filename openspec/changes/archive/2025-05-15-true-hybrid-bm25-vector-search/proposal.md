## Why

The current "hybrid" search is misnamed — it runs ANN vector search alone and applies cosmetic post-retrieval boosts (+0.04 to +0.18). BM25 keyword matching never contributes its own independent candidates. This means structural queries ("custom React hooks", "TypeScript interfaces", symbol names like `mergeProjectData`) fail when the embedding model doesn't capture code syntax well. True fusion of vector + keyword search will recover these failures and improve precision from ~60% to ~75-80%.

## What Changes

- Run BM25/FTS5 keyword search **independently** alongside ANN vector search for every query
- Fuse both result sets using **Reciprocal Rank Fusion (RRF)** to produce a unified ranking
- Remove the current post-retrieval boosting heuristics (`rerankHybridResults`) — RRF replaces them
- Keep semantic-only fallback when keyword search yields no results, and vice versa
- Preserve the existing text-only fallback when embedding provider is unavailable
- The search mode becomes `"semantic_hybrid"` (already used, now accurate)

## Capabilities

### New Capabilities

- `keyword-vector-fusion`: Combine BM25 full-text keyword search results with ANN vector search results using Reciprocal Rank Fusion to produce a unified, higher-quality ranking that leverages both syntactic and semantic relevance signals.

### Modified Capabilities

- `semantic-search`: The search pipeline SHALL run both vector and keyword searches independently and fuse results, rather than running vector search alone with post-retrieval boosts. The existing fallback behavior (text-only when embeddings unavailable) is preserved. The candidate pool limit and deduplication logic are adapted to work with fused results.

## Impact

- **Affected code**: `cmd/vectos/search_exec.go` (new dual-search + fusion flow), `cmd/vectos/search_ranking.go` (replace boosts with RRF, keep dedup/penalties), `internal/storage/sqlite.go` (SearchText is already available, may need `SearchTextRanked` returning BM25 scores)
- **Affected config**: None — no new configuration needed
- **Dependencies**: None new — SQLite FTS5 is already used for text search
- **Performance**: Adds one extra SQLite FTS5 query per search (~1-5ms), negligible vs embedding generation (~10-50ms)
- **User impact**: No migration needed. Search results improve immediately for all queries
