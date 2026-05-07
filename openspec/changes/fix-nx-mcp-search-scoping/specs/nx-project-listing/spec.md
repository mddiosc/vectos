## ADDED Requirements

### Requirement: Vectos SHALL expose an MCP tool to list available Nx project names
The system SHALL provide a `list_projects` MCP tool that returns the names of all Nx projects in the workspace.

#### Scenario: List projects in an Nx workspace
- **WHEN** an MCP client calls `list_projects` with a path inside an Nx workspace
- **THEN** the server SHALL return a JSON object containing an array of project names sorted alphabetically

#### Scenario: List projects outside an Nx workspace
- **WHEN** an MCP client calls `list_projects` with a path that is not inside an Nx workspace
- **THEN** the server SHALL return a JSON object with an empty projects array (not an error)

#### Scenario: List projects with explicit path
- **WHEN** an MCP client calls `list_projects` with an explicit `path` parameter pointing to a directory inside an Nx workspace
- **THEN** the server SHALL discover and return projects from that workspace root

#### Scenario: List projects with no path defaults to working directory
- **WHEN** an MCP client calls `list_projects` without a `path` parameter
- **THEN** the server SHALL use the server's current working directory as the starting point for workspace detection
