## ADDED Requirements

### Requirement: The system SHALL expose a documentation search tool
The system SHALL provide a `search_docs` MCP tool that searches project documentation separately from code.

#### Scenario: MCP client lists tools
- **WHEN** an MCP-compatible client requests the available tools
- **THEN** the server SHALL advertise `search_docs` alongside `search_code` and `index_project`

#### Scenario: MCP client calls search_docs
- **WHEN** an MCP client calls `search_docs` with a query
- **THEN** the server SHALL search the documentation index and return results in the same `mcpSearchFileResult` format as `search_code`

#### Scenario: Docs search returns results
- **WHEN** an MCP client calls `search_docs` and matching documentation chunks are found
- **THEN** the server SHALL return file_path, line_ranges, signatures, and relevance for each result in the same format as code search

#### Scenario: Docs search on unindexed documentation
- **WHEN** an MCP client calls `search_docs` for a project scope that has no documentation index
- **THEN** the system SHALL set the `guidance` field to `IDX_DOCS_MISSING` and the `next_action` field to suggest `index_project` with `docs: true`

#### Scenario: Docs search with path scoping
- **WHEN** an MCP client calls `search_docs` with a `path` or `project` parameter
- **THEN** the server SHALL scope the search to the specified project or directory path

#### Scenario: Docs search with no matches
- **WHEN** an MCP client calls `search_docs` and no documentation chunks match the query
- **THEN** the server SHALL return an empty results array with no guidance code