## ADDED Requirements

### Requirement: The system SHALL expose a documentation search MCP tool
The system SHALL provide a `search_docs` tool in the MCP server that searches only documentation files indexed under the same project scope.

#### Scenario: MCP client lists available tools
- **WHEN** an MCP client requests the available tools
- **THEN** the server SHALL advertise `search_docs` in addition to `search_code` and `index_project`

#### Scenario: MCP client calls documentation search
- **WHEN** an MCP client calls `search_docs` with a query
- **THEN** the server SHALL search only the documentation index (`<name>-docs.db`) and return results in the same format as `search_code` with file path, line ranges, signatures, and relevance

#### Scenario: Documentation search on unindexed project
- **WHEN** an MCP client calls `search_docs` for a project that has no documentation index
- **THEN** the system SHALL return guidance indicating that documentation indexing is required (`IDX_DOCS_MISSING`)

#### Scenario: Documentation search returns results
- **WHEN** `search_docs` finds matching documentation chunks
- **THEN** each result SHALL include file path, line ranges, section context (heading or paragraph), and relevance score

#### Scenario: Documentation search with path scoping
- **WHEN** an MCP client calls `search_docs` with a `path` parameter
- **THEN** the search SHALL be scoped to the specified path within the documentation index