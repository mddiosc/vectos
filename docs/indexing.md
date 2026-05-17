# Indexing And Retrieval

## What Vectos Does

- Indexes source files into per-project SQLite databases under `~/.vectos/projects/`
- Generates embeddings for code chunks using a configurable embedding provider (default: **jina-embeddings-v3**, a code+text-aware model with 8192 token context. Default output: **512 dimensions** via Matryoshka truncation, configurable from 32 to 1024)
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

1. Vectos embeds the query using the active provider
2. It searches the HNSW vector index for approximate nearest neighbors (O(log n) lookup)
3. If the vector index is missing or stale, it falls back to linear scan over stored embeddings
4. Results are fused using Reciprocal Rank Fusion (RRF), combining semantic similarity with BM25 text matching
5. If semantic retrieval fails entirely, it falls back to pure text search

Semantic retrieval is only used when the current provider metadata matches the metadata stored with the index. If the active provider, model, or vector dimensions changed, Vectos treats the index as incompatible and avoids mixing embeddings from different providers.

## Vector Index Configuration

Vectos uses an HNSW (Hierarchical Navigable Small World) vector index for fast approximate nearest neighbor search. The index is persisted as a binary `.vectorindex` file alongside the SQLite database, with content-hash staleness detection and automatic rebuild on `vectos index`.

Configuration is set in `~/.vectos/config.json` under `vector_index`:

```json
{
  "vector_index": {
    "index_type": "hnsw",
    "hnsw_m": 16,
    "hnsw_ef_construction": 200,
    "hnsw_ef_search": 200,
    "compression": "none"
  }
}
```

| Key | Default | Description |
|-----|---------|-------------|
| `index_type` | `"hnsw"` | Vector index type (currently only `hnsw`) |
| `hnsw_m` | `16` | Max connections per node in the HNSW graph |
| `hnsw_ef_construction` | `200` | Search depth during index construction (higher = better quality, slower build) |
| `hnsw_ef_search` | `200` | Search depth during query (higher = better recall, slower query) |
| `compression` | `"none"` | Set to `"sq8"` for 8-bit scalar quantization (4x storage reduction, lossy) |

### SQ8 Scalar Quantization

Setting `compression: "sq8"` compresses each embedding dimension from 32-bit float to 8-bit integer, reducing storage by ~4x. This is lossy — recall may degrade slightly for some queries but typically remains above 80%.

### Embedding Batch Size

The embedding batch size controls how many chunks are embedded in a single ONNX inference batch. Configuration is in `~/.vectos/config.json`:

```json
{
  "embeddings": {
    "embedded": {
      "batch_size": 32
    }
  }
}
```

Default: `32`. Larger values may improve indexing throughput at the cost of higher memory usage.

### Matryoshka Dimensions

jina-embeddings-v3 supports **Matryoshka Representation Learning (MRL)** — embeddings can be truncated to a smaller dimension while preserving most of the search quality. This lets you trade retrieval accuracy for speed and storage.

Set the output dimension with the `--dimensions` flag or via config:

```bash
vectos index . --dimensions 256
```

Config in `~/.vectos/config.json`:

```json
{
  "embeddings": {
    "embedded": {
      "dimensions": 256
    }
  }
}
```

Valid values: `32`, `64`, `128`, `256`, `512`, `768`, `1024`.

| Dimension | Retrieval Quality (nDCG@10) | vs 1024d | Storage per vector |
|-----------|----------------------------|----------|-------------------|
| 1024 | 63.35% | baseline | 4,096 bytes |
| 768 | 63.30% | −0.05 | 3,072 bytes |
| 512 | 63.16% | −0.19 | 2,048 bytes ← **default** |
| 256 | 62.72% | −0.63 | 1,024 bytes |
| 128 | 61.64% | −1.71 | 512 bytes |
| 64 | 58.54% | −4.81 | 256 bytes |
| 32 | 52.54% | −10.81 | 128 bytes |

**Recommendations:**
- **512** (default) — good balance for most projects
- **256** — large monorepos where speed and storage matter
- **1024** — when maximum retrieval precision is critical

Changing dimensions triggers automatic re-embedding of all files. The previous index is invalidated and rebuilt with the new dimension.

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
- exclude only projects with Nx type `"e2e"` from the logical root set by default (name-based heuristics removed; all internal libs are included)
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

By default, Vectos excludes only projects whose Nx type is `"e2e"` from the expanded root set. All internal dependency libs — regardless of name — are included. Set `VECTOS_NX_INCLUDE_E2E=1` to override and include e2e projects.

When the Nx graph is unavailable, Vectos falls back to the selected project's `PrimaryRoot` only. When this happens, Vectos surfaces a warning so you know the scope is incomplete.

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
- changing the Matryoshka `dimensions` setting (e.g., 512 → 256)
- rebuilding the index with a different embedding space

See also: [Development](development.md)

If results look stale or low quality, also see [Troubleshooting](troubleshooting.md).

For measured retrieval output and a `Vectos vs rg` comparison, see [Retrieval Benchmark](benchmarking.md).

## Index Exclusions

Vectos excludes sensitive files (`.env`, private keys, certificates) from all indexes. You can add additional exclusions via configuration.

### Hardcoded Exclusions (always applied)

| Category | Patterns |
|----------|----------|
| Sensitive files | `.env`, `id_rsa`, `credentials.json`, `*.pem`, `*.key` |
| Directories | `.git`, `node_modules`, `dist`, `build`, `coverage`, `.next` |
| Lockfiles | `pnpm-lock.yaml`, `package-lock.json`, `yarn.lock`, `go.sum` |
| Config files | `eslint.config.*`, `tailwind.config.*`, `tsconfig.json` |

### Project Config (`vectos.config.json`)

Place this file in your project root to add exclusion patterns:

```json
{
  "index": {
    "docs": {
      "exclude": ["src/content/blog/**", ".github/prompts/**"]
    },
    "code": {
      "exclude": ["**/__mocks__/**", "**/*.generated.*"]
    }
  }
}
```

Patterns use gitignore/glob syntax (`**` for recursive, `*` for single-level).

### Global Config (`~/.vectos/config.json`)

Add an `index` section to set defaults for all projects:

```json
{
  "embeddings": { ... },
  "index": {
    "docs": { "exclude": [".agents/**"] },
    "code": { "exclude": [] }
  }
}
```

### .gitignore

Vectos automatically respects your project's `.gitignore`. Files ignored by git are excluded from indexing. No configuration needed.

Limitations:

- Negation patterns (`!important.log`) are not supported — they are skipped during parsing
- Only the root `.gitignore` is read; nested `.gitignore` files in subdirectories are not processed

### Exclusions are cumulative

Hardcoded exclusions always apply. Global config adds more. Project config adds even more. Patterns from all three layers are active simultaneously — removing a global exclusion requires editing the global config.
