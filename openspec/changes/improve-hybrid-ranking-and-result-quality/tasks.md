## 1. Hybrid Ranking Pipeline

- [x] 1.1 Define the hybrid ranking stages and the candidate signals to combine with semantic similarity
- [x] 1.2 Implement reranking that can apply text-aware and structural boosts without breaking current fallback behavior
- [x] 1.3 Add configuration or internal thresholds needed to tune hybrid ranking safely during evaluation

## 2. Result Quality Improvements

- [x] 2.1 Implement result deduplication or redundancy reduction for overlapping top candidates
- [x] 2.2 Prefer more actionable code entry points when file, symbol, or chunk-role signals provide stronger evidence
- [x] 2.3 Ensure CLI and MCP search paths surface the improved ranking order consistently

## 3. Validation

- [x] 3.1 Add tests for reranking and redundancy reduction behavior
- [x] 3.2 Run the retrieval benchmark suite before and after the ranking changes to validate top-result improvements
- [x] 3.3 Run `go build ./...` and verify normal semantic and text fallback behavior still works
