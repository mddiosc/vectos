## ADDED Requirements

### Requirement: Project status endpoint
The server SHALL expose a `GET /status/:project` endpoint that returns the index status for a named project.

#### Scenario: Status for indexed project
- **WHEN** a GET request is made to `/status/my-app` and the project has an existing index
- **THEN** the server SHALL return HTTP 200 with JSON containing `project`, `indexed: true`, `chunk_count`, `file_count`, `last_modified` (ISO 8601), and `database_path`

#### Scenario: Status for unindexed project
- **WHEN** a GET request is made to `/status/unknown-project` and the project has no index
- **THEN** the server SHALL return HTTP 200 with `{"project": "unknown-project", "indexed": false}`

#### Scenario: Status with invalid project name
- **WHEN** a GET request is made to `/status/../../etc` (path traversal or unsanitized characters)
- **THEN** the server SHALL return HTTP 400 with `{"error": "invalid project name", "code": "INVALID_PROJECT"}`

#### Scenario: Wrong HTTP method on status
- **WHEN** a POST request is made to `/status/some-project`
- **THEN** the server SHALL return HTTP 405
