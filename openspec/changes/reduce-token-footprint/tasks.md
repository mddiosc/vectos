## 1. MCP Response Format — Guidance Codes and Field Removal

- [ ] 1.1 Update `cmd/vectos/mcp_format.go`: replace guidance strings with error codes (`IDX_MISSING`, `IDX_STALE`, `IDX_ACTION:*`)
- [ ] 1.2 Update `cmd/vectos/mcp_format.go`: remove `Rank` field from `mcpSearchFileResult` struct
- [ ] 1.3 Update `cmd/vectos/mcp_format.go`: remove `FileName`, `Language`, `Category` fields from `mcpSearchFileResult` struct
- [ ] 1.4 Update `buildMCPSearchPayload()` to compute relative paths from `scope.PrimaryRoot` using `filepath.Rel`
- [ ] 1.5 Update `buildMCPSearchPayload()` to encode `Relevance` as integer (0-100) instead of float
- [ ] 1.6 Add `languageFromExtension(filename string) string` helper for client-side language inference

## 2. Embedding Enrichment Reduction

- [ ] 2.1 Update `internal/indexer/chunker.go` `buildSemanticContent()`: remove `Language:` prefix line
- [ ] 2.2 Update `internal/indexer/chunker.go` `buildSemanticContent()`: remove `Category:` prefix line
- [ ] 2.3 Update `internal/indexer/chunker.go` `classifyCategory()`: ensure function is exported or accessible for client-side reconstruction

## 3. Tests and Verification

- [ ] 3.1 Update `cmd/vectos/mcp_format_test.go`: adjust assertions for new field set (no Rank, no FileName/Language/Category)
- [ ] 3.2 Update `cmd/vectos/mcp_payload_test.go`: adjust `SearchFileResult` fixtures to match new struct
- [ ] 3.3 Update `cmd/vectos/mcp_format_test.go`: add assertions for relative path format and integer relevance
- [ ] 3.4 Run `go build ./...` and fix any compilation errors
- [ ] 3.5 Run `go test ./... -count=1` and verify all tests pass

## 4. Benchmark Verification

- [ ] 4.1 Run `vectos index .` on this project to re-index with new enrichment
- [ ] 4.2 Run `vectos benchmark benchmarks/retrieval/token-efficiency.json` before and after enrichment change to verify hit rate not degraded
- [ ] 4.3 Run `vectos benchmark benchmarks/retrieval/vectos-core.json` to verify core retrieval still works
- [ ] 4.4 Measure MCP payload sizes with `MCP_BENCHMARK_OUT` to confirm expected token reduction
- [ ] 4.5 If hit rate drops >5% on enrichment change, revert 2.1 and 2.2 and keep only guidance/field changes