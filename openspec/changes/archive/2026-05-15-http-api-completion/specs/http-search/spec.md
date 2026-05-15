## ADDED Requirements

### Requirement: HTTP search endpoint
The server SHALL expose a `POST /search` endpoint that accepts a JSON body with `query`, optional `project`, and optional `limit` fields, and returns ranked search results from the code index.

#### Scenario: Successful semantic search
- **WHEN** a POST request is made to `/search` with body `{"query": "how does auth work", "limit": 5}`
- **THEN** the server SHALL return HTTP 200 with a JSON array of results, each containing `file_path`, `file_name`, `language`, `relevance`, `line_ranges`, and `signatures`, sorted by relevance descending

#### Scenario: Search with project scoping
- **WHEN** a POST request is made to `/search` with body `{"query": "auth", "project": "my-app"}`
- **THEN** the server SHALL search only the index for project "my-app" and return scoped results

#### Scenario: Search defaults limit to 10
- **WHEN** a POST request is made to `/search` without a `limit` field
- **THEN** the server SHALL return at most 10 results

#### Scenario: Search with missing query
- **WHEN** a POST request is made to `/search` with body `{"limit": 5}` (no query)
- **THEN** the server SHALL return HTTP 400 with `{"error": "query is required", "code": "INVALID_QUERY"}`

#### Scenario: Search with empty query
- **WHEN** a POST request is made to `/search` with body `{"query": ""}`
- **THEN** the server SHALL return HTTP 400 with `{"error": "query is required", "code": "INVALID_QUERY"}`

#### Scenario: Search with query too long
- **WHEN** a POST request is made to `/search` with a query exceeding 500 characters
- **THEN** the server SHALL return HTTP 400 with `{"error": "query too long (max 500)", "code": "INVALID_QUERY"}`

#### Scenario: Search with invalid limit
- **WHEN** a POST request is made to `/search` with `{"query": "auth", "limit": 0}`
- **THEN** the server SHALL return HTTP 400 with `{"error": "limit must be between 1 and 100", "code": "INVALID_LIMIT"}`

#### Scenario: Search with limit exceeding maximum
- **WHEN** a POST request is made to `/search` with `{"query": "auth", "limit": 200}`
- **THEN** the server SHALL return HTTP 400 with `{"error": "limit must be between 1 and 100", "code": "INVALID_LIMIT"}`

#### Scenario: Search falls back to text search
- **WHEN** the embedding provider fails to generate a query vector
- **THEN** the server SHALL fall back to `SearchText` and return results, possibly with lower relevance

#### Scenario: Wrong HTTP method on search
- **WHEN** a GET request is made to `/search`
- **THEN** the server SHALL return HTTP 405 with `{"error": "method not allowed", "code": "METHOD_NOT_ALLOWED"}`

### Requirement: Code-specific search endpoint
The server SHALL expose a `POST /search/code` endpoint that searches only the code index (excluding documentation chunks).

#### Scenario: Code search excludes docs
- **WHEN** a POST request is made to `/search/code` with body `{"query": "error handling"}`
- **THEN** the server SHALL return results from the code index only, excluding chunks categorized as "docs"

#### Scenario: Code search accepts same parameters as /search
- **WHEN** a POST request is made to `/search/code` with body `{"query": "auth", "project": "my-app", "limit": 5}`
- **THEN** the server SHALL apply the same validation and scoping as `/search`

### Requirement: Documentation-specific search endpoint
The server SHALL expose a `POST /search/docs` endpoint that searches only the documentation index.

#### Scenario: Docs search returns documentation chunks
- **WHEN** a POST request is made to `/search/docs` with body `{"query": "getting started"}`
- **THEN** the server SHALL search the docs database and return documentation results

#### Scenario: Docs search with unindexed docs
- **WHEN** a POST request is made to `/search/docs` for a project that has no documentation index
- **THEN** the server SHALL return HTTP 404 with `{"error": "no documentation index found for project", "code": "PROJECT_NOT_FOUND"}`

#### Scenario: Docs search accepts same parameters
- **WHEN** a POST request is made to `/search/docs` with body `{"query": "setup", "project": "my-app"}`
- **THEN** the server SHALL apply the same validation as `/search`
