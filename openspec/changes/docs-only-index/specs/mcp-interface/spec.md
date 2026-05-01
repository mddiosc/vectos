## MODIFIED Requirements

### Requirement: The MCP server SHALL expose code search and indexing tools
The system SHALL expose tools for code search and project indexing through MCP.

#### Scenario: MCP client lists available tools
- **WHEN** an MCP client requests the available tools
- **THEN** the server SHALL return `search_code`, `search_docs`, and `index_project`

#### Scenario: MCP client calls documentation search
- **WHEN** an MCP client calls `search_docs` with a query
- **THEN** the server SHALL search only the documentation index (`<name>-docs.db`) and return structured results using the same `mcpSearchFileResult` format as `search_code` (file path, line ranges, signatures, relevance)

## ADDED Requirements

### Requirement: MCP search failures SHALL suggest documentation search as alternative
The system SHALL provide guidance suggesting documentation search when code search yields no results but a documentation index exists.

#### Scenario: Code search returns no results but docs index exists
- **WHEN** an MCP client calls `search_code` and gets zero results, but a documentation index file exists on disk
- **THEN** the system SHALL include `TRY_DOCS` in the guidance field and suggest using `search_docs` in the next_action field

#### Scenario: Documentation index does not exist
- **WHEN** an MCP client calls `search_docs` and no documentation index exists
- **THEN** the system SHALL return `IDX_DOCS_MISSING` in guidance with instructions to run `index_project` with `docs: true`

### Requirement: index_project SHALL support documentation-only indexing
The system SHALL accept a `docs` parameter that when set to `true` indexes only documentation files.

#### Scenario: Index project with docs flag
- **WHEN** an MCP client calls `index_project` with `docs: true`
- **THEN** the system SHALL index only files with `category == "docs"` into the documentation database (`<name>-docs.db`)

#### Scenario: Index project without docs flag
- **WHEN** an MCP client calls `index_project` with `docs: false` or `docs` omitted
- **THEN** the system SHALL index only source files (excluding documentation) into the source code database (`<name>.db`)