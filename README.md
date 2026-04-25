# Vectos

Vectos is a local-first code context engine for AI agents.

It indexes source code into project-scoped SQLite databases, generates embeddings for code chunks, and exposes search and indexing tools over MCP so clients like OpenCode can use the indexed codebase as structured context.

## Project Status

Vectos is still an evolving project under active development. Supported file types, setup flows, indexing behavior, and CLI/MCP details may change as the project matures.

## Usage Disclaimer

Use Vectos at your own responsibility. Review generated configuration changes, validate search/indexing results before relying on them, and avoid assuming the tool is production-hardened for every repository shape or workflow.

## Quick Start

### Option A — Download an experimental release (recommended for most users)

> ⚠️ Experimental/internal builds. Not a stable public release.
> Supported platforms: `darwin/arm64` and `linux/amd64` only.

1. Go to the [Releases](../../releases) page and download the archive for your platform.

2. Verify the checksum:

```bash
sha256sum -c checksums.txt
```

3. Extract and install:

```bash
tar -xzf vectos_<version>_<os>_<arch>.tar.gz
install -m 0755 vectos ~/.local/bin/vectos
```

4. Make sure `~/.local/bin` is in your `PATH`, then verify:

```bash
vectos version
```

5. Index a project:

```bash
cd /path/to/your/project
vectos index .
```

6. Search code:

```bash
vectos search "checkout payment"
```

**First-run note**: on first use, the embedded provider downloads ONNX Runtime and model assets from the internet into `~/.vectos/models/`. Subsequent runs use the cached assets.

---

### Option B — Install from source (fallback / development)

You need `git` and `go`.

If you do not have Go installed yet:

- https://go.dev/doc/install

On macOS with Homebrew:

```bash
brew install go
```

Clone and install:

```bash
git clone <YOUR_REPO_URL>
cd vectos
./scripts/install.sh
```

By default this installs `vectos` into `~/.local/bin`. To use a different directory:

```bash
DEST_DIR=/your/bin/dir ./scripts/install.sh
```

Make sure the chosen directory is in your `PATH`.

Verify:

```bash
vectos version
```

## Manual Install

If you prefer not to use the install script, you can still build and install manually.

### Build Vectos

```bash
go build -o vectos ./cmd/vectos
```

### Install the binary globally

System-wide install:

```bash
sudo install -m 0755 vectos /usr/local/bin/vectos
```

User-local install:

```bash
mkdir -p ~/.local/bin
install -m 0755 vectos ~/.local/bin/vectos
```

If you use the user-local install, make sure `~/.local/bin` is in your `PATH`.

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

The currently supported installation path is still: clone, build, install globally from source.

Minimal helper:

```bash
make build
```

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

If `~/.config/opencode/AGENTS.md` does not exist yet, the setup also installs a small global guidance block so OpenCode prefers `vectos_search_code` and `vectos_index_project` before broad file-search tools.

If you already have a global OpenCode `AGENTS.md`, the setup asks before appending the managed Vectos guidance block so your existing instructions are preserved.

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

When the optional global guidance block is installed, OpenCode is instructed to prefer those Vectos MCP tools before `grep`, `find`, `glob`, or broad file reads, and only fall back when Vectos does not return useful results.

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
- `.json`
- `.sh`
- `.md`
- `.toml`
- `.ini`
- `.xml`
- `.properties`
- `Dockerfile`
- `docker-compose*.yml`
- `*.yml`
- `*.yaml`
- `BUILD`
- `BUILD.bazel`
- `WORKSPACE`
- `MODULE.bazel`
- `*.bzl`

Secret-prone `.env` files are intentionally excluded in the current phase.

## How Retrieval Works

Vectos stores:

- the original code chunk
- file path and line ranges
- language
- file category (`source` or `infra_config`)
- file category (`source`, `infra_config`, `scripts`, `docs`, or `dependency_metadata`)
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
