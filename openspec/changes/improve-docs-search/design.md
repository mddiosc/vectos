## Context

The docs search pipeline (`executeSearchDocs` in `cmd/vectos/search_exec.go`) uses the same RRF fusion engine as code search (vector + keyword), but the initialization path lacks HNSW loading. The `runSearch` function in `commands_search.go` only calls `store.LoadVectorIndex()` when `docsOnly=false`. For docs search, the store uses a separate database (`-docs.db`) whose HNSW index is never loaded, forcing a linear scan over all 6000+ chunks.

Additionally, the documentation index includes content that semantically matches markdown but is not project documentation: blog posts (`src/content/blog/`), agent skill manifests (`.agents/skills/`), and GitHub dev-tool prompts (`.github/prompts/`). These dominate keyword scoring because they contain large amounts of English text.

### Current state

- Code search: HNSW loaded, linear scan for <1000 chunks, avg 0.25s
- Docs search: HNSW never loaded, always linear scan, avg 1.85s
- Docs index includes: `docs/`, READMEs, `.github/prompts/`, blog posts, agent skill files
- Keyword scoring: uniform for all files (no path-based weighting)

## Goals / Non-Goals

**Goals:**
- Load HNSW vector index for documentation search stores
- Apply the same <1000-chunk linear-scan threshold used in code search
- Exclude blog content and dev-tool prompts from documentation indexing
- Add path-based keyword scoring boosts for files under `docs/` and project READMEs
- Bring docs search latency from ~1.85s to ~0.25s

**Non-Goals:**
- Changing the chunking strategy for documentation files
- Separate embedding model for docs (jina-embeddings-v3 works well for both)
- Adding a dedicated docs-only ranking pipeline
- Changing the docs search API surface

## Decisions

### Decision 1: Load HNSW for docs search, same pattern as code search

Add `store.LoadVectorIndex()` call in the docs search path (`commands_search.go:47`), identical to the code search path. If the HNSW index is missing or fails to load, fall back to linear scan (same behavior as code search).

**Alternative considered**: Create a separate `openStorageForScope` variant for docs. Rejected because the existing function already handles docs via the `docsOnly` parameter — the only missing piece is the HNSW load call.

### Decision 2: Exclude blog and dev-tool directories from docs indexing

Add content directories to the directory exclusion list but only for documentation indexing (not code indexing). Blog posts in `src/content/blog/` are content, not documentation. Files in `.github/prompts/` are dev-tool configuration, not project docs.

Implementation: Use the existing `skippedDirs` map but differentiate between code-index and docs-index exclusions via a separate `skippedDocsDirs` map.

**Alternative considered**: Use a single `skippedDirs` for both. Rejected because blog directories contain source code and ARE valid for code indexing (they're not source code, but they're project content that might be semantically relevant).

### Decision 3: Path-based keyword scoring boosts for docs/ files

In `computeKeywordScore`, add a ×1.5 multiplier for chunks whose file path starts with `docs/` or whose filename is `README.md`. This makes documentation files rank higher than blog posts or prompt files in keyword search results.

**Alternative considered**: Penalize non-docs files instead. Rejected because boosting is more intuitive and avoids penalizing blog posts that might be genuinely relevant for certain queries.

## Risks / Trade-offs

- **[Blog content may be semantically relevant]** Some queries might legitimately want blog content (e.g., "React Server Components example") → mitigated by path boost, not exclusion — blog posts still appear but rank lower
- **[HNSW may degrade small-doc results]** Linear scan is more accurate for <1000 chunks → already handled by the threshold check in `SearchSemantic`
