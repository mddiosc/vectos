# CLI Usage

## Common Commands

### Show version

```bash
vectos version
```

### Index a file or directory

```bash
vectos index .
vectos index sample_code.go
```

Control embedding dimensions with `--dimensions` (default: 512):

```bash
vectos index . --dimensions 256
vectos index . --dimensions 1024
```

Valid values: `32`, `64`, `128`, `256`, `512`, `768`, `1024`. Lower values use less storage but reduce retrieval precision slightly. See [Indexing And Retrieval](indexing.md#matryoshka-dimensions) for quality benchmarks.

Refresh only changed files within the selected project scope:

```bash
vectos index . --changed src/App.tsx,src/hooks/useAuth.ts
```

Inside an Nx workspace:

```bash
vectos index --project web .
```

If you run the command from inside a project directory, Vectos can often infer the workspace automatically and still use the selected project scope:

```bash
cd apps/app-main
vectos index .
```

For Nx workspaces, the selected project scope can expand to include internal dependent libs when the Nx project graph is available.

### Search the current project index

```bash
vectos search "checkout payment"
```

By default, `search` prints compact snippets with file, line range, score, and a short
reason. Use `--full` only when you need the complete chunk content.

Inside an Nx workspace:

```bash
vectos search --project web "checkout"
```

This searches the selected Nx project together with any included dependency roots in the logical scope.

`search` uses semantic retrieval first when the active index metadata matches the current embedding provider, then falls back to text search if semantic retrieval is unavailable or incompatible.

### Search Documentation

To search the documentation index (separate from source code):

```bash
vectos search --docs "getting started"
vectos search --project web --docs "authentication"
```

This uses the `--docs` flag to search `<project>-docs.db` instead of the source code database.

### Show index status

```bash
vectos status
```

To see documentation index status instead:

```bash
vectos status --docs
```

Inside an Nx workspace:

```bash
vectos status --project web
```

For Nx scopes, `status` also shows the resolved project roots so you can verify which app and libs were included.

Example output includes:

- active project database path
- database size
- indexed files
- indexed chunks
- chunks with embeddings
- chunks without embeddings
- provider health
- whether reindexing is required

### Run diagnostics

```bash
vectos doctor
```

The `doctor` command checks installation, runtime readiness, embedding provider
health, and index consistency in one step. It reports each check with a
pass/warn/fail marker and an actionable hint if something is wrong.

`doctor` is read-only and safe to run at any time. It exits with code 0 when
all critical checks pass.

### Start the HTTP server with watch mode

```bash
vectos serve
```

Starts an HTTP server on `127.0.0.1:7438` (port configurable via `--port` or `VECTOS_PORT` env). Watch mode is enabled by default — Vectos watches the project directory for file changes and triggers incremental reindexing automatically.

Watch mode flags:

| Flag | Default | Description |
|------|---------|-------------|
| `--watch` | `true` | Enable filesystem watching |
| `--watch-debounce` | `500ms` | Debounce interval for batching rapid changes |
| `--watch-ignore` | `.git,node_modules,*.lock` | Comma-separated glob patterns to ignore |

Limitations:

- Requires local filesystem (not supported on network mounts)
- Hidden files and directories matched by ignore patterns are excluded

### HTTP API Endpoints

When `vectos serve` is running, the following endpoints are available:

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/health` | Health check |
| `POST` | `/search` | Search source code (same as `/search/code`) |
| `POST` | `/search/code` | Search source code index |
| `POST` | `/search/docs` | Search documentation index |
| `POST` | `/reindex` | Trigger project reindex (rate-limited: 1 req/s, burst 5) |
| `GET` | `/metrics` | Index stats, provider info, uptime, watcher status |
| `GET` | `/status/:project` | Per-project index status |

#### Search Request

```json
{
  "query": "checkout payment",
  "project": "my-project",
  "limit": 10
}
```

- `query` (required) — search query string
- `project` (optional) — Nx project name to scope the search
- `limit` (optional, 1–100, default 10) — max results to return

#### Search Response

```json
{
  "results": [
    {
      "file_path": "src/checkout.go",
      "file_name": "checkout.go",
      "language": "go",
      "relevance": 0.87,
      "line_ranges": [[12, 45]],
      "signatures": ["func processPayment(ctx context.Context, order *Order) error"]
    }
  ],
  "mode": "semantic_hybrid",
  "total": 3
}
```

- `mode` is `"semantic_hybrid"` when vector search is used, `"text"` when falling back to text search

#### Error Response

```json
{
  "error": "query is required",
  "code": "INVALID_QUERY"
}
```

Error codes: `INVALID_QUERY`, `INVALID_PROJECT`, `INVALID_LIMIT`, `PROJECT_NOT_FOUND`, `INTERNAL_ERROR`, `METHOD_NOT_ALLOWED`

### Start the MCP server manually

```bash
vectos mcp
```

### Help

```bash
vectos help
vectos help setup
vectos setup --help
```

See also: [Indexing And Retrieval](indexing.md)

For a detailed explanation of Nx logical scopes, roots, exclusions, and fallback behavior, see the `Nx Scope Model` section in [Indexing And Retrieval](indexing.md).

If commands do not behave as expected, also see [Troubleshooting](troubleshooting.md).
