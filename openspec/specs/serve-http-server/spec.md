## Purpose
Define the persistent localhost HTTP server used by agent clients to trigger silent reindexing while keeping embeddings and SQLite connections warm.

## Requirements

### Requirement: HTTP server startup and lifecycle
The system SHALL provide a `vectos serve` command that starts a persistent HTTP server on localhost. The server SHALL keep the embedding model and SQLite connections in memory for the duration of the process.

#### Scenario: Start server on default port
- **WHEN** `vectos serve` is executed without port configuration
- **THEN** the server starts on port 7438 and logs a startup message indicating the port

#### Scenario: Start server on custom port
- **WHEN** `vectos serve` is executed with `--port 8080` or `VECTOS_PORT=8080` env var
- **THEN** the server starts on the specified port

#### Scenario: Port already in use
- **WHEN** the configured port is already occupied by another process
- **THEN** the server SHALL exit with a clear error message indicating the port conflict

#### Scenario: Graceful shutdown
- **WHEN** the server receives SIGTERM or SIGINT
- **THEN** the server SHALL close all SQLite connections, stop accepting new requests, finish in-flight requests, and exit cleanly

### Requirement: Health check endpoint
The server SHALL expose a `GET /health` endpoint that returns HTTP 200 with `{"status": "ok"}` when the server is running and ready to accept requests.

#### Scenario: Server is healthy
- **WHEN** a GET request is made to `/health`
- **THEN** the response SHALL have status 200 and body `{"status": "ok"}`

#### Scenario: Server is starting up
- **WHEN** a request is made to `/health` before the embedding model is fully loaded
- **THEN** the response SHALL have status 503 and body `{"status": "starting"}`

### Requirement: Incremental reindex endpoint
The server SHALL expose a `POST /reindex` endpoint that accepts a JSON body and triggers an incremental reindex of changed files.

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

### Requirement: Embedding model persistence
The server SHALL load the embedding model once at startup and keep it in memory for all subsequent requests. The model SHALL NOT be reloaded between requests.

#### Scenario: First request after startup
- **WHEN** the first `/reindex` request arrives after server startup
- **THEN** the embedding model SHALL already be loaded (no cold-start delay)

#### Scenario: Subsequent requests reuse model
- **WHEN** multiple `/reindex` requests are made sequentially
- **THEN** each request SHALL use the same in-memory embedding model without reloading

### Requirement: SQLite connection caching
The server SHALL cache SQLite connections per project and keep them open for the duration of the server process. Connections SHALL be closed on graceful shutdown.

#### Scenario: First request for a project
- **WHEN** a `/reindex` request arrives for a project not yet seen
- **THEN** the server SHALL open a new SQLite connection, cache it, and use it for the request

#### Scenario: Subsequent request for same project
- **WHEN** a `/reindex` request arrives for a previously seen project
- **THEN** the server SHALL reuse the cached SQLite connection

#### Scenario: Server shutdown
- **WHEN** the server shuts down gracefully
- **THEN** all cached SQLite connections SHALL be closed


<!-- Added by native-watch-mode -->

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


<!-- Added by http-api-completion -->
## MODIFIED Requirements

### Requirement: HTTP server startup and lifecycle
The system SHALL provide a `vectos serve` command that starts a persistent HTTP server on localhost. The server SHALL keep the embedding model and SQLite connections in memory for the duration of the process. The server SHALL register routes for `/health`, `/reindex`, `/search`, `/search/code`, `/search/docs`, `/metrics`, and `/status/` (prefix).

#### Scenario: Start server on default port
- **WHEN** `vectos serve` is executed without port configuration
- **THEN** the server starts on port 7438 and logs a startup message indicating the port

#### Scenario: Start server on custom port
- **WHEN** `vectos serve` is executed with `--port 8080` or `VECTOS_PORT=8080` env var
- **THEN** the server starts on the specified port

#### Scenario: Port already in use
- **WHEN** the configured port is already occupied by another process
- **THEN** the server SHALL exit with a clear error message indicating the port conflict

#### Scenario: Graceful shutdown
- **WHEN** the server receives SIGTERM or SIGINT
- **THEN** the server SHALL close all SQLite connections, stop accepting new requests, finish in-flight requests, and exit cleanly

### Requirement: Health check endpoint
The server SHALL expose a `GET /health` endpoint that returns HTTP 200 with `{"status": "ok"}` when the server is running and ready to accept requests.

#### Scenario: Server is healthy
- **WHEN** a GET request is made to `/health`
- **THEN** the response SHALL have status 200 and body `{"status": "ok"}`

#### Scenario: Server is starting up
- **WHEN** a request is made to `/health` before the embedding model is fully loaded
- **THEN** the response SHALL have status 503 and body `{"status": "starting"}`

### Requirement: Incremental reindex endpoint
The server SHALL expose a `POST /reindex` endpoint that accepts a JSON body and triggers an incremental reindex of changed files.

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

### Requirement: Embedding model persistence
The server SHALL load the embedding model once at startup and keep it in memory for all subsequent requests. The model SHALL NOT be reloaded between requests.

#### Scenario: First request after startup
- **WHEN** the first request arrives after server startup
- **THEN** the embedding model SHALL already be loaded (no cold-start delay)

#### Scenario: Subsequent requests reuse model
- **WHEN** multiple requests are made sequentially
- **THEN** each request SHALL use the same in-memory embedding model without reloading

### Requirement: SQLite connection caching
The server SHALL cache SQLite connections per project and keep them open for the duration of the server process. Connections SHALL be closed on graceful shutdown.

#### Scenario: First request for a project
- **WHEN** a request arrives for a project not yet seen
- **THEN** the server SHALL open a new SQLite connection, cache it, and use it for the request

#### Scenario: Subsequent request for same project
- **WHEN** a request arrives for a previously seen project
- **THEN** the server SHALL reuse the cached SQLite connection

#### Scenario: Server shutdown
- **WHEN** the server shuts down gracefully
- **THEN** all cached SQLite connections SHALL be closed

## ADDED Requirements

### Requirement: Search endpoint family
The server SHALL register `POST /search`, `POST /search/code`, and `POST /search/docs` routes. These endpoints SHALL accept JSON bodies with `query`, optional `project`, and optional `limit` fields.

#### Scenario: Search routes registered at startup
- **WHEN** the server starts successfully
- **THEN** the `/search`, `/search/code`, and `/search/docs` routes SHALL be registered and accept POST requests

### Requirement: Metrics endpoint
The server SHALL register a `GET /metrics` route that returns runtime and index statistics as JSON.

#### Scenario: Metrics route registered at startup
- **WHEN** the server starts successfully
- **THEN** the `/metrics` route SHALL be registered and accept GET requests

### Requirement: Status endpoint
The server SHALL register a `GET /status/` route prefix that serves per-project index status.

#### Scenario: Status route registered at startup
- **WHEN** the server starts successfully
- **THEN** the `/status/` prefix SHALL be registered and serve project-specific status for GET requests
