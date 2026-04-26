## ADDED Requirements

### Requirement: MCP server setup SHALL be isolated from the CLI entrypoint
The system SHALL structure MCP server startup and tool registration so they are not embedded inline in the central CLI entrypoint file.

#### Scenario: Developer updates MCP server behavior
- **WHEN** a developer changes MCP server startup or tool registration
- **THEN** the relevant MCP setup flow SHALL be locatable without navigating unrelated CLI command code

### Requirement: MCP refactors SHALL preserve tool compatibility
The system SHALL preserve the existing MCP tool names, request shapes, and response behavior while isolating MCP handlers.

#### Scenario: Existing agent calls MCP search or indexing tools
- **WHEN** an existing agent calls `search_code` or `index_project`
- **THEN** the MCP tool surface SHALL remain compatible after the refactor
