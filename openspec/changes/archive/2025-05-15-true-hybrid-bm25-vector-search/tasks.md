## 1. Storage Layer — Keyword Search with Scores

- [x] 1.1 Add `SearchTextRanked(query string, limit int) ([]CodeChunk, error)` to `internal/storage/sqlite.go` — uses Go-based term-frequency scoring, returns chunks with relevance scores in the `Score` field
- [x] 1.2 Update `CodeChunk.Score` field documentation to clarify it holds either cosine similarity (from vector search) or keyword score (from keyword search), depending on source
- [x] 1.3 Add unit test: `SearchTextRanked` returns results with non-zero scores
- [x] 1.4 Add unit test: `SearchTextRanked` returns empty slice (not error) for no matches

## 2. RRF Fusion Engine

- [x] 2.1 Create `cmd/vectos/search_fusion.go` with `fuseResults(vectorResults, keywordResults []storage.CodeChunk, k float64) []storage.CodeChunk`
- [x] 2.2 Implement Reciprocal Rank Fusion: `RRF_score = 1/(k + rank_in_list)` summed across both lists, using k=60
- [x] 2.3 Handle deduplication by chunk ID: if a chunk appears in both lists, sum both RRF contributions
- [x] 2.4 Handle edge case: one or both input lists are empty → return whichever has results
- [x] 2.5 Add unit test: RRF correctly fuses two result lists with known ranks
- [x] 2.6 Add unit test: chunk appearing in both lists gets both RRF contributions
- [x] 2.7 Add unit test: empty keyword list doesn't break fusion (vector-only works)

## 3. Post-Fusion Penalties

- [x] 3.1 Create `applyFusionPenalties(results []storage.CodeChunk) []storage.CodeChunk` in `cmd/vectos/search_fusion.go`
- [x] 3.2 Apply test file penalty (-0.08) to results from test files (`*_test.go`, `*.test.*`, `/test/`)
- [x] 3.3 Apply build artifact penalty (-0.25) to results from `/dist/`, `/coverage/`, `/build/`, `/.next/`
- [x] 3.4 Apply help text penalty (-0.10) to results matching help text patterns
- [x] 3.5 Verify: no content-matching boosts (exact phrase, token overlap, file name) are applied post-fusion
- [x] 3.6 Add unit test: test file chunk receives penalty in fused results
- [x] 3.7 Add unit test: source code chunk does NOT receive penalty

## 4. Search Pipeline Integration

- [x] 4.1 Refactor `trySemanticSearch()` in `cmd/vectos/search_exec.go` to:
  - Run `SearchSemantic()` for vector results (top 25)
  - Run `SearchTextRanked()` for keyword results (top 25)
  - Call `fuseResults()` + `applyFusionPenalties()`
  - Apply per-file dedup (max 2 per file) on fused results
  - Return top 10
- [x] 4.2 Update `textSearchFallback()` to use `SearchTextRanked()` when available
- [x] 4.3 Remove `rerankHybridResults()` and associated boost constants from `cmd/vectos/search_ranking.go`
- [x] 4.4 Keep helper functions that are still needed: `isTestFilePath`, `isBuildArtifactPath`, `looksLikeHelpText`, dedup logic, range overlap, token utilities
- [x] 4.5 Update search mode output — stays `"semantic_hybrid"` when fusion succeeds, `"text"` when only keyword available

## 5. Docs Search Integration

- [x] 5.1 Update `executeSearchDocs()` in `cmd/vectos/search_exec.go` to also benefit from fusion (vector + keyword for docs)
- [x] 5.2 Verify docs search still works correctly with the new pipeline

## 6. Removal of Legacy Boosting

- [x] 6.1 Remove `computeHybridScore()` function
- [x] 6.2 Remove `applyContentBoosts()` function
- [x] 6.3 Remove boost constants: `hybridExactPhraseBoost`, `hybridTokenOverlapWeight`, `hybridFileNameBoost`, `hybridActionableCodeBoost`, `hybridFallbackBoost`, `hybridBroadQueryPenalty`
- [x] 6.4 Remove `tokenOverlapRatio()` if no longer used elsewhere
- [x] 6.5 Remove `looksActionableCode()` and `looksLikeSemanticFallback()` if no longer used

## 7. Tests

- [x] 7.1 Add integration test: run search with fusion → verify both vector and keyword contribute to results
- [x] 7.2 Add integration test: search for exact symbol name → keyword match appears in top results
- [x] 7.3 Add integration test: natural language query → vector match still appears in top results
- [x] 7.4 Add integration test: query with no keyword matches → returns vector-only results
- [x] 7.5 Run existing test suite (`go test ./...`) to verify no regressions
- [x] 7.6 Verify `TestVectorSearchFallbackWhenNoIndex` still passes with new pipeline
