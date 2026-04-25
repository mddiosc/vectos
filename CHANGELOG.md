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
- Indexed file coverage now includes more common project manifests and source/config formats used by Nx, Next.js, Vite, Java, Kotlin, GraphQL, SQL, `.conf`, and related toolchains, while keeping `.env*` files excluded

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
