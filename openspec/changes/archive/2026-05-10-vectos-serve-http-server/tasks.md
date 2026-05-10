## 1. HTTP Server Core

- [x] 1.1 Create `internal/server/server.go` with `ServeConfig` struct holding embedding provider, project base dir, port, and cached SQLite connections
- [x] 1.2 Implement `NewServer(cfg ServeConfig) *Server` that initializes the HTTP mux, registers routes, and pre-loads the embedding model
- [x] 1.3 Implement `GET /health` handler returning `{"status": "ok"}` (200) or `{"status": "starting"}` (503) based on model readiness
- [x] 1.4 Implement `POST /reindex` handler accepting JSON body `{path, changed?, project?, docs?}` and triggering incremental or full reindex
- [x] 1.5 Add request serialization via mutex/channel so concurrent `/reindex` requests are processed one at a time
- [x] 1.6 Implement graceful shutdown on SIGTERM/SIGINT: close cached SQLite connections, stop HTTP server, finish in-flight requests

## 2. CLI Integration

- [x] 2.1 Add `serve` command to `cmd/vectos/cli_dispatch.go` routing to `runServeCommand`
- [x] 2.2 Add serve flags to `cmd/vectos/cli_flags.go`: `--port` (default 7438), `--project-base-dir`
- [x] 2.3 Create `cmd/vectos/commands_serve.go` with `runServe()` that resolves config, creates server, and starts listening
- [x] 2.4 Add `serve` subcommand help text to `cmd/vectos/cli_help.go`
- [x] 2.5 Add `serve` to `SupportedAgents()` or equivalent command listing

## 3. Reindex Logic Reuse

- [x] 3.1 Extract shared reindex logic from `cmd/vectos/commands_index.go` into a callable function in `internal/indexer/` or `internal/server/` that both CLI and HTTP server can use
- [x] 3.2 Ensure the extracted function accepts `projectBaseDir`, `embedConfig`, `path`, `changedPaths`, `projectName`, `docsOnly` parameters
- [x] 3.3 Verify existing `vectos index` CLI still works identically after extraction

## 4. OpenCode Plugin Refactor

- [x] 4.1 Replace `Bun.spawnSync()` calls in `internal/setup/plugins/vectos.ts` with `fetch()` calls to `http://127.0.0.1:${VECTOS_PORT}/reindex`
- [x] 4.2 Add `isVectosRunning()` health check function using `fetch()` to `GET /health` with 500ms timeout
- [x] 4.3 Add auto-start logic: if health check fails, spawn `vectos serve` via `Bun.spawn()` with detached stdio, wait 500ms, then retry
- [x] 4.4 Remove `extractProjectName()` helper (no longer needed — project resolution happens server-side)
- [x] 4.5 Update `VECTOS_PORT` env var support (default 7438) and `VECTOS_BIN` for server binary path
- [x] 4.6 Rebuild and reinstall plugin via `vectos setup opencode`

## 5. Setup Integration

- [x] 5.1 Update `internal/setup/opencode.go` to embed the updated plugin source (already uses `//go:embed`)
- [x] 5.2 Verify `vectos setup opencode` installs the updated plugin correctly
- [x] 5.3 Verify `vectos setup opencode --uninstall` removes the plugin correctly

## 6. Testing & Verification

- [x] 6.1 Write unit tests for HTTP handlers (`/health`, `/reindex`) in `internal/server/server_test.go`
- [x] 6.2 Write integration test: start server, send `/reindex` request, verify response
- [x] 6.3 Test concurrent `/reindex` requests are serialized correctly
- [x] 6.4 Test graceful shutdown closes SQLite connections
- [x] 6.5 Manual test: start `vectos serve`, trigger reindex from plugin, verify no TUI pollution
- [x] 6.6 Manual test: plugin auto-starts server if not running