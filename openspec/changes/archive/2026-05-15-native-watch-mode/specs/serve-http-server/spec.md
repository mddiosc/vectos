## MODIFIED Requirements

### Requirement: HTTP server startup and lifecycle
The system SHALL provide a `vectos serve` command that starts a persistent HTTP server on localhost. The server SHALL keep the embedding model and SQLite connections in memory for the duration of the process. When `--watch` is enabled (default true), the server SHALL start a filesystem watcher that monitors the project directory and triggers automatic incremental reindexes on file changes.

#### Scenario: Start server on default port
- **WHEN** `vectos serve` is executed without port configuration
- **THEN** the server starts on port 7438, starts a filesystem watcher with default options (500ms debounce, `.git,node_modules,*.lock` ignored), and logs a startup message indicating the port and watch status

#### Scenario: Start server on custom port
- **WHEN** `vectos serve` is executed with `--port 8080` or `VECTOS_PORT=8080` env var
- **THEN** the server starts on the specified port

#### Scenario: Port already in use
- **WHEN** the configured port is already occupied by another process
- **THEN** the server SHALL exit with a clear error message indicating the port conflict

#### Scenario: Graceful shutdown
- **WHEN** the server receives SIGTERM or SIGINT
- **THEN** the server SHALL stop the filesystem watcher, close all SQLite connections, stop accepting new requests, finish in-flight requests, and exit cleanly

#### Scenario: Server started with watch disabled
- **WHEN** `vectos serve` is executed with `--watch=false`
- **THEN** the server starts without a filesystem watcher and relies solely on manual `/reindex` calls

### Requirement: Incremental reindex endpoint
The server SHALL expose a `POST /reindex` endpoint that accepts a JSON body and triggers an incremental reindex of changed files. This endpoint SHALL coexist with automatic watcher-triggered reindexes; both SHALL use the same serialized reindex path.

#### Scenario: Reindex with changed paths
- **WHEN** a POST request is made to `/reindex` with body `{"path": "/project/dir", "changed": "src/main.go,src/util.go"}`
- **THEN** the server SHALL reindex only the specified changed files and return `{"status": "ok", "files_indexed": 2, "chunks_indexed": <N>}`

#### Scenario: Reindex with project scoping
- **WHEN** a POST request is made to `/reindex` with body `{"path": "/workspace", "project": "my-app", "changed": "src/main.go"}`
- **THEN** the server SHALL resolve the Nx project scope and reindex within that project's roots

#### Scenario: Reindex with docs flag
- **WHEN** a POST request is made to `/reindex` with body `{"path": "/project", "docs": true, "changed": "README.md"}`
- **THEN** the server SHALL reindex the changed files in the docs database

#### Scenario: Reindex without changed paths (full reindex)
- **WHEN** a POST request is made to `/reindex` with body `{"path": "/project/dir"}`
- **THEN** the server SHALL perform a full reindex of all indexable files in the project

#### Scenario: Reindex with invalid path
- **WHEN** a POST request is made to `/reindex` with a non-existent path
- **THEN** the server SHALL return HTTP 400 with `{"status": "error", "message": "path does not exist"}`

#### Scenario: Concurrent reindex requests
- **WHEN** multiple `/reindex` requests arrive while a reindex is in progress
- **THEN** the server SHALL serialize the requests (process one at a time) and return results for each

#### Scenario: Manual reindex during watcher-triggered reindex
- **WHEN** a manual `POST /reindex` request arrives while a watcher-triggered reindex is in progress
- **THEN** the server SHALL serialize the requests under the same mutex and process them sequentially

## ADDED Requirements

### Requirement: Watcher-triggered reindex logging
The server SHALL emit log output when the filesystem watcher detects changes and triggers a reindex, including the number of changed files detected and the debounce window used.

#### Scenario: Watcher detects a single file change
- **WHEN** the watcher triggers a reindex for a single changed file
- **THEN** the server SHALL log a message indicating "watcher detected file change" with the file path

#### Scenario: Watcher detects multiple file changes in a batch
- **WHEN** the watcher triggers a reindex for multiple changed files after debounce
- **THEN** the server SHALL log a message indicating "watcher detected N file changes" with the count of changed files
