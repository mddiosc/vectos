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
- **THEN** the server SHALL execute the search against the active project index and return the result content in MCP tool result format with enough ranking metadata for an agent to choose what to inspect next

#### Scenario: MCP client calls project indexing
- **WHEN** an MCP client calls `index_project` with a file or directory path
- **THEN** the server SHALL index the requested path and return a summary of the indexing operation

#### Scenario: MCP client calls project indexing for changed project content
- **WHEN** an MCP client calls `index_project` for a scope that already has indexed content
- **THEN** the system SHALL be able to refresh only the changed files and return a summary of the applied indexing update

#### Scenario: Agent guidance for mixed memory and code workflows
- **WHEN** an MCP-compatible agent has access to both session-memory tools and Vectos MCP tools
- **THEN** the recommended guidance SHALL prefer memory tools for prior decisions and Vectos tools for current code context without implying that Vectos depends on the presence of memory tools

### Requirement: MCP search results SHALL include concise actionable metadata
The system SHALL return MCP search results with concise metadata that helps an agent decide whether a result is worth reading.

#### Scenario: Search result includes relevance context
- **WHEN** the system returns an MCP search result
- **THEN** each result SHALL include at least the file path and enough concise metadata, such as rank, line range, chunk role, or short match context, to support agent decision-making

### Requirement: MCP search failures SHALL suggest the next useful action
The system SHALL provide explicit recovery guidance when MCP search cannot return useful results because the project is missing an index or requires refresh.

#### Scenario: Project is not indexed
- **WHEN** an MCP client calls `search_code` for a project scope that has no usable index
- **THEN** the system SHALL indicate that indexing is required and identify the relevant indexing action

#### Scenario: Project index is stale or incomplete
- **WHEN** an MCP client calls `search_code` and the system can determine that the available project index is stale or incomplete
- **THEN** the system SHALL indicate that a refresh is recommended and identify the relevant indexing action
