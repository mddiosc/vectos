## ADDED Requirements

### Requirement: Metrics endpoint
The server SHALL expose a `GET /metrics` endpoint that returns JSON with index statistics and runtime information.

#### Scenario: Metrics returns index stats
- **WHEN** a GET request is made to `/metrics`
- **THEN** the server SHALL return HTTP 200 with a JSON object containing `chunk_count`, `file_count`, `database_size_bytes`, `embedded_count`, `unembedded_count`

#### Scenario: Metrics includes embedding model info
- **WHEN** a GET request is made to `/metrics`
- **THEN** the response SHALL include `provider`, `model`, and `dimensions` fields from the index metadata

#### Scenario: Metrics includes uptime
- **WHEN** a GET request is made to `/metrics`
- **THEN** the response SHALL include `uptime_seconds` as a float representing seconds since server start

#### Scenario: Metrics includes watcher status placeholder
- **WHEN** a GET request is made to `/metrics`
- **THEN** the response SHALL include `watcher_status` with value `"disabled"` (watch mode not yet implemented)

#### Scenario: Metrics includes last index time
- **WHEN** a GET request is made to `/metrics`
- **THEN** the response SHALL include `last_index_time` from `index_metadata.updated_at` in ISO 8601 format

#### Scenario: Metrics on uninitialized index
- **WHEN** a GET request is made to `/metrics` and no index metadata exists
- **THEN** the server SHALL return HTTP 200 with `chunk_count: 0` and `provider: "unknown"`

#### Scenario: Wrong HTTP method on metrics
- **WHEN** a POST request is made to `/metrics`
- **THEN** the server SHALL return HTTP 405
