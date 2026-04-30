## Why

Vectos should consume fewer tokens than native agent tools (grep, glob, etc.) when providing code context. Currently, the search output is optimized for "returning good results" but not for "giving the agent the minimum information needed to act." This results in large MCP payloads with redundant metadata, truncated previews without semantic value, and complex rankings that do not save tokens for the agent.

The fundamental problem: native grep/glob returns raw text and the agent spends tokens navigating. Vectos should act as a "smart pointer" that says *"read this specific function in this file"* instead of returning 5 individual chunks with 150 tokens each.

## What Changes

- **Redesign MCP output format from chunk-level to file-level**, grouping results by file and showing function signatures instead of truncated code previews.
- **Implement token-budget preview system**: when semantic confidence is high (> 0.90), no content is sent to the agent, only path + lines + signature. When it is low, a contextual preview is sent.
- **Collapse overlapping chunks from the same file** into a single consolidated result, eliminating metadata duplication (file_path, file_name, language, category).
- **Simplify hybrid ranking**: reduce `hybridCandidateLimit` from 25 to 10 candidates and remove heuristic boosts that do not significantly move the needle, prioritizing fewer results of higher confidence.
- **Extract and persist function signatures and purposes** during chunking so they are available in output without re-parsing.

## Capabilities

### New Capabilities

(none — this is a refinement of existing output and ranking behavior)

### Modified Capabilities

- `semantic-search`: Modify ranking and deduplication requirements to prioritize token-efficient output. Reduce candidate pool (25 → 10), collapse overlapping chunks per file, simplify hybrid boosting surface.
- `mcp-interface`: Modify search result payload format from chunk-level entries to file-level summaries with signature pointers. Add token-budget preview tiers (high-confidence = pointer-only, low-confidence = contextual preview).
- `code-indexing`: Require chunk signatures and inferred purposes to be extractable and storable for direct inclusion in agent-facing output without re-parsing code at query time.

## Impact

- **Affected modules**: `cmd/vectos/mcp_format.go`, `cmd/vectos/search_output.go`, `cmd/vectos/search_ranking.go`, `internal/indexer/chunker.go`, `internal/storage/models.go`, `internal/storage/sqlite.go`.
- **MCP output format**: Breaking change in `mcpSearchResultEntry` structure — shifts from per-chunk array to per-file array with consolidated signatures/ranges.
- **CLI output**: `formatSearchResults` will also adopt file-level grouping for consistency, though less critical since CLI prints for humans, not token budgets.
- **Tests**: All ranking tests, MCP payload tests, and output formatting tests will require updates.
