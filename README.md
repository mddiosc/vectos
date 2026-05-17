# Vectos

[![v1.0.0](https://img.shields.io/badge/version-v1.0.0-blue)](docs/README.md)

Vectos is a local-first code context engine for AI agents.

It indexes source code into project-scoped SQLite databases, generates code-aware embeddings using **jina-embeddings-v3** (1024-dim, 8192-token context, supports code + text + multilingual), and exposes search and indexing tools over MCP so agent clients can use the indexed codebase as structured context.

Vectos is designed to be useful as a standalone product. It can also work alongside session-memory systems such as Engram, but it does not depend on them.

## Project Status

Vectos is still under active development. Supported file types, setup flows, indexing behavior, and CLI/MCP details may change as the project matures.

## Key Features

- **Hybrid search (RRF fusion)** — combines vector similarity with keyword matching via Reciprocal Rank Fusion for higher precision
- **Code-aware embeddings** — defaults to `jina-embeddings-v3` (1024-dim, code+text+multilingual)
- **TypeScript structural tagging** — detects interfaces, type aliases, enums, hooks, and async functions for richer search
- **Configurable exclusions** — `vectos.config.json` per project, `~/.vectos/config.json` global, plus automatic `.gitignore` respect

## Quick Start

Install the latest release:

```sh
curl -fsSL https://github.com/mddiosc/vectos/releases/latest/download/install.sh | sh
```

Verify:

```bash
vectos version
```

Index and search a project:

```bash
cd /path/to/your/project
vectos index .
vectos search "checkout payment"
```

Index and search documentation separately:

```bash
vectos index . --docs
vectos search --docs "getting started"
```

Inside an Nx workspace, you can scope indexing and search to a selected Nx project:

```bash
vectos index --project app-main .
vectos search --project app-main "shared ui"
```

When the Nx project graph is available, Vectos expands that logical project scope to include internal dependency roots automatically.

Connect a validated client:

```bash
vectos setup opencode
vectos setup claude
vectos setup codex
```

### Watch Mode

`vectos serve` automatically watches the project directory for file changes and triggers incremental reindexing.

- **Enabled by default** — use `--watch=false` to disable
- **Debounce** — multiple rapid changes are batched (default 500ms)
- **Ignore patterns** — comma-separated glob patterns (default: `.git,node_modules,*.lock`)

**Limitations:**
- Requires local filesystem (not supported on network mounts)
- Hidden files and directories matched by ignore patterns are excluded

### Index Exclusions

Control what gets indexed via `vectos.config.json` in your project root:

```json
{
  "index": {
    "docs": { "exclude": ["src/content/blog/**"] },
    "code": { "exclude": ["**/__mocks__/**", "**/*.generated.*"] }
  }
}
```

Global defaults can be set in `~/.vectos/config.json` under the `index` key. Patterns from both files are merged. `.gitignore` is respected automatically.

## Documentation

- [Documentation Index](docs/README.md)
- [Installation](docs/installation.md)
- [Agent Setup](docs/agent-setup.md)
- [CLI Usage](docs/cli.md)
- [Error Guide](docs/errors.md)
- [Indexing And Retrieval](docs/indexing.md)
- [Performance Guide](docs/performance.md)
- [Nx Scope Model](docs/indexing.md#nx-scope-model)
- [Development](docs/development.md)
- [Optional Engram Synergy](docs/engram-synergy.md)
- [Troubleshooting](docs/troubleshooting.md)

## Usage Disclaimer

Use Vectos at your own responsibility. Review generated configuration changes, validate search and indexing results before relying on them, and avoid assuming the tool is production-hardened for every repository shape or workflow.
