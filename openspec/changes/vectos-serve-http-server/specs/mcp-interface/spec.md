## MODIFIED Requirements

### Requirement: The MCP server SHALL expose code search and indexing tools

The system SHALL expose tools for code search and project indexing through MCP. The MCP server remains the primary interface for agent tools. The new HTTP server (`vectos serve`) provides a complementary API for programmatic reindex triggers from the OpenCode plugin, but does not replace MCP.

#### Scenario: MCP client calls code search
- **WHEN** an MCP client calls `search_code` with a query
- **THEN** the server SHALL execute the search against the active project index and return the result content in MCP tool result format with enough ranking metadata for an agent to choose what to inspect next

#### Scenario: MCP client calls project indexing
- **WHEN** an MCP client calls `index_project` with a file or directory path
- **THEN** the server SHALL index the requested path and return a summary of the indexing operation

#### Scenario: MCP client calls project indexing for changed project content
- **WHEN** an MCP client calls `index_project` for a scope that already has indexed content
- **THEN** the system SHALL be able to refresh only the changed files and return a summary of the applied indexing update