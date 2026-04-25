# Indexing And Retrieval

## What Vectos Does

- Indexes source files into per-project SQLite databases under `~/.vectos/projects/`
- Generates embeddings for code chunks using a configurable embedding provider
- Supports hybrid retrieval:
  - semantic search with cosine similarity over stored embeddings
  - text fallback when semantic search is unavailable or insufficient
- Exposes MCP tools for agent workflows:
  - `search_code`
  - `index_project`

## Indexed Files

Vectos currently detects files by extension and common project file names.

Supported extensions and file names:

- `.go`
- `.js`
- `.mjs`
- `.cjs`
- `.jsx`
- `.ts`
- `.mts`
- `.cts`
- `.tsx`
- `.py`
- `.java`
- `.kt`
- `.kts`
- `.json`
- `.sh`
- `.md`
- `.mdx`
- `.toml`
- `.ini`
- `.conf`
- `.xml`
- `.properties`
- `.gradle`
- `.sql`
- `.proto`
- `.graphql`
- `.gql`
- `.css`
- `.scss`
- `.sass`
- `.less`
- `Dockerfile`
- `docker-compose*.yml`
- `*.yml`
- `*.yaml`
- `BUILD`
- `BUILD.bazel`
- `WORKSPACE`
- `MODULE.bazel`
- `*.bzl`
- `*.lock` such as `Cargo.lock`, `yarn.lock`, and `poetry.lock`
- `.editorconfig`
- `.npmrc`
- `.yarnrc`
- `.nvmrc`
- `.prettierrc`
- `.prettierignore`
- `.eslintignore`
- `.tool-versions`
- `gradlew`
- `mvnw`

Secret-prone `.env` files are intentionally excluded in the current phase, including `.env.example` and `.env.sample`.

## How Retrieval Works

Vectos stores:

- the original code chunk
- file path and line ranges
- language
- file category (`source`, `infra_config`, `scripts`, `docs`, or `dependency_metadata`)
- embedding vector

When a search query arrives:

1. Vectos tries to embed the query
2. It ranks indexed chunks using cosine similarity
3. If semantic retrieval fails or returns nothing useful, it falls back to text search

Semantic retrieval is only used when the current provider metadata matches the metadata stored with the index. If the active provider, model, or vector dimensions changed, Vectos treats the index as incompatible and avoids mixing embeddings from different providers.

Search results preserve both logical project scope and file classification metadata.

## Workspace Selection

Vectos supports an Nx-aware workspace phase.

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

See also: [Development](development.md)

If results look stale or low quality, also see [Troubleshooting](troubleshooting.md).
