## Context

The `vectos serve` command (internal/server/server.go) currently runs an HTTP server on `127.0.0.1:<port>` with two endpoints: `GET /health` and `POST /reindex`. Search is only available through MCP tools (`search_code`, `search_docs`) exposed by the MCP server in `cmd/vectos/mcp_handlers.go`. The storage layer (`internal/storage/sqlite.go`) already provides `SearchSemantic(queryVector, limit, includeDocs)` and `SearchText(query)`, plus `Stats()` and `GetIndexMetadata()`.

This design adds HTTP search and observability endpoints, reusing the existing storage methods with no new external dependencies.

## Goals / Non-Goals

**Goals:**
- Add `POST /search` endpoint for semantic + text search over the code index
- Add `POST /search/code` and `POST /search/docs` variants for targeted index search
- Add `GET /metrics` for index stats and runtime info
- Add `GET /status/:project` for per-project index status
- Validate request inputs (query length, project name, limit range)

**Non-Goals:**
- Authentication or authorization (localhost-only, same as current server)
- Middleware pipeline (CORS, logging, rate limiting)
- Watch mode integration (watcher status field in metrics returns placeholder)
- Full-text index search (already handled by `SearchText`)
- TLS/HTTPS
- Change to the `/reindex` or `/health` endpoints

## Decisions

### 1. Endpoint naming and HTTP method

`POST /search` accepts a JSON body with `query`, `project`, `limit`. POST chosen over GET because the request body carries structured parameters including an optional project name and limit. GET would require query-string encoding for all parameters, which is awkward for the `query` field (may contain spaces and special characters).

`POST /search/code` and `POST /search/docs` are convenience variants — `/search/code` is equivalent to `/search` with the code index (default behavior), and `/search/docs` searches the docs index only. These could be implemented as a single endpoint with a `source` field, but separate routes make intent clearer and match the existing MCP tool naming convention (`search_code`, `search_docs`).

**Alternatives considered:**
- `GET /search?q=...&limit=...` — simpler but query encoding is fragile
- Single `/search` endpoint with a `source` field — reduces route count but obscures intent
- GraphQL — overkill for 3 queries

### 2. Search implementation

The search handler delegates to `storage.SearchSemantic` for vector similarity search and `storage.SearchText` for text fallback, exactly as the MCP handlers do. The handler uses `CollapseFileResults` to produce the same deduplicated, merged output format the MCP tools emit. An embedding function callback (similar to `ReindexFunc`) is injected into the server to keep the embedding provider decoupled from the HTTP layer.

**Alternatives considered:**
- Reimplement search logic in the server — code duplication, maintenance risk
- Proxy to MCP tools — circular dependency, no benefit

### 3. Request validation

Manual validation in the handler (no external library). Three checks:
- `query` is required, 1–500 characters
- `project` (optional) is sanitized: alphanumeric + hyphens, max 128 chars
- `limit` (optional) defaults to 10, clamped to 1–100

`go-playground/validator` would add a dependency for three simple checks. Manual validation keeps the binary lean.

**Alternatives considered:**
- `go-playground/validator` — expressive but overkill for 3 fields
- No validation — risk of SQL injection or resource exhaustion from large limits

### 4. Error response format

New endpoints use `{"error": "message", "code": "ERROR_CODE"}` with appropriate HTTP status codes. This is a cleaner format than the current `{"status": "error", "message": "..."}`. Existing endpoints (`/health`, `/reindex`) keep their current format — no breaking change. If a future change normalizes all errors, that's handled separately.

Error codes: `INVALID_QUERY`, `INVALID_PROJECT`, `INVALID_LIMIT`, `PROJECT_NOT_FOUND`, `INTERNAL_ERROR`.

**Alternatives considered:**
- Retrofit new format to `/health` and `/reindex` — unnecessary churn, breaking change
- Keep old format everywhere — inconsistent with modern API conventions

### 5. Metrics and status data sources

`GET /metrics` aggregates data from:
- `storage.Stats()` for chunk count, file count, db size, embedding model info
- `storage.GetIndexMetadata()` for provider, model, dimensions, last updated
- Server uptime (tracked internally with a `startTime` field)
- Watcher status: a placeholder field returning `"disabled"` since watch mode is not yet implemented

`GET /status/:project` uses `storage.Stats()` after opening a connection for the named project via `ProjectManager`. Returns a subset: project name, chunk count, file count, last modified.

## Risks / Trade-offs

- **Embedding model must be available**: search endpoints require an embedding provider. If the provider fails, the handler falls back to `SearchText` (same as MCP tools). [Risk: no embedding model loaded] → Fall back to text search with a degraded flag in the response.
- **No concurrent search serialization**: unlike `/reindex`, search requests are not serialized. SQLite handles concurrent reads natively (WAL mode). [Risk: write contention during concurrent reindex+search] → SQLite WAL mode ensures readers don't block writers. Acceptable for localhost-only workload.
- **Project name in URL path**: `/status/:project` uses a URL path parameter. Project names may contain characters that need URL encoding. [Risk: encoding issues] → Sanitize project name on the server side before lookup, same regex as request validation.
- **Metrics endpoint is unauthenticated**: any local process can query /metrics. [Risk: information disclosure] → Mitigated by binding to 127.0.0.1 only. Localhost-only is the existing security model.
