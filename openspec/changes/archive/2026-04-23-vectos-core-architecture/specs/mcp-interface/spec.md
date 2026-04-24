## ADDED Requirements

### Requirement: The system SHALL expose an MCP server over stdio
The system SHALL provide a Model Context Protocol server over stdio so external agents can discover and call Vectos tools.

#### Scenario: MCP client initializes successfully
- **WHEN** an MCP-compatible client connects to the server
- **THEN** the server SHALL complete the MCP initialization handshake and advertise tool capabilities

### Requirement: The MCP server SHALL expose code search and indexing tools
The system SHALL expose tools for code search and project indexing through MCP.

#### Scenario: MCP client lists available tools
- **WHEN** an MCP client requests the available tools
- **THEN** the server SHALL return at least `search_code` and `index_project`

#### Scenario: MCP client calls code search
- **WHEN** an MCP client calls `search_code` with a query
- **THEN** the server SHALL execute the search against the active project index and return the result content in MCP tool result format

#### Scenario: MCP client calls project indexing
- **WHEN** an MCP client calls `index_project` with a file or directory path
- **THEN** the server SHALL index the requested path and return a summary of the indexing operation
