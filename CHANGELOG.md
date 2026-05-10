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
