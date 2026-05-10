## Why

Vectos currently has no automatic reindex mechanism — the agent must manually call `index_project` with `changed` paths after editing files. The OpenCode plugin we created spawns `vectos index --changed` via `Bun.spawnSync()` on every file change, which:

1. **Pollutes the TUI** — process output leaks into the OpenCode interface
2. **Blocks the event loop** — `spawnSync` is synchronous
3. **Cold-starts every time** — each invocation loads the ONNX embedding model, opens SQLite, and tears down, making incremental reindex ~10x slower than it could be
4. **Is fragile** — CLI process spawning from a plugin is inherently noisy and error-prone

Engram solves this pattern with a persistent HTTP server (`engram serve` on port 7437). The plugin makes lightweight `fetch()` calls — async, silent, no process overhead. Vectos should adopt the same architecture.

Additionally, keeping the embedding model loaded in memory eliminates the ~2-3s cold start per reindex, making incremental updates near-instantaneous.

## What Changes

- **New command: `vectos serve`** — starts a persistent HTTP server that keeps the embedding model and SQLite connections in memory
- **New HTTP endpoints**: `/health`, `/reindex` (POST with `{path, changed, project, docs}`), `/search` (POST with query params)
- **Modified OpenCode plugin** — replaces `Bun.spawnSync()` with `fetch()` calls to the local server, auto-starts server if not running
- **Modified `vectos setup opencode`** — plugin now auto-starts the server instead of spawning CLI processes
- **Server lifecycle** — auto-started by the plugin, health-checked on startup, graceful shutdown on SIGTERM

## Capabilities

### New Capabilities
- `serve-http-server`: Persistent HTTP server mode that keeps embedding model and SQLite in memory, exposing reindex and search endpoints for low-latency incremental updates

### Modified Capabilities
- `mcp-interface`: The MCP server remains the primary interface for agent tools, but the HTTP server provides a complementary API for the OpenCode plugin's reindex triggers
- `runtime-command-services`: New `serve` command added to the CLI dispatch alongside existing `index`, `search`, `mcp`, etc.

## Impact

- **New files**: `cmd/vectos/serve.go` (serve command), `internal/server/` (HTTP server, handlers)
- **Modified files**: `cmd/vectos/cli_dispatch.go` (add serve command), `cmd/vectos/cli_flags.go` (serve flags), `internal/setup/plugins/vectos.ts` (fetch instead of spawnSync), `internal/setup/opencode.go` (plugin source update)
- **Dependencies**: Only stdlib `net/http` — no new external dependencies
- **Backward compatible**: CLI commands (`vectos index`, `vectos search`, `vectos mcp`) continue to work exactly as before. The HTTP server is an optional addition.