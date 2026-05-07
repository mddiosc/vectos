## ADDED Requirements

### Requirement: The MCP server SHALL expose a project listing tool
The system SHALL expose `list_projects` as an MCP tool so agents can discover available Nx project names before indexing or searching.

#### Scenario: MCP client lists available tools includes list_projects
- **WHEN** an MCP client requests the available tools
- **THEN** the server SHALL return at least `search_code`, `search_docs`, `index_project`, and `list_projects`

#### Scenario: MCP client calls list_projects in an Nx workspace
- **WHEN** an MCP client calls `list_projects` inside an Nx workspace
- **THEN** the server SHALL return a JSON object with an array of sorted project names
