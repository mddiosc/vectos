## 1. Schema and Storage Layer

- [ ] 1.1 Add `signature TEXT` and `purpose TEXT` columns to `code_chunks` table in `internal/storage/sqlite.go`'s `migrate()` function, following the existing `category` migration pattern.
- [ ] 1.2 Update `internal/storage/models.go` to add `Signature` and `Purpose` fields to the `CodeChunk` struct.
- [ ] 1.3 Update `internal/storage/sqlite.go` `SaveChunk()` to bind the new `signature` and `purpose` columns.
- [ ] 1.4 Update `internal/storage/sqlite.go` `SearchSemantic()` and `SearchText()` to scan the new columns into `CodeChunk`.
- [ ] 1.5 Update `internal/storage/sqlite.go` `DeleteChunksByPath()` and `DeleteChunksByPathPrefix()` signatures if needed (should be fine as-is).

## 2. Chunking and Indexing

- [ ] 2.1 Update `internal/indexer/chunker.go` `ChunkResult` struct to include `Signature` and `Purpose` strings.
- [ ] 2.2 Ensure `buildChunk()` in `internal/indexer/chunker.go` captures the extracted signature via the existing `extractSignature()` logic and purpose via `inferPurpose()`.
- [ ] 2.3 Update `cmd/vectos/commands_index.go` to populate `Signature` and `Purpose` when constructing `storage.CodeChunk` before saving.
- [ ] 2.4 Update `cmd/vectos/mcp_handlers.go` `makeIndexProjectHandler` to populate the same fields when constructing `storage.CodeChunk`.

## 3. Ranking Simplification

- [ ] 3.1 Reduce `hybridCandidateLimit` constant in `cmd/vectos/search_ranking.go` from 25 to 10.
- [ ] 3.2 Reduce `hybridResultLimitPerFile` if appropriate (keep at 2; the file-level collapse handles this after). 
- [ ] 3.3 Remove the fine-grained per-intent boost functions from `cmd/vectos/search_ranking.go`: `genericIntentBoost()`, `configSpecificBoost()`, `databaseSpecificBoost()`, `seoSpecificBoost()`, `routingNameSignalBoost()`, plus all constants no longer used.
- [ ] 3.4 Remove or merge `hybridSourcePathBoost`, `hybridBroadQueryPenalty`, `hybridConfigIntentBoost`, `hybridConfigPathBoost`, `hybridApiRoutePenalty`, `hybridDbIntentBoost`, `hybridDbPathBoost`, `hybridGenericConfigPenalty`, `hybridUiIntentBoost`, `hybridSeoIntentBoost`, `hybridSeoHeadBoost`, `hybridSeoPagePenalty`, `hybridFormIntentBoost`, `hybridStateIntentBoost`, `hybridAuthIntentBoost`, `hybridDataIntentBoost` constants. Keep: exact phrase, token overlap, file name overlap, actionable code boost, fallback boost, broad query penalty, test penalty, build artifact penalty, help text penalty.
- [ ] 3.5 Update `cmd/vectos/search_ranking_test.go` to remove tests for deleted intent boosts and adjust expectations for the reduced candidate pool.

## 4. File-Level Output and Chunk Collapse

- [ ] 4.1 Add `internal/storage/search_result.go` (or extend existing) with a new `SearchFileResult` struct containing: `FilePath`, `FileName`, `Language`, `Category`, `Relevance float64`, `LineRanges []struct{Start, End int}`, `Signatures []string`, `Hint string`.
- [ ] 4.2 Implement a `CollapseFileResults(chunks []CodeChunk, lineWindow int) []SearchFileResult` helper in `internal/storage/` or `cmd/vectos/`. Overlapping or touching chunks (within 5 lines) are merged; signatures deduplicated; relevance = max score.
- [ ] 4.3 Update `cmd/vectos/search_output.go` `formatSearchResults()` to accept `[]SearchFileResult` instead of `[]CodeChunk`, print file-level summaries with signatures and line ranges.
- [ ] 4.4 Update `cmd/vectos/mcp_format.go` `mcpSearchResultEntry` struct to `mcpSearchFileResult` with the new fields, and update `buildMCPSearchPayload()` to construct it from collapsed results.

## 5. Token-Budget Preview System

- [ ] 5.1 Define a `HighConfidenceThreshold` constant (default 0.90) in `cmd/vectos/search_output.go`.
- [ ] 5.2 Implement `buildHint(result SearchFileResult, query string) string` that returns a concise contextual string when relevance < threshold (using the stored `Purpose` field or category fallback).
- [ ] 5.3 Update `buildMCPSearchPayload()` so that `Preview` is omitted (or left empty) when `Relevance >= threshold`, and `Hint` is populated only when `Relevance < threshold`.
- [ ] 5.4 Update `formatSearchResult()` for CLI output: skip code preview entirely when score >= threshold (print pointer only), otherwise show compact hint.

## 6. MCP Output and Payload Tests

- [ ] 6.1 Update `cmd/vectos/mcp_payload_test.go` to assert the new `mcpSearchFileResult` shape instead of the old per-chunk structure.
- [ ] 6.2 Update `cmd/vectos/mcp_format.go` `mcpSearchPayload` struct: change `Results` field type from `[]mcpSearchResultEntry` to `[]mcpSearchFileResult`.
- [ ] 6.3 Update `cmd/vectos/search_output_test.go` to assert file-level formatting, signature display, and threshold-based hint suppression.

## 7. End-to-End Verification

- [ ] 7.1 Run `go test ./...` and fix compilation errors across all packages.
- [ ] 7.2 Run `go test ./cmd/vectos/...` specifically for ranking, output, and MCP format tests.
- [ ] 7.3 Run a local test index (`vectos index .`) on this project to verify `signature` and `purpose` columns are populated correctly.
- [ ] 7.4 Run a local test search (`vectos search "checkout payment"`) and inspect output format for file-level grouping and signature display.
- [ ] 7.5 Verify `vectos status` still works and reports the correct chunk/file counts.
