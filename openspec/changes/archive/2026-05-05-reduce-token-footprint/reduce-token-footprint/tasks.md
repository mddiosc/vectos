## 1. MCP Response Format — Guidance Codes and Field Removal

- [x] 1.1 Update `cmd/vectos/mcp_format.go`: replace guidance strings with error codes (`IDX_MISSING`, `IDX_STALE`). Keep `next_action` unchanged (still carries the actual command).
- [x] 1.2 Update `cmd/vectos/mcp_format.go`: remove `Rank` field from `mcpSearchFileResult` struct and its assignment in `buildMCPSearchPayload()`
- [x] 1.3 Update `cmd/vectos/mcp_format.go`: remove `FileName`, `Language`, `Category` fields from `mcpSearchFileResult` struct and their assignments
- [x] 1.4 Update `buildMCPSearchPayload()` to compute relative paths from `scope.PrimaryRoot` using `filepath.Rel`. Fall back to absolute path if `filepath.Rel` returns an error.
- [x] 1.5 Change `Relevance` field type in `mcpSearchFileResult` from `float64` to `int`. In `buildMCPSearchPayload()`, convert via `int(math.Round(fr.Relevance * 100))`.
- [x] 1.6 Update `cmd/vectos/mcp_server.go`: update `search_code` tool description to document error codes (`IDX_MISSING`, `IDX_STALE`) in guidance field.

## 2. Embedding Enrichment Reduction

- [x] 2.1 Update `internal/indexer/chunker.go` `buildSemanticContent()`: remove `Language:` prefix line
- [x] 2.2 Update `internal/indexer/chunker.go` `buildSemanticContent()`: remove `Category:` prefix line

## 3. Tests and Verification

- [x] 3.1 Update `cmd/vectos/mcp_format_test.go`: adjust assertions for new field set (no Rank, no FileName/Language/Category, integer relevance, relative paths)
- [x] 3.2 Update `cmd/vectos/mcp_payload_test.go`: adjust `SearchFileResult` fixtures — note: `storage.SearchFileResult` keeps its fields, only `mcpSearchFileResult` output changes
- [x] 3.3 Run `go build ./...` and fix any compilation errors
- [x] 3.4 Run `go test ./... -count=1` and verify all tests pass

## 4. Benchmark Verification

- [x] 4.1 Run `vectos index .` on this project to re-index with new enrichment
- [x] 4.2 Run `vectos benchmark benchmarks/retrieval/token-efficiency.json` before and after enrichment change to verify hit rate not degraded
- [x] 4.3 Run `vectos benchmark benchmarks/retrieval/vectos-core.json` to verify core retrieval still works
- [x] 4.4 Measure MCP payload sizes with `MCP_BENCHMARK_OUT` to confirm expected token reduction
- [x] 4.5 If hit rate drops >5% on enrichment change, revert 2.1 and 2.2 and keep only guidance/field changes (NOT TRIGGERED — no degradation detected, 100% hit rate on both benchmarks)