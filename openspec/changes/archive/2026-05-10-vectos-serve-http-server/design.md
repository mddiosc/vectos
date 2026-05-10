## Context

Vectos currently operates in two modes: CLI commands (`vectos index`, `vectos search`) and MCP server (`vectos mcp`). Both are process-per-invocation — each starts fresh, loads the ONNX embedding model (~2-3s), opens SQLite, and shuts down. This is fine for one-off CLI usage but problematic for the OpenCode plugin, which needs to trigger incremental reindexes on file changes.

The current plugin spawns `vectos index --changed <paths>` via `Bun.spawnSync()` on every file change (debounced 3s). This:
- Leaks output into the OpenCode TUI
- Blocks the Bun event loop during execution
- Cold-starts the embedding model each time
- Creates process overhead (fork + exec + model load + SQLite open + close)

Engram's pattern (HTTP server on localhost) solves all of these: `fetch()` is async, silent, and the server keeps state in memory.

## Goals / Non-Goals

**Goals:**
- Provide a persistent HTTP server (`vectos serve`) that keeps the embedding model and SQLite connections in memory
- Expose a `/reindex` endpoint for incremental reindexing via HTTP POST
- Expose a `/health` endpoint for liveness checks
- Auto-start the server from the OpenCode plugin if not running
- Make incremental reindex near-instantaneous (no cold start)
- Keep the plugin completely silent (no TUI pollution)

**Non-Goals:**
- Replacing the MCP server — MCP remains the primary agent interface
- Exposing search via HTTP — the MCP server already handles this
- Multi-user or remote access — this is strictly localhost
- Authentication — localhost-only, no auth needed
- WebSocket support — simple request/response is sufficient
- Replacing CLI commands — `vectos index` and `vectos search` remain as-is

## Decisions

### 1. HTTP server using stdlib `net/http`

**Choice**: Go stdlib `net/http` with a simple mux.

**Alternatives considered**:
- **gin/echo/fiber**: Overkill for 2-3 endpoints. Adds dependencies for no benefit.
- **gRPC**: Too complex for localhost JSON endpoints.

**Rationale**: We need exactly 2 endpoints (`/health`, `/reindex`). Stdlib is sufficient, zero new dependencies, and Go's HTTP server is production-grade.

### 2. Port selection: 7438

**Choice**: Default port 7438 (Engram uses 7437).

**Alternatives considered**:
- **Random port**: Harder for plugin to discover.
- **Unix socket**: More complex, no real benefit for localhost.

**Rationale**: Fixed port is simple and predictable. Configurable via `VECTOS_PORT` env var. 7438 follows Engram's 7437 convention.

### 3. Embedding model stays loaded in memory

**Choice**: Initialize the ONNX embedder once at server startup, keep it in the `ServeConfig` struct.

**Rationale**: The ONNX model load is the dominant cost (~2-3s). Keeping it in memory makes each reindex near-instant. Memory footprint is ~50MB for bge-small-en-v1.5, acceptable for a local dev tool.

### 4. SQLite: open per-project, kept in memory

**Choice**: Open SQLite connections per project on first request, cache them in a map. Close on server shutdown.

**Rationale**: Avoids repeated open/close overhead. The server handles one project at a time (the current workspace), so the map will typically have 1-2 entries.

### 5. Plugin uses `fetch()` instead of `Bun.spawnSync()`

**Choice**: Replace all `Bun.spawnSync()` calls in the plugin with `fetch()` to `http://127.0.0.1:7438/reindex`.

**Rationale**: `fetch()` is async, non-blocking, and produces zero TUI output. The plugin auto-starts the server if not running (same pattern as Engram).

### 6. Server auto-start from plugin

**Choice**: On plugin initialization, check `/health`. If not running, spawn `vectos serve` in background (`Bun.spawn` with detached stdio). Wait 500ms for startup.

**Rationale**: Matches Engram's pattern exactly. User doesn't need to manually start the server.

## Risks / Trade-offs

- **[Port conflict]** → Mitigation: `VECTOS_PORT` env var, clear error message if port is taken. Health check before starting.
- **[Memory usage]** → Mitigation: ONNX model is ~50MB. Acceptable for dev tool. Server can be stopped when not needed.
- **[Stale server]** → Mitigation: Health check on plugin init. If server is unresponsive, kill and restart.
- **[Server crash]** → Mitigation: Plugin falls back to CLI spawn on fetch failure (graceful degradation).
- **[Concurrent reindex requests]** → Mitigation: Server serializes reindex requests (mutex or channel-based). Rapid file changes get batched by the plugin's debounce timer before hitting the server.

## Open Questions

- Should the server also expose a `/search` endpoint for direct HTTP search? Currently out of scope (MCP handles this), but could be useful for future integrations.
- Should we add a `/status` endpoint that returns index metadata (last reindex time, chunk count)? Low priority but useful for debugging.