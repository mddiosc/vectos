## Why

Docs search (via `vectos search --docs`) takes 1.85s per query (7× slower than code search) due to linear scan over 6000+ chunks. Precision is 50% vs grep's 58% because blog posts and development-tool prompts dominate the index, burying the actual project documentation. The docs search path was overlooked when HNSW loading and directory exclusions were implemented for code search.

## What Changes

- Load the HNSW vector index for documentation search (currently only loaded for code search)
- Exclude blog content (`src/content/blog/`) and agent/dev-tool prompts (`.github/prompts/`, `.agents/`) from the documentation index
- Add path-based relevance boosts: files under `docs/` and project README files get higher keyword scores
- Apply the same linear-scan threshold (<1000 chunks) that code search uses, but with HNSW for larger indexes
- Reduce docs search latency from ~1.85s to ~0.25s (matching code search performance)

## Capabilities

### New Capabilities

- `docs-search-optimization`: Improve documentation search speed and relevance by loading the HNSW vector index, excluding non-documentation content from the docs index, and applying path-based relevance boosts for documentation files.

### Modified Capabilities

- `code-indexing`: The documentation indexing scope SHALL exclude content directories (`src/content/`, especially blogs) and development-tool prompts (`.github/prompts/`) that contain markdown files unrelated to project documentation.

## Impact

- **Affected code**: `cmd/vectos/commands_search.go` (HNSW loading for docs store), `internal/content/language.go` (docs-specific exclusion patterns), `internal/storage/sqlite.go` (path-based scoring adjustments)
- **Affected config**: None
- **Dependencies**: None new
- **User impact**: Requires reindex of docs (`vectos index --docs`) to apply new exclusions. Search results improve immediately with no API changes.
