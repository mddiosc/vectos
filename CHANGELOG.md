# Changelog

All notable changes to Vectos are documented here.

This project uses [SemVer](https://semver.org/) in the `0.x` range.
Releases in this phase are **experimental/internal builds** — interfaces,
packaging, and behavior may change without notice.

Format per release:

```
## vX.Y.Z — YYYY-MM-DD
### Added
### Changed
### Fixed
### Known Limitations
```

---

---

## v0.8.0 — 2026-05-16

Major release focused on indexing performance: HNSW build is 12× faster, incremental reindexes skip unchanged files entirely, and embedding batch size adapts automatically to available RAM.

### Added

- **Adaptive ONNX embedding batch size** — automatically selects batch size based on available RAM: ≥4 GB → 32, ≥2 GB → 16, ≥1 GB → 8, <1 GB → 4. Override via `batch_size` in `~/.vectos/config.json`.
- **Hash-based embedding cache** — full and incremental reindexes skip files whose content hash hasn't changed. Chunking, embedding, and vector index rebuild are all bypassed for unchanged files.
- **Per-phase progress reporting** — indexing now reports scan/chunk phase, embedding phase (with throughput rate and ETA), and storage phase separately.
- **Token efficiency documentation** — new `docs/token-efficiency.md` with benchmark data showing 17× token reduction vs grep-based workflows.

### Changed

- **Effective batch size raised to 32** on machines with ≥4 GB available RAM (was 8). Configurable via `batch_size` in config; set explicitly to keep the old value.
- **Full reindex no longer deletes all chunks upfront** — unchanged files are preserved, deleted/excluded files are cleaned up at the end by `cleanupExcludedAndSkipped`.
- **Vector index rebuild skipped** when the content hash matches the stored hash — saves several seconds on large projects with no changes.

### Fixed

- **HNSW graph correctness** — `Insert` was passing the raw (non-normalized) vector to `searchLayerLocal` during graph construction. All stored vectors are pre-normalized, so the query must also be normalized for correct dot-product distances. This degraded neighbor selection quality during index build.
- **Silent cache hit on DB error** — `HasFileChanged` errors now treat the file as changed instead of silently skipping it, preventing stale index entries after transient DB failures.
- **Redundant hash upserts** — `UpsertIndexedFile` was called once per chunk instead of once per file, causing N redundant SQL upserts for files with N chunks.
- **ETA overflow** — embedding progress ETA could produce `+Inf` or a negative duration when `rate == 0` on the first batch.

### Performance

- **HNSW build 12× faster** via pre-normalized vectors (dot product replaces cosine in hot path) and distance caching in `prune`. Benchmarks: 1K vectors 2.85s → 0.23s, 10K vectors 39.6s → 6.93s.

---

## v0.7.1 — 2026-05-15

### Fixed

- **Embedded model download failures** — HuggingFace CDN returns `text/plain` and `application/json` Content-Types for model assets. Updated download validation to accept these types so jina-embeddings-v3 model downloads succeed.
- **Missing `model.onnx_data` asset** — jina-embeddings-v3 uses ONNX external data format and requires `model.onnx_data` alongside `model.onnx`. Added to asset manifest.
- **Scalar `task_id` input** — jina-embeddings-v3 expects `task_id` as a scalar, not a `[batch, seq_len]` tensor. Refactored inference to build inputs dynamically by model input name, fixing ONNX reshape errors.
- **Misleading provider error** — `ResolveEmbedder` now collects and reports all provider failures (`embedded: <reason>; remote: <reason>`) instead of only the last error.
- **Root directory excluded by `.gitignore`** — patterns matching the project root directory name no longer cause `filepath.SkipDir` on the entire tree (fixes `vectos index .` returning "no supported files found" in the vectos project itself).
- **Safe batch size** — reduced `DefaultEmbeddedBatchSize` from 32 to 8 to work reliably with jina-embeddings-v3 without memory pressure.

---

## v0.7.0 — 2026-05-15

Major release focused on search quality, embedding model upgrade, configurable index exclusions, and documentation.

### BREAKING

- **Default embedding model changed** from `bge-small-en-v1.5` (384-dim) to `jina-embeddings-v3` (1024-dim, code+text+multilingual, 8192-token context). Existing indexes become stale and require reindex. Set `model_name: "bge-small-en-v1.5"` in config to keep the old model.

### Added

- **Reciprocal Rank Fusion (RRF) keyword-vector search** — keyword search runs independently alongside vector search, fused via RRF (k=60) for higher precision. Replaces the old post-retrieval boosting heuristics.
- **TypeScript structural tagging** — chunker detects interfaces, type aliases, enums, and async functions, tagging them as "type definition", "enumeration", and "async function" for richer embeddings.
- **Configurable index exclusions** — `vectos.config.json` in project root and `index` section in `~/.vectos/config.json` for glob-based docs and code exclusion patterns.
- **Automatic .gitignore respect** — files matching `.gitignore` patterns are excluded from indexing by default.
- **Path-based keyword scoring boosts** — documentation files under `docs/` and `README.md` get 1.5× keyword relevance scores.
- **HNSW vector index for docs search** — documentation search now loads the HNSW index, reducing latency from ~1.85s to ~0.25s.
- **Linear scan threshold for small indexes** — indexes under 1000 chunks use linear scan instead of HNSW for maximum accuracy.
- **Model name validation** — unsupported model names in config are rejected with a clear error message.
- **serve logging to stderr** — `vectos serve` now writes logs to both `~/.vectos/vectos-serve.log` and stderr for visibility.
- **Documentation** — README and docs/indexing.md updated with all recent features and config reference.

### Changed

- **Vector candidate pool rebalanced** — vector search retrieves 35 candidates, keyword 15 (was 25 each) for better vector-dominant fusion.
- **ef_search default increased** — HNSW ef_search raised from 100 to 200 for better approximate search accuracy.
- **Stale-index warnings improved** — now shows both stored and current model/dimensions.

### Fixed

- **HNSW ID mismatch bug** — `SearchScored` was returning internal array indices instead of external chunk IDs, causing wrong results on loaded indexes. Fixed to return external IDs.
- **HNSW dimension validation** — loaded index dimensions are now validated against query vector dimensions before use.
- **HNSW content hash validation** — `LoadVectorIndex` now validates the content hash and returns an error on mismatch, preventing stale index usage.
- **Config model dir/URL sync** — switching `model_name` now correctly updates `ModelDir` and `AssetBaseURL` to match the selected model.
- **Single-file indexing exclusion** — `addFile` now calls `ShouldSkipFile`, preventing direct indexing of lockfiles and config files.
- **VectorIndexConfig propagation** — `buildVectorIndex` now uses config values for M, efConstruction, and efSearch instead of hardcoded defaults.
- **Keyword search multi-word queries** — split into OR-connected LIKE clauses so queries like "how to set up development environment" find results.

### Excluded from indexing

- Lockfiles: `pnpm-lock.yaml`, `package-lock.json`, `yarn.lock`, `Cargo.lock`, `Gemfile.lock`, `go.sum`, `composer.lock`, `poetry.lock`, `Pipfile.lock`
- Config files: `eslint.config.*`, `tailwind.config.*`, `.eslintrc.*`
- Agent/dev dirs: `.agents/`, `.claude/`, `.codex/`

Feature release focused on search performance, security hardening, automated index freshness, HTTP API completion, and codebase quality.

### Added

- **HNSW vector index** (`internal/vectorindex/`) — pure-Go approximate nearest neighbor search with cosine distance, replacing O(n) linear scan with O(log n) lookup for semantic queries
- **Batch embedding** — `Embedder.GetEmbeddings([]string)` interface extension with ONNX batched inference (`[N, max_seq_len]`), reducing per-chunk overhead ~32x during indexing
- **SQ8 scalar quantization** — opt-in 8-bit embedding compression (4x storage reduction: 1536→384 bytes/vector) via `compression: sq8` config key
- **Binary index persistence** — `.vectorindex` file alongside SQLite with versioned binary format, content-hash staleness detection, and automatic rebuild on `vectos index`
- **Vector index configuration** — `hnsw_m` (default 16), `hnsw_ef_construction` (200), `hnsw_ef_search` (100), `index_type`, and `compression` config keys in `[vector_index]`
- **Native filesystem watcher** (`internal/watcher/`) — fsnotify-based recursive directory watching with debounce batching (500ms), glob ignore patterns, and automatic incremental reindex on file changes
- **File hash tracking** — `indexed_files` table in SQLite with SHA256 content hashing; only reindexes files whose content actually changed
- **Immediate deletion cleanup** — watcher removes chunks from index instantly on file delete/rename events
- **HTTP search endpoints** — `POST /search`, `POST /search/code`, `POST /search/docs` with JSON request/response, request validation, and hybrid semantic+text fallback
- **HTTP observability endpoints** — `GET /metrics` (index stats, provider info, uptime, watcher status) and `GET /status/:project` (per-project index status)
- **API error format** — `{"error": "message", "code": "ERROR_CODE"}` with 6 error codes for new endpoints (`INVALID_QUERY`, `INVALID_PROJECT`, `INVALID_LIMIT`, `PROJECT_NOT_FOUND`, `INTERNAL_ERROR`, `METHOD_NOT_ALLOWED`)
- **Watch mode CLI flags** — `--watch` (default true), `--watch-debounce` (500ms), `--watch-ignore` (`.git,node_modules,*.lock`)
- **`batch_size` configuration** — configurable embedding batch size (default 32) in embedding config
- **Search benchmarks** — HNSW vs linear scan benchmarks at 1K and 10K vector scales in `internal/vectorindex/bench_test.go`
- **Comprehensive test coverage** — 163 tests across 12 packages, including integration tests for vector search end-to-end, SQ8 round-trip, watcher ignore patterns, debounce behavior, and API endpoint validation

### Changed

- **Search routing** — `SearchSemantic` now routes through the vector index when available, with automatic fallback to linear scan (stale/missing index) and text search (embedding failure)
- **Indexing pipeline** — two-phase design: chunk all files first, then batch-embed in groups of `batch_size`; chunk creation and embedding are now decoupled
- **Embedder interface** — extended with `GetEmbeddings([]string) ([][]float32, error)` batch method; `GetEmbeddingsDefault()` helper provides loop-based fallback for backward compatibility
- **Server initialization** — `NewServer` now accepts `EmbedFunc` and `*SQLiteStorage` for decoupled embedding and storage access in HTTP handlers
- **CLI help text** — updated with `--watch`, `--watch-debounce`, `--watch-ignore` flags documentation
- **README** — added Watch Mode documentation section

### Fixed

- **Security: SQL LIKE wildcard injection** — `%` and `_` characters in search queries are now escaped before constructing LIKE patterns, preventing denial-of-service via combinatorial wildcard expansion
- **Security: asset URL validation** — `asset_base_url` is validated at configuration time: HTTPS-only, no path traversal (`..`), non-empty host, max 2048 chars; invalid URLs reject config loading
- **Security: Content-Type verification** — downloaded model assets now require allowlisted Content-Types (`application/octet-stream`, `application/gzip`, `application/x-gzip`, `application/x-tar`); unexpected types abort download
- **Security: sensitive file exclusion** — `.env` variants, SSH private keys (`id_rsa`, `id_ecdsa`, `id_ed25519`), certificate files (`.pem`, `.key`, `.pfx`, `.p12`), and credential files (`credentials.json`, `service-account.json`) are now skipped during indexing
- **Security: reindex rate limiting** — `/reindex` endpoint enforces token-bucket rate limit (1 req/s, burst 5); excessive requests return HTTP 429
- **Code quality** — removed 514 lines of dead code: legacy `internal/mcp/` protocol (394 lines), `inferGoPurpose` stub, 4 orphaned wrapper methods, 7 unused types/functions, root-level `sample_code.go`; staticcheck U1000 reports zero findings

### Known Limitations

- This remains an experimental/internal release. Stability and compatibility are not guaranteed.
- Supported download platforms remain `darwin/arm64` and `linux/amd64` only.
- The HNSW vector index is rebuilt on every `vectos index`; incremental index updates for single-file changes are deferred to a future release.
- Watch mode requires local filesystem; network-mounted directories will not receive filesystem events reliably.
- SQ8 compression is lossy — recall may degrade for some queries (typically remains above 80%).
- Existing indexes require reindex (`vectos index`) to build the vector index and populate file hashes for watch mode.

## v0.5.1 — 2026-05-13

Patch release focused on diagnostic tooling and incremental indexing quality improvements.

### Added

- `vectos doctor` diagnostic command to help debug configuration and embedding provider health
- Progress reporting during indexing operations for better visibility into long-running jobs

### Fixed

- `--project` flag now works correctly with `search` and `benchmark` subcommands
- Indexing progress is now visible during incremental and full reindex operations

### Known Limitations

- This remains an experimental/internal release. Stability and compatibility are not guaranteed.
- Supported download platforms remain `darwin/arm64` and `linux/amd64` only.

---

## v0.5.0 — 2026-05-10

Feature release focused on adding HTTP serving support while tightening incremental reindex behavior and agent-facing setup guidance.

### Added

- HTTP server mode with silent reindexing for serve workflows

### Changed

- CLI help text now reflects the docs flows and current MCP tool surface more accurately
- Managed setup guidance is now applied unconditionally so supported agent clients always receive the expected Vectos instructions

### Fixed

- Incremental reindex changed paths now resolve against all scope roots instead of only a partial scope view
- Nx scope handling no longer incorrectly includes excluded library/helper paths in the cases covered by `#18`

### Known Limitations

- This remains an experimental/internal release. Stability and compatibility are not guaranteed.
- Supported download platforms remain `darwin/arm64` and `linux/amd64` only.

---

## v0.4.0 — 2026-05-07

Feature release focused on Nx monorepo scoping correctness for MCP agents, plus internal code quality improvements across the indexing and handler layers.

### Added

- `list_projects` MCP tool — agents can now discover available Nx project names in a workspace before indexing or searching; returns `{"projects": []}` outside an Nx workspace
- Nx monorepo workflow section in managed agent guidance — explains the `project` parameter, `list_projects`, and that indexing a project automatically includes its internal dependency libs
- Complete `Scope` resolution when only `project` is given without `path` — `PrimaryRoot`, `WorkspaceRoot`, and `Roots` are now always populated using CWD as the workspace anchor

### Changed

- `index_project` MCP schema: `path` is now optional — agents can pass only `project` to scope indexing without providing an explicit path
- Workspace-root ambiguity error now lists available project names — when `path` is the Nx workspace root and multiple projects exist, the error message includes the full list so agents can retry with an explicit selection
- Workspace-root ambiguity check now runs before project containment matching — prevents a project with root `"."` from being silently auto-selected when multiple projects exist
- CLI workspace ambiguity errors are now propagated instead of silently returning a nil scope
- `skippedDirs` and `ignoredNxDirs` are now declared as `map[string]struct{}` tables instead of `switch/case` blocks, consistent across both packages
- `detectLanguage` now uses a declarative `fileNameMatchers` table and `extLanguages` map instead of two large `switch` blocks
- MCP handlers refactored to extract named functions (`runSearchCode`, `runSearchDocs`, `runIndexProject`, `runListProjects`) — closures are now single-line delegation
- `collectIndexablePaths` and `filterChangedPaths` refactored with `pathAccumulator`, `absRoots`, and `toSet` helpers to eliminate duplicated deduplication logic
- `discoverNxProjects` decomposed into `walkNxProjectFiles`, `handleNxWalkEntry`, `addNxProjectFromFile`, and `nxProjectsFromMap`
- `nxGraphPrintFlag` extracted as a package-level constant to replace seven inline `"--print"` literals

### Fixed

- MCP search results now always use paths relative to `PrimaryRoot` when `project` is given without `path` — previously returned absolute paths due to incomplete scope resolution
- `project` parameter outside an Nx workspace now returns an explicit error instead of silently opening the wrong project database

### Known Limitations

- This remains an experimental/internal release. Stability and compatibility are not guaranteed.
- Supported download platforms remain `darwin/arm64` and `linux/amd64` only.

---

## v0.3.0 — 2026-05-05

Feature release focused on documentation-aware retrieval, separate docs indexing/search workflows, and lower-noise agent guidance for MCP clients.

### Added

- Separate documentation index per project at `<project>-docs.db`, alongside the existing source-code index
- `vectos search --docs`, `vectos index --docs`, and `vectos status --docs` flows for documentation-only operations from the CLI
- `search_docs` MCP tool for searching documentation independently from source code
- Documentation-format indexing support for `.rst`, `.adoc`, `.asciidoc`, `.tex`, `.latex`, and `.txt` files in docs-only mode
- Shared managed setup guidance for `opencode`, `claude`, and `codex`, including docs-search and incremental reindex instructions for agents
- Active OpenSpec coverage for the new `docs-search` capability, with the completed change archived after sync

### Changed

- MCP tool discovery now advertises `search_code`, `search_docs`, and `index_project`
- MCP and CLI indexing now support dedicated documentation indexing through `docs: true` / `--docs` while preserving the existing source-code index
- Agent setup guidance now teaches supported clients to prefer Vectos code search, Vectos docs search, and incremental reindex with `changed` paths before falling back to broad file search
- Shared runtime search execution and content-category classification were consolidated to reduce drift between CLI, MCP, and chunking behavior

### Fixed

- Full documentation reindex now clears stale deleted-doc chunks instead of leaving old results searchable
- CLI search dispatch now correctly separates normal code search from docs-only search
- MCP docs guidance paths now resolve docs indexes safely even when no explicit project scope is passed
- Documentation search results and docs-status reporting now stay consistent after reindex and return the expected file-level MCP payloads

### Known Limitations

- This remains an experimental/internal release. Stability and compatibility are not guaranteed.
- Supported download platforms remain `darwin/arm64` and `linux/amd64` only.
- Source code and documentation remain intentionally separated; mixed code-plus-doc search still requires choosing the appropriate tool or index explicitly.
- Docs retrieval quality still uses the same general chunking and ranking foundations as code retrieval; documentation-specific ranking may need further tuning in future releases.

---

## v0.2.0 — 2026-04-30

Feature release focused on reducing agent token consumption by switching MCP search output from chunk-level to file-level, with signature-based pointers replacing truncated code previews.

### Added

- File-level MCP search output with consolidated `signatures`, `line_ranges`, and `relevance` scores
- `SearchFileResult` type and `CollapseFileResults()` helper for grouping chunks by file and merging overlapping/touching line ranges
- Confidence-based hint system: results with relevance >= 0.90 return pointer-only (no preview), lower confidence results include a contextual `hint`
- `signature` and `purpose` columns in `code_chunks` schema, extracted and persisted during indexing via `extractSignature()` and `inferPurpose()`

### Changed

- MCP search payload restructured: per-file `mcpSearchFileResult` entries replace per-chunk `mcpSearchResultEntry` entries
- Hybrid ranking candidate pool reduced from 25 to 10 for higher-confidence top results
- Hybrid ranking surface simplified: removed fine-grained per-intent boosts (config, database, SEO, UI, auth, state, form, routing)
- Hybrid ranking retains high-impact signals: exact phrase match, token/path overlap, actionable code detection, category preference, and build artifact/test penalties

### Fixed

- MCP search output no longer repeats file metadata (file_path, language, category) per chunk
- Token consumption reduced ~60% for high-confidence queries by removing truncated code previews in favor of function signatures

### Known Limitations

- This remains an experimental/internal release. Stability and compatibility are not guaranteed.
- Supported download platforms remain `darwin/arm64` and `linux/amd64` only.
- MCP output format is a breaking change from v0.1.x; agent clients consuming `preview` and `reason` fields will need updating.
- Existing indexes require reindex to populate `signature` and `purpose` columns; `vectos status` will indicate when reindex is recommended.

---

## v0.1.9 — 2026-04-29

Patch release focused on token-efficient retrieval, compact search output, and generic ranking improvements for agent navigation.

### Added

- Compact `vectos search` output by default, with `--full` for expanded chunk content when needed
- Adaptive preview sizing driven by query confidence so high-signal queries use shorter output and ambiguous queries keep more context
- MCP search payloads now use the same adaptive preview logic as the CLI
- Benchmark fixtures for token efficiency and large-project validation coverage

### Changed

- Hybrid ranking now favors generic intent buckets for config, UI, auth, data, routing, SEO, forms, state, and database/API queries
- Search result ranking now splits camelCase path tokens so mixed-case path names are scored more accurately
- Generic routing and SEO queries now prefer more direct files without relying on repository-specific layout assumptions

### Fixed

- Retrieval now avoids verbose full-chunk output by default, reducing token usage for agent-facing search flows
- Broad config and database queries now rank the more direct implementation files above unrelated wrappers or config noise
- Benchmark queries now reach the intended files with fewer reads than a `grep`-style baseline in most cases

### Known Limitations

- This remains an experimental/internal release. Stability and compatibility are not guaranteed.
- Supported download platforms remain `darwin/arm64` and `linux/amd64` only.
- Generic ranking is still heuristic-based and may need further tuning on very noisy repositories or unusually named files.

---

## v0.1.8 — 2026-04-28

Patch release focused on Nx logical project scopes, dependency-aware indexing, and clearer workspace documentation.

### Added

- Nx logical scope expansion from the selected project into internal workspace dependency roots using the Nx project graph when available
- Process-local caching of Nx graph data per workspace to avoid repeated graph resolution during the same run
- Test coverage for Nx workspace scope expansion, helper-project exclusion, graph caching, and workspace-relative changed-path resolution
- Explicit Nx scope documentation in the main docs and a top-level README entry for discoverability

### Changed

- CLI help and flag descriptions now explain that `--project` scopes indexing, search, status, and benchmarking to dependency-aware Nx logical projects
- Indexing and CLI docs now describe how `workspaceRoot`, `PrimaryRoot`, and Vectos `Roots` interact inside Nx workspaces

### Fixed

- `vectos index --changed <paths>` now resolves workspace-relative paths correctly for Nx scopes that span multiple roots
- Common helper projects such as `e2e`, `storybook`, and `docs` no longer expand into the logical code scope by default

### Known Limitations

- This remains an experimental/internal release. Stability and compatibility are not guaranteed.
- Supported download platforms remain `darwin/arm64` and `linux/amd64` only.
- Dependency-aware logical roots rely on the Nx project graph being available from local workspace tooling; when it is unavailable, Vectos falls back to the selected project's main root.
- Default helper-project exclusions cover common `e2e`, `storybook`, and `docs` patterns, but more specialized workspace conventions may still need future configuration.

---

## v0.1.7 — 2026-04-26

Patch release focused on retrieval evaluation, better hybrid ranking, and more actionable MCP search ergonomics.

### Added

- A new `vectos benchmark <file>` workflow for repeatable retrieval evaluation against an indexed project
- A benchmark fixture format for representative queries with expected files or chunk ranges
- A seed benchmark at `benchmarks/retrieval/vectos-core.json` for validating core retrieval behavior
- A new `retrieval-evaluation` OpenSpec capability capturing benchmark behavior and reporting expectations
- Richer MCP search result metadata including rank, file info, score, preview, and relevance hints

### Changed

- Semantic retrieval now uses a hybrid reranking stage that combines embedding similarity with text and path overlap signals
- Top search results now reduce redundant overlapping candidates and limit repeated results from the same file
- MCP `search_code` now returns clearer recovery guidance when a project is unindexed or when semantic ranking should be refreshed
- MCP `index_project` now returns a structured summary with indexing mode, counts, roots, and summary text
- Main OpenSpec specs now reflect the phase 2 retrieval evaluation, hybrid ranking, and MCP ergonomics work, and the completed changes have been archived

### Fixed

- Retrieval quality improves for real code-navigation queries where pure semantic ranking previously surfaced weaker or noisier top results
- Agents can now distinguish more easily between usable search results and situations that require indexing or refresh before proceeding

### Known Limitations

- This remains an experimental/internal release. Stability and compatibility are not guaranteed.
- Supported download platforms remain `darwin/arm64` and `linux/amd64` only.
- Hybrid ranking still cannot fully compensate for all low-signal indexed content; some repositories may still need further indexing-policy cleanup for best results.
- MCP ergonomics are improved, but the tool surface is still intentionally small and may evolve in future `0.x` releases.

---

## v0.1.6 — 2026-04-25

Patch release focused on frontend retrieval quality, incremental indexing, and clearer standalone-first workflow guidance.

### Added

- Better structural chunking for JavaScript, TypeScript, TSX, and JSX files, including components, hooks, exported functions, classes, and common test blocks
- Incremental refresh support through `vectos index --changed <paths>`
- Equivalent incremental indexing support in MCP through `index_project.changed`
- A new `docs/engram-synergy.md` guide describing how Vectos can work alongside Engram without depending on it

### Changed

- Semantic enrichment now captures more useful chunk roles for frontend code
- CLI indexing now accepts both `vectos index . --changed ...` and `vectos index --changed ... .`
- Product/docs guidance now frames Vectos as standalone-first, with Engram treated as optional complementary memory
- OpenSpec main specs now reflect the merged roadmap changes, and the completed changes have been archived

### Fixed

- Retrieval quality improves for TS/React-heavy projects by using higher-signal structural chunk boundaries instead of relying mostly on generic line windows
- Incremental refresh now cleans up deleted files and files that are no longer indexable within the changed set

### Known Limitations

- This remains an experimental/internal release. Stability and compatibility are not guaranteed.
- Supported download platforms remain `darwin/arm64` and `linux/amd64` only.
- Incremental indexing currently depends on an explicit changed-file set; automatic hook or watcher-based refresh is still future work.

---

## v0.1.5 — 2026-04-25

Patch release focused on remote embedding health reporting and lower-noise indexing defaults.

### Changed

- Remote embedding provider health now performs a real probe instead of reporting a purely configuration-based status
- Remote embedding dimensions are now detected from the provider and persisted in index metadata
- Default indexing now skips `docs` and `dependency_metadata` categories to keep semantic retrieval focused on higher-signal code and config
- Semantic search now ignores `docs` and `dependency_metadata` chunks by default to reduce result noise

### Fixed

- Remote provider status no longer reports `ready` with `Embedding dimensions: 0` when the provider is actually returning valid vectors
- Reindexing now clears chunks for files that are no longer part of the default indexing set after category filtering changes
- Search quality improves in smaller projects where markdown and JSON metadata previously dominated the index

### Known Limitations

- This remains an experimental/internal release. Stability and compatibility are not guaranteed.
- Supported download platforms remain `darwin/arm64` and `linux/amd64` only.
- The current indexing defaults intentionally favor code-navigation quality over exhaustive repository coverage; `docs` and dependency metadata are not part of semantic retrieval by default.

---

## v0.1.4 — 2026-04-25

Patch release focused on indexing command visibility and progress feedback.

### Added

- `vectos index` now reports the resolved project name and root before indexing starts
- `vectos index` now reports how many supported files were found in the selected scope
- `vectos index` now prints periodic file/chunk progress updates during long indexing runs

### Changed

- Indexing output now makes Nx workspace resolution more visible by showing workspace context when applicable
- Indexing output now announces the excluded-directory cleanup phase before finishing

### Fixed

- Long-running `vectos index .` sessions no longer appear idle after the initial `Indexing:` line

### Known Limitations

- This remains an experimental/internal release. Stability and compatibility are not guaranteed.
- Supported download platforms remain `darwin/arm64` and `linux/amd64` only.
- Progress reporting is line-based and periodic; it is not yet a real-time progress bar with per-phase timing.

---

## v0.1.3 — 2026-04-25

Patch release focused on broader indexing coverage and a more structured documentation experience.

### Added

- Support for indexing more common project and devops file types, including JS/TS variants, Kotlin, GraphQL, SQL, CSS variants, lockfiles, wrapper scripts, and `.conf`
- A dedicated `docs/` documentation set covering installation, agent setup, CLI usage, indexing, development, and troubleshooting
- First-class manual MCP setup guidance for unsupported agent clients

### Changed

- The root `README.md` is now a lightweight landing page that points to the structured documentation set
- Product documentation is now organized by workflow instead of being concentrated in a single long README

### Fixed

- `.env*` files remain excluded from indexing, including `.env.example` and `.env.sample`, to avoid indexing potentially sensitive environment data

### Known Limitations

- This remains an experimental/internal release. Stability and compatibility are not guaranteed.
- Supported download platforms remain `darwin/arm64` and `linux/amd64` only.
- Manual MCP setup for unsupported clients follows a generic command pattern, but client-specific configuration examples are not yet documented.

---

## v0.1.2 — 2026-04-25

Patch release focused on broader agent integration support and installer PATH reliability.

### Added

- `vectos setup claude` to configure Claude Code with a Vectos MCP entry and managed global guidance
- `vectos setup codex` to configure Codex with a Vectos MCP entry and managed global guidance
- `vectos setup <agent> --uninstall` support for `opencode`, `claude`, and `codex`
- OpenSpec main specs now track the expanded setup matrix and setup help behavior

### Changed

- `vectos help setup` and `vectos setup --help` now document `opencode`, `claude`, `codex`, and `--uninstall`
- Installer PATH handling is now shell-aware for `zsh`, `bash`, and `fish`
- Release/install docs now explain the managed PATH block behavior more clearly

### Fixed

- Codex setup now creates `~/.codex/` before writing `config.toml`
- Installer uninstall now removes the Vectos-managed PATH block from the detected shell startup file

### Known Limitations

- This remains an experimental/internal release. Stability and compatibility are not guaranteed.
- Supported download platforms remain `darwin/arm64` and `linux/amd64` only.
- Agent uninstall removes only Vectos-managed MCP entries and managed guidance blocks. It does not delete unrelated user configuration.

---

## v0.1.1 — 2026-04-25

Patch release focused on CLI usability and installation lifecycle polish.

### Added

- Centralized CLI help with support for `vectos help`, `vectos --help`, `vectos -h`, and per-subcommand help
- `--uninstall` support in `scripts/install.sh` to remove the installed binary cleanly
- Manual cleanup guidance after uninstall for cached models, indexes, and OpenCode config

### Changed

- All user-visible CLI output in the main CLI layer is now in English
- OpenCode managed guidance text is now written in English

### Fixed

- Release installation UX now includes a documented uninstall path for installed binaries

### Known Limitations

- This remains an experimental/internal release. Stability and compatibility are not guaranteed.
- `--uninstall` removes only the installed binary. It does not automatically delete `~/.vectos/` data or agent configuration files.
- Supported download platforms remain `darwin/arm64` and `linux/amd64` only.

---

## v0.1.0 — 2026-04-25

First experimental/internal GitHub release.

### Added

- Local-first code indexing into per-project SQLite databases under `~/.vectos/projects/`
- Embedded embedding provider using `bge-small-en-v1.5` via ONNX Runtime (no external API required by default)
- Remote embedding fallback via OpenAI-compatible API (opt-in)
- Hybrid retrieval: semantic search with cosine similarity, text fallback when semantic is unavailable
- MCP server exposing `vectos_search_code` and `vectos_index_project` tools
- Nx workspace awareness: `--project` flag for scoped indexing and search
- `vectos setup opencode` to configure OpenCode MCP integration and optional global Vectos-first guidance
- `vectos version` command reporting version, commit, and build date
- Build-time version injection via `ldflags` (`buildinfo.Version`, `buildinfo.Commit`, `buildinfo.Date`)
- Experimental GitHub release assets for `darwin/arm64` and `linux/amd64`
- `checksums.txt` published alongside each release
- Source-based install script (`scripts/install.sh`) kept as fallback

### Known Limitations

- This is an experimental/internal release. Stability and compatibility are not guaranteed.
- Supported download platforms: `darwin/arm64` and `linux/amd64` only. `linux/arm64` and Windows are not validated in this release.
- On first run, the embedded provider downloads ONNX Runtime and model assets from the internet into `~/.vectos/models/`. Subsequent runs use the cached assets.
- Language support for chunking: Go (function-aware), JS/TS/JSX/TSX, Python (structured), plus a broad set of config and infra file types (line-window chunking).
- No Homebrew formula or package manager support in this release.
- CLI and MCP interface details may change in future `0.x` releases without a deprecation period.
