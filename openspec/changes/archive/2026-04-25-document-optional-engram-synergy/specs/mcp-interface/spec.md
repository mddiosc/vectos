## MODIFIED Requirements

### Requirement: The MCP server SHALL expose code search and indexing tools
The system SHALL expose tools for code search and project indexing through MCP.

#### Scenario: Agent guidance for mixed memory and code workflows
- **WHEN** an MCP-compatible agent has access to both session-memory tools and Vectos MCP tools
- **THEN** the recommended guidance SHALL prefer memory tools for prior decisions and Vectos tools for current code context without implying that Vectos depends on the presence of memory tools
