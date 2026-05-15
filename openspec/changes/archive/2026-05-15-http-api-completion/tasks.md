## 1. Shared Infrastructure

- [x] 1.1 Define error response struct (`ErrorResponse`) and helper (`writeError`) in a new file `internal/server/errors.go`, following the `{"error": "message", "code": "ERROR_CODE"}` format with appropriate HTTP status codes per the design (error codes: `INVALID_QUERY`, `INVALID_PROJECT`, `INVALID_LIMIT`, `PROJECT_NOT_FOUND`, `INTERNAL_ERROR`, `METHOD_NOT_ALLOWED`)
- [x] 1.2 Define the search request struct (`SearchRequest`) and search result struct (`SearchResult`) in `internal/server/types.go` with JSON tags matching the spec (`query`, `project`, `limit`; `file_path`, `file_name`, `language`, `relevance`, `line_ranges`, `signatures`)
- [x] 1.3 Implement request validation function (`validateSearchRequest`) in `internal/server/validation.go`: query required (1–500 chars), project optional (alphanumeric + hyphens, max 128 chars), limit optional (default 10, clamped to 1–100)
- [x] 1.4 Implement project name sanitization function (`sanitizeProject`) matching the regex from the design (alphanumeric + hyphens, max 128 chars)

## 2. Search Endpoints

- [x] 2.1 Implement `handleSearch` handler in `internal/server/search.go` that accepts a `Source` parameter (code or docs), validates the request body, generates an embedding vector via the injected `EmbedFunc`, calls `storage.SearchSemantic` or falls back to `storage.SearchText` when embedding fails, deduplicates with `CollapseFileResults`, and returns ranked results as JSON
- [x] 2.2 Implement exception handling in `handleSearch` for missing query → 400 `INVALID_QUERY`, empty query → 400 `INVALID_QUERY`, query too long → 400 `INVALID_QUERY`, limit out of range → 400 `INVALID_LIMIT`, invalid project name → 400 `INVALID_PROJECT`, wrong HTTP method → 405 `METHOD_NOT_ALLOWED`, internal errors → 500 `INTERNAL_ERROR`
- [x] 2.3 Implement `handleSearchCode` handler that calls `handleSearch` with source=code (excluding docs chunks per spec scenario "Code search excludes docs")
- [x] 2.4 Implement `handleSearchDocs` handler that calls `handleSearch` with source=docs, returning 404 `PROJECT_NOT_FOUND` when the project has no documentation index (per spec scenario "Docs search with unindexed docs")

## 3. Metrics Endpoint

- [x] 3.1 Add a `startTime time.Time` field to the server struct in `internal/server/server.go` and set it during server initialization to track uptime
- [x] 3.2 Implement `handleMetrics` handler in `internal/server/metrics.go` that aggregates data from `storage.Stats()` (chunk_count, file_count, database_size_bytes, embedded_count, unembedded_count) and `storage.GetIndexMetadata()` (provider, model, dimensions, updated_at as `last_index_time` in ISO 8601), computes `uptime_seconds` from `startTime`, returns `watcher_status: "disabled"` placeholder, and handles uninitialized index (chunk_count: 0, provider: "unknown") with HTTP 200
- [x] 3.3 Handle wrong HTTP method on `/metrics` → 405 per spec scenario

## 4. Project Status Endpoint

- [x] 4.1 Implement `handleStatus` handler in `internal/server/status.go` that extracts the project name from the URL path (`/status/:project`), sanitizes it with `sanitizeProject`, opens a storage connection via `ProjectManager`, and uses `storage.Stats()` to return `project`, `indexed: true/false`, `chunk_count`, `file_count`, `last_modified` (ISO 8601), and `database_path`
- [x] 4.2 Handle unindexed project → HTTP 200 with `{"project": "...", "indexed": false}` per spec scenario
- [x] 4.3 Handle invalid project name (path traversal, unsanitized characters) → 400 `INVALID_PROJECT` per spec scenario
- [x] 4.4 Handle wrong HTTP method on `/status/` prefix → 405 per spec scenario

## 5. Server Integration

- [x] 5.1 Inject the `EmbedFunc` callback (same signature as `ReindexFunc`) into the server struct to keep embedding decoupled from the HTTP layer, per design decision #2
- [x] 5.2 Register new routes in the server startup: `POST /search` → `handleSearchCode`, `POST /search/code` → `handleSearchCode`, `POST /search/docs` → `handleSearchDocs`, `GET /metrics` → `handleMetrics`, `GET /status/` (prefix) → `handleStatus`
- [x] 5.3 Ensure existing endpoints (`GET /health`, `POST /reindex`) retain their current error format (`{"status": "error", "message": "..."}`) unchanged — no breaking change, per design decision #4

## 6. Verification & Cleanup

- [x] 6.1 Run `go build ./...` to verify the project compiles without errors
- [x] 6.2 Run `go vet ./internal/server/...` to catch any issues
- [x] 6.3 Run existing tests with `go test ./internal/server/...` to ensure no regressions
- [x] 6.4 Manual smoke test: start the server with `vectos serve`, then curl `POST /search` with valid/invalid bodies, `GET /metrics`, and `GET /status/:project` to verify all spec scenarios pass
