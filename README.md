# Vectos

Vectos is a local-first code context engine for AI agents.

It indexes source code into project-scoped SQLite databases, generates embeddings for code chunks, and exposes search and indexing tools over MCP so agent clients can use the indexed codebase as structured context.

Vectos is designed to be useful as a standalone product. It can also work alongside session-memory systems such as Engram, but it does not depend on them.

## Project Status

Vectos is still under active development. Supported file types, setup flows, indexing behavior, and CLI/MCP details may change as the project matures.

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

Connect a validated client:

```bash
vectos setup opencode
vectos setup claude
vectos setup codex
```

## Documentation

- [Documentation Index](docs/README.md)
- [Installation](docs/installation.md)
- [Agent Setup](docs/agent-setup.md)
- [CLI Usage](docs/cli.md)
- [Indexing And Retrieval](docs/indexing.md)
- [Development](docs/development.md)
- [Optional Engram Synergy](docs/engram-synergy.md)
- [Troubleshooting](docs/troubleshooting.md)

## Usage Disclaimer

Use Vectos at your own responsibility. Review generated configuration changes, validate search and indexing results before relying on them, and avoid assuming the tool is production-hardened for every repository shape or workflow.
