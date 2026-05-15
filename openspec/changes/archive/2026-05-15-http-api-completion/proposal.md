## Why

The `vectos serve` HTTP server currently exposes only `/health` and `/reindex` — clients (including agent orchestrators) must use MCP tools for search. An HTTP search API enables lightweight integrations, health-check-only clients, and observability tooling without requiring MCP protocol support.

## What Changes

- Add `POST /search` endpoint accepting `{"query": "...", "project": "...", "limit": 10}` returning ranked results
- Add `POST /search/code` and `POST /search/docs` variants for targeted index search
- Add `GET /metrics` returning index size, last index time, uptime, watcher status, embedding model info
- Add `GET /status/:project` returning per-project indexed status
- Add request validation (query length, project name sanitization, limit range 1–100)
- Normalize JSON error responses to `{"error": "message", "code": "ERROR_CODE"}` with appropriate HTTP status codes
- **BREAKING**: The current error format `{"status": "error", "message": "..."}` changes to `{"error": "...", "code": "..."}` for newly added endpoints. Existing `/health` and `/reindex` endpoints retain backward compat via a deprecation grace period.

## Capabilities

### New Capabilities
- `http-search`: HTTP endpoints for semantic and text search over code and docs indexes
- `http-metrics`: Observability endpoint exposing index and runtime metrics
- `http-project-status`: Per-project index status endpoint

### Modified Capabilities
- `serve-http-server`: Server gains new route handlers for `/search*`, `/metrics`, and `/status/:project`. Error response format evolved.

## Impact

- **`internal/server/server.go`**: New handler functions and route registration
- **`internal/storage/sqlite.go`**: `SearchSemantic` and `SearchText` already exist; reused as-is
- **`cmd/vectos/cli_dispatch.go`**: May need updated search function wiring if server-side search is added to CLI
- No new external dependencies required; validation uses manual checks
