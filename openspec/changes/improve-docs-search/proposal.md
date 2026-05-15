## Why

Docs search (via `vectos search --docs`) takes 1.85s per query (7× slower than code search) due to linear scan over 6000+ chunks. Precision is 50% vs grep's 58% because blog posts, development-tool prompts, and other non-documentation markdown files dominate the index, burying the actual project documentation. There is no mechanism for users to control what gets indexed — exclusion rules are entirely hardcoded. Users need per-project and global configuration to define what should be excluded from indexing, with sensible defaults (`.gitignore`).

## What Changes

- Load the HNSW vector index for documentation search (currently only loaded for code search)
- Add **project-level and global configuration** for index exclusion patterns via `vectos.config.json` (project root) and `~/.vectos/config.json` (global)
- Automatically respect `.gitignore` patterns — files ignored by git are excluded from indexing by default
- Support glob patterns for both `docs` and `code` index exclusions independently
- Global config provides project-wide defaults; per-project config augments with additional patterns
- Add path-based relevance boosts: files under `docs/` and project README files get higher keyword scores
- Apply the same linear-scan threshold (<1000 chunks) that code search uses, but with HNSW for larger indexes
- Reduce docs search latency from ~1.85s to ~0.25s (matching code search performance)
- Document all recent features (model, RRF fusion, TS tags, config) in README and docs/

## Capabilities

### New Capabilities

- `docs-search-optimization`: Improve documentation search speed and relevance by loading the HNSW vector index, supporting configurable index exclusions (global + per-project), respecting `.gitignore`, applying path-based relevance boosts, and documenting all recent features.

### Modified Capabilities

- `code-indexing`: The indexing scope SHALL support user-configurable exclusion patterns via `vectos.config.json` and `~/.vectos/config.json`, in addition to hardcoded sensitive-file exclusions. Patterns are cumulative: global defaults + project overrides + `.gitignore`.
- `embedding-provider`: The global config (`~/.vectos/config.json`) gains a new `index` section for docs and code exclusion defaults.

## Impact

- **Affected code**: `cmd/vectos/commands_search.go` (HNSW for docs), `internal/content/language.go` + `paths.go` (config exclusions + gitignore), `internal/config/` (new project config loading), `internal/storage/sqlite.go` (path-based scoring)
- **Affected config**: `vectos.config.json` (new, project root), `~/.vectos/config.json` (new `index` section)
- **Dependencies**: None new
- **User impact**: Optional — existing projects work without config. Adding `vectos.config.json` enables fine-grained control. Requires reindex after changing exclusion patterns.
