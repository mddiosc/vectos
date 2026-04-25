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
