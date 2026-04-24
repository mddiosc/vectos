# Vectos

Vectos is a local-first code context engine for AI agents.

It indexes source code into project-scoped SQLite databases, generates embeddings for code chunks, and exposes search and indexing tools over MCP so clients like OpenCode can use the indexed codebase as structured context.

## What It Does

- Indexes source files into per-project SQLite databases under `~/.vectos/projects/`
- Generates embeddings for code chunks using a configurable embedding provider
- Supports hybrid retrieval:
  - semantic search with cosine similarity over stored embeddings
  - text fallback when semantic search is unavailable or insufficient
- Exposes MCP tools for agent workflows:
  - `search_code`
  - `index_project`
- Automates OpenCode MCP setup with `vectos setup opencode`

## Current Capabilities

- Project-aware indexing and retrieval
- MCP integration with OpenCode via the official Go MCP SDK
- Function-aware chunking for Go
- Basic structured chunking for:
  - JavaScript
  - JSX
  - TypeScript
  - TSX
  - Python
- Reindexing without duplicate rows for the same file
- Index status inspection

## Architecture

Vectos currently uses this stack:

- Storage: SQLite
- Embeddings: embedded-first provider strategy
- Embedded runtime: in-process ONNX + local tokenizer
- Remote fallback: OpenAI-compatible embeddings API
- MCP server: `github.com/modelcontextprotocol/go-sdk/mcp`
- Index isolation: one database per active project

Default embedded configuration:

- Provider: `embedded`
- Model: `bge-small-en-v1.5`
- Cache directory: `~/.vectos/models/bge-small-en-v1.5/`

Default fallback configuration:

- Provider order: `embedded`, then `remote`
- Remote provider: disabled by default
- Remote model: `text-embedding-nomic-embed-text-v1.5`
- Remote base URL: user-provided when enabled

## Build

```bash
go build -o vectos ./cmd/vectos
```

## Install

Recommended global install during development:

```bash
go build -o vectos ./cmd/vectos
install -m 0755 vectos /usr/local/bin/vectos
```

After that, use `vectos` directly instead of `./vectos`.

Optional release-oriented helpers:

```bash
make build
make release-layout
make release-snapshot
```

The currently supported installation path is still: clone, build, install globally from source.

Homebrew-compatible metadata and release scaffolding may exist as future-oriented packaging groundwork, but they are not the primary supported installation path right now.

Current packaging constraint:

- cross-platform release builds are not yet fully self-contained from a single host because the embedded ONNX runtime dependency is platform-specific
- if you later choose to publish binaries, use native per-platform CI/release jobs for final packaged artifacts

## CLI Commands

### Index a file

```bash
vectos index sample_code.go
```

When indexing inside an Nx workspace, you can select the logical project explicitly:

```bash
vectos index --project web .
```

### Search current project index

```bash
vectos search "suma"
```

Inside an Nx workspace, you can scope search explicitly to one Nx project:

```bash
vectos search --project web "checkout"
```

`search` now uses semantic retrieval first when the active index metadata matches the current embedding provider, then falls back to text search if semantic retrieval is unavailable or incompatible.

### Show index status

```bash
vectos status
```

Inside an Nx workspace, you can inspect the selected logical project scope:

```bash
vectos status --project web
```

Example output includes:

- active project database path
- database size
- indexed files
- indexed chunks
- chunks with embeddings
- chunks without embeddings

It also reports provider health for both embedded and remote providers, plus whether a reindex is required because the current provider/model/dimensions no longer match the stored index metadata.

When a workspace-scoped project is selected, status also reports:

- logical project name
- workspace type
- resolved project roots

### Start MCP server manually

```bash
vectos mcp
```

### Configure OpenCode automatically

```bash
vectos setup opencode
```

This creates or updates `~/.config/opencode/opencode.json` with a working local MCP entry for Vectos.

## Supported Setup Targets

Current validated setup target:

- `opencode`

Current explicit non-validated targets for this phase:

- `claude`
- `codex`
- `gemini`

Running `vectos setup <agent>` for a non-validated target currently fails with an explicit error instead of pretending support exists.

Setup commands now assume a global `vectos` binary UX. The generated agent config points at the resolved `vectos` executable rather than documenting `./vectos` as the intended installation model.

## OpenCode Workflow

Typical agent workflow:

1. Index a project:

```text
Use `vectos_index_project` to reindex /path/to/project
```

2. Search semantically:

```text
Use `vectos_search_code` to find the function that divides two integers and avoids division by zero.
```

## Indexed Languages

Vectos currently detects language by file extension.

Supported extensions:

- `.go`
- `.js`
- `.jsx`
- `.ts`
- `.tsx`
- `.py`
- `.java`
- `Dockerfile`
- `docker-compose*.yml`
- `*.yml`
- `*.yaml`
- `BUILD`
- `BUILD.bazel`
- `WORKSPACE`
- `MODULE.bazel`
- `*.bzl`

## How Retrieval Works

Vectos stores:

- the original code chunk
- file path and line ranges
- language
- file category (`source` or `infra_config`)
- embedding vector

When a search query arrives:

1. Vectos tries to embed the query
2. It ranks indexed chunks using cosine similarity
3. If semantic retrieval fails or returns nothing useful, it falls back to text search

Semantic retrieval is only used when the current provider metadata matches the metadata stored with the index. If the active provider, model, or vector dimensions changed, Vectos treats the index as incompatible and avoids mixing embeddings from different providers.

Search results now preserve both:

- logical project scope when workspace selection is used
- file classification metadata such as `source/java` or `infra_config/dockerfile`

Before embedding, chunks are enriched with lightweight semantic metadata such as:

- file name
- language
- extracted signature when available
- inferred purpose heuristics

## Project Storage

Project indexes are stored under:

```text
~/.vectos/projects/<project-name>/<project-name>.db
```

This keeps each project's code context isolated.

For Nx-scoped projects, the database name is based on the logical Nx project name rather than the current directory name.

## Workspace Selection

Vectos now supports a first Nx-aware workspace phase.

Current behavior:

- If the current path is not inside an Nx workspace, Vectos keeps the existing single-project behavior.
- If the current path is inside an Nx workspace, you can select an Nx project explicitly with `--project` in the CLI.
- MCP tools also accept an optional `project` field when working inside an Nx workspace.

Current Nx-supported flow:

- detect `nx.json`
- discover Nx projects from `project.json`
- resolve the selected Nx project's root
- index/search/status against that logical project scope

Current limitation:

- the first implementation phase resolves a selected Nx project to its main project root; generic manual multi-root path groups are intentionally out of scope for now

## Embedding Configuration

Embedding configuration is loaded from `~/.vectos/config.json`.

Example:

```json
{
  "embeddings": {
    "default_provider": "embedded",
    "fallback_order": ["embedded", "remote"],
    "embedded": {
      "enabled": true,
      "model_name": "bge-small-en-v1.5",
      "model_dir": "/Users/you/.vectos/models/bge-small-en-v1.5",
      "auto_download": true,
      "asset_base_url": "https://huggingface.co/BAAI/bge-small-en-v1.5/resolve/main",
      "timeout_seconds": 60
    },
    "remote": {
      "enabled": true,
      "base_url": "http://localhost:4000/v1",
      "model": "text-embedding-nomic-embed-text-v1.5",
      "timeout_seconds": 30
    }
  }
}
```

Notes:

- `embedded.enabled: false` cleanly disables the local provider.
- `remote.enabled: false` disables remote fallback.
- `remote.base_url` is intentionally not hardcoded by default; set it to your own OpenAI-compatible endpoint only if you want remote fallback.
- `fallback_order` controls provider resolution explicitly.
- `default_provider` controls which provider Vectos tries first.

## Embedded Model Cache

The default embedded provider manages its own local cache under `~/.vectos/models/`.

For `bge-small-en-v1.5`, Vectos automatically downloads and caches:

- `config.json`
- `tokenizer.json`
- `model.onnx`
- a platform-specific ONNX Runtime shared library, normalized locally as:
  - `onnxruntime.dylib` on macOS
  - `onnxruntime.so` on Linux

Vectos downloads model assets from Hugging Face and extracts the ONNX Runtime shared library from the official Microsoft release for the current platform.

## Reindex Behavior

Vectos stores index metadata with each project database:

- provider name
- model name
- embedding dimensions

If any of those values differ from the currently active embedding provider, Vectos reports that a reindex is required.

Typical cases that require reindexing:

- switching from `remote` to `embedded`
- switching from `embedded` to `remote`
- changing the embedded model
- changing the remote model to one with different dimensions
- rebuilding the index with a different embedding space

In practice:

- run `vectos status` to see whether reindexing is required
- run `vectos index /path/to/project` again to rebuild embeddings with the active provider

## Current Limitations

- Multi-language chunking is heuristic-based, not parser-based
- Search result formatting is functional but still noisy for large projects
- There are not yet automated tests for all indexing and retrieval paths
- CLI project scoping still defaults to the current working directory, so `status` and `search` should be run from the indexed project root unless an explicit project path option is added later

## Next Likely Improvements

- richer result formatting and ranking controls
- parser-based chunking for more languages
- automated tests for indexing and retrieval
- additional agent setup targets beyond OpenCode

## Development Notes

Relevant project files:

- `cmd/vectos/main.go` — CLI, MCP server, setup automation
- `internal/indexer/chunker.go` — chunking and semantic enrichment
- `internal/storage/sqlite.go` — storage, stats, semantic ranking
- `internal/storage/project_manager.go` — project-aware database routing
- `AGENTS.md` — local guidance for agents using this repository
