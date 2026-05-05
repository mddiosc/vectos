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

To reduce search noise, the current default indexing path prioritizes higher-signal content categories. Files classified as `docs` and `dependency_metadata` are detected but skipped during indexing, so semantic retrieval stays focused on source code, scripts, and infrastructure/configuration that are more likely to answer code-navigation queries.

## Documentation Indexing

Vectos supports a separate documentation index alongside the source code index. This allows searching documentation (README, API docs, ADRs) without polluting code search results.

### Supported Documentation Files

In addition to the extensions listed above, Vectos can index these documentation formats:

- `.md` — Markdown (already listed above)
- `.mdx` — MDX (Markdown + JSX)
- `.rst` — reStructuredText
- `.adoc`, `.asciidoc` — AsciiDoc
- `.tex`, `.latex` — LaTeX
- `.txt` — Plain text

These files are classified as `docs` category and skipped during normal indexing. To index them, use the `--docs` flag:

```bash
vectos index . --docs
```

This creates a separate database at `~/.vectos/projects/<project>/<project>-docs.db`.

### Two-Index Model

Vectos maintains two separate indexes per project:

| Database | Content | Default search |
|----------|----------|----------------|
| `<project>.db` | Source code | `search_code` or `vectos search` |
| `<project>-docs.db` | Documentation | `search_docs` or `vectos search --docs` |

Both indexes share the same project scope and can coexist. The documentation index does not interfere with code search, and vice versa.

### When to Use Documentation Search

- README files, API documentation
- Architecture decision records (ADRs)
- Onboarding guides, code of conduct
- Any `.md`, `.rst`, `.adoc`, `.tex` files in the project

Agents working with documentation-heavy projects benefit from having a separate search capability that does not mix doc chunks with source code chunks.

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

For Go, Vectos prefers function-oriented chunk boundaries. For TypeScript and React-heavy files, Vectos now also prefers higher-signal structural boundaries such as exported functions, hooks, components, classes, and common test blocks when those boundaries can be derived safely. When they cannot, it falls back to the generic chunking strategy.

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
- expand internal Nx project dependencies into additional logical roots when the Nx project graph is available
- exclude non-code helper projects such as common `e2e`, `storybook`, and `docs` targets from the logical root set by default
- index/search/status against that logical project scope

Current limitation:

- dependency-aware logical roots rely on the Nx project graph being available from the local workspace tooling; when it is unavailable, Vectos falls back to the selected project's main root
- the Nx project graph is cached in memory per workspace during the current process so repeated scope resolutions do not re-run the graph command each time

### Nx Scope Model

In an Nx workspace, Vectos treats a selected project as a logical scope.

- `workspaceRoot` is the directory where `nx.json` was found
- `PrimaryRoot` is the selected Nx project's own root
- `Roots` is the full set of directories that Vectos indexes and searches together for that project scope

`Roots` is a Vectos concept, not an Nx field.

Example workspace:

```text
repo/
├─ nx.json
├─ apps/
│  └─ app-main/
│     └─ project.json           # name: app-main
└─ libs/
   ├─ feature-shell/
   │  └─ project.json           # name: feature-shell
   ├─ shared-ui/
   │  └─ project.json           # name: shared-ui
   └─ shared-auth/
      └─ project.json           # name: shared-auth
```

Example Nx dependency graph:

```text
app-main -> feature-shell
app-main -> shared-ui
feature-shell -> shared-auth
```

Running:

```bash
vectos index --project app-main .
```

Resolves a logical scope similar to:

```text
Name: app-main
WorkspaceRoot: /repo
PrimaryRoot: /repo/apps/app-main
Roots:
- /repo/apps/app-main
- /repo/libs/feature-shell
- /repo/libs/shared-ui
- /repo/libs/shared-auth
```

Vectos uses the Nx project graph to discover those related projects. It does not infer dependencies from matching names or folder structure.

By default, Vectos excludes common helper projects such as `e2e`, `storybook`, and `docs` from the expanded root set.

When the Nx graph is unavailable, Vectos falls back to the selected project's `PrimaryRoot` only.

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

For measured retrieval output and a `Vectos vs rg` comparison, see [Retrieval Benchmark](benchmarking.md).
