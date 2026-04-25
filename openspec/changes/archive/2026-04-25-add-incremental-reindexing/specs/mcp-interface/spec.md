## MODIFIED Requirements

### Requirement: The MCP server SHALL expose code search and indexing tools
The system SHALL expose tools for code search and project indexing through MCP.

#### Scenario: MCP client calls project indexing for changed project content
- **WHEN** an MCP client calls `index_project` for a scope that already has indexed content
- **THEN** the system SHALL be able to refresh only the changed files and return a summary of the applied indexing update
