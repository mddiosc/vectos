## MODIFIED Requirements

### Requirement: The MCP server SHALL expose code search and indexing tools
The system SHALL expose tools for code search and project indexing through MCP.

#### Scenario: MCP client lists available tools
- **WHEN** an MCP client requests the available tools
- **THEN** the server SHALL return at least `search_code`, `search_docs`, and `index_project`

#### Scenario: MCP client calls code search
- **WHEN** an MCP client calls `search_code` with a query
- **THEN** the server SHALL execute the search against the active project index and return the result content in MCP tool result format with enough ranking metadata for an agent to choose what to inspect next

#### Scenario: MCP client calls docs search
- **WHEN** an MCP client calls `search_docs` with a query
- **THEN** the server SHALL search the documentation index and return results in the same format as `search_code`

#### Scenario: MCP client calls project indexing
- **WHEN** an MCP client calls `index_project` with a file or directory path
- **THEN** the server SHALL index the requested path and return a summary of the indexing operation

#### Scenario: MCP client calls project indexing for documentation
- **WHEN** an MCP client calls `index_project` with `docs: true`
- **THEN** the server SHALL index only documentation files into a separate docs database

#### Scenario: MCP client calls project indexing for changed project content
- **WHEN** an MCP client calls `index_project` for a scope that already has indexed content
- **THEN** the system SHALL be able to refresh only the changed files and return a summary of the applied indexing update

### Requirement: MCP search failures SHALL suggest the next useful action
The system SHALL provide explicit recovery guidance when MCP search cannot return useful results because the project is missing an index or requires refresh.

#### Scenario: Project is not indexed
- **WHEN** an MCP client calls `search_code` for a project scope that has no usable index
- **THEN** the system SHALL set the `guidance` field to `IDX_MISSING` and the `next_action` field to the command required to index the project

#### Scenario: Project index is stale or incomplete
- **WHEN** an MCP client calls `search_code` and the system can determine that the available project index is stale or incomplete
- **THEN** the system SHALL set the `guidance` field to `IDX_STALE` and the `next_action` field to the command required to refresh the index

#### Scenario: Code search returns 0 but docs index exists
- **WHEN** an MCP client calls `search_code` and the result count is 0 but a documentation index exists with chunks
- **THEN** the system SHALL set the `guidance` field to `TRY_DOCS` and the `next_action` field to suggest the `search_docs` tool or `index_project` with `docs: true`

#### Scenario: Docs search called but docs index is empty
- **WHEN** an MCP client calls `search_docs` for a project scope that has no documentation index
- **THEN** the system SHALL set the `guidance` field to `IDX_DOCS_MISSING` and the `next_action` field to the command required to index documentation