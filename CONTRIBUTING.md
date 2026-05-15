# Contributing to Vectos

Vectos is still in the `0.x` experimental phase. Interfaces, packaging, and behavior may change without notice.

## Prerequisites

- Go 1.26+
- CGO (required for SQLite)
- `staticcheck` for linting: `go install honnef.co/go/tools/cmd/staticcheck@latest`

## Development

Build:

```bash
make build
```

Build with version info:

```bash
make build-dev
```

## Checks

Before submitting a change, run:

```bash
make check
```

This runs `go vet`, `staticcheck`, and `go test ./...`.

Individual targets:

```bash
make lint
make test
```

## Pull Requests

- Open PRs against `main`.
- CI runs vet, staticcheck, and tests automatically.
- Keep changes focused. One concern per PR is easier to review.
- If a change adds or modifies user-facing behavior, update the relevant docs under `docs/` and `CHANGELOG.md`.

## Changelog

Vectos uses a changelog under `CHANGELOG.md`. When adding a user-facing change, add an entry under the `### Added`, `### Changed`, or `### Fixed` section of the current unreleased version. If no unreleased section exists, ask the maintainer to create one.

## Project Structure

```
cmd/vectos/       CLI commands, MCP server, search output
internal/
  buildinfo/      Version metadata (injected via ldflags)
  config/         Configuration loading and defaults
  content/        File categorization and language detection
  embeddings/     Embedding providers (embedded ONNX, remote API)
  indexer/        Chunking and semantic enrichment
  search/         Search dispatch and ranking
  server/         HTTP server (serve mode)
  setup/          Agent setup automation
  storage/        SQLite storage layer
  vectorindex/    HNSW vector index, SQ8 compression, persistence
  watcher/        Filesystem watcher for watch mode
  workspace/      Nx workspace detection and scope resolution
docs/             User documentation
openspec/         Design specs and change history
benchmarks/       Retrieval benchmark fixtures
scripts/          Install script
```

## Questions

If something is unclear, open an issue before investing significant work.
