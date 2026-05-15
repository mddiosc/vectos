## Why

`vectos serve` provides a persistent HTTP server for low-latency reindexing, but agents must still explicitly trigger `/reindex` on every file change. Agents forget to reindex and get stale search results — this is fragile and undermines the value of warm indexing. A native filesystem watcher would make the index stay in sync automatically, eliminating a manual step that agents routinely get wrong.

## What Changes

- **Add `fsnotify` watcher to `vectos serve`** — On startup, the server optionally starts a filesystem watcher on the project directory that monitors for file create, modify, delete, and rename events.
- **Add file-hash tracking in SQLite** — Store SHA256 hashes of indexed files so change events can be verified (debouncing duplicate/incomplete events). Only re-index files whose content actually changed.
- **Add `--watch` flags to `vectos serve`** — `--watch` (default on), `--watch-debounce` (default 500ms), `--watch-ignore` (comma-separated globs, default `.git,node_modules,*.lock`).
- **Graceful watcher lifecycle** — Batch rapid changes via debounce, handle file deletions (remove chunks), handle directory additions/removals, respect `.gitignore` patterns, stop watcher on server shutdown.

## Capabilities

### New Capabilities
- `watch-mode`: Filesystem watching that keeps the project index in sync automatically by detecting file changes during `vectos serve`, verifying content changes via file hashes, debouncing rapid events, and triggering incremental reindexes without manual intervention.

### Modified Capabilities
- `serve-http-server`: Server startup gains watcher initialization with `--watch`, `--watch-debounce`, and `--watch-ignore` configuration. Graceful shutdown must also stop the watcher. The `/reindex` endpoint continues to work alongside automatic watcher-triggered reindexes.
- `code-indexing`: Indexing gains file-hash storage (SHA256 of indexed file content) and incremental reindex-by-hash capability. File deletions from the filesystem must trigger chunk removal.
- `runtime-command-services`: The `serve` command gains `--watch`, `--watch-debounce`, and `--watch-ignore` CLI flags alongside existing `--port` and `--project-base-dir`.

## Impact

- **New dependency**: `github.com/fsnotify/fsnotify` — standard Go filesystem notification library
- **Affected code**: `internal/server/server.go` (watcher init/lifecycle), `internal/storage/sqlite.go` (file hash columns), `internal/indexer/` (incremental reindex by hash, deletion handling), `cmd/vectos/cli_flags.go` (new watch flags), `cmd/vectos/cli_dispatch.go` (pass new flags to serve)
- **Database migration**: New `file_hash` column in chunks table (or separate file-tracking table)
- **Backward compatible**: Watcher defaults on for `vectos serve` but is optional (`--watch=false`). CLI `vectos index` unaffected. `/reindex` endpoint unaffected.
