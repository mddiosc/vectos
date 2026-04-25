## ADDED Requirements

### Requirement: Vectos SHALL support setup for multiple agent clients
Vectos SHALL provide setup automation through agent-specific adapters for validated clients, and SHALL also support uninstalling the Vectos-managed integration state for those clients.

#### Scenario: Configure a supported agent
 - **WHEN** the user runs `vectos setup <agent>` for a validated agent
- **THEN** Vectos SHALL create or update that agent's configuration with a valid MCP entry for Vectos

#### Scenario: Uninstall a supported agent integration
- **WHEN** the user runs `vectos setup <agent> --uninstall` for a validated agent
- **THEN** Vectos SHALL remove the Vectos-managed MCP entry and any Vectos-managed guidance block for that agent without deleting unrelated user configuration

#### Scenario: Reject unvalidated agent target
- **WHEN** the user requests setup for an agent target that is not validated in the current implementation phase
- **THEN** Vectos SHALL fail with a clear unsupported-or-unvalidated-agent error

#### Scenario: Reject unsupported agent target
- **WHEN** the user requests setup for an unsupported agent
- **THEN** Vectos SHALL fail with a clear unsupported-agent error

### Requirement: Vectos SHALL validate Claude Code and Codex setup targets
Vectos SHALL validate `claude` and `codex` as supported setup targets in the current implementation phase.

#### Scenario: Configure Claude Code
- **WHEN** the user runs `vectos setup claude`
- **THEN** Vectos SHALL add a user-scoped Vectos MCP server entry to Claude Code configuration and manage a Vectos guidance block in Claude's global instructions file

#### Scenario: Configure Codex
- **WHEN** the user runs `vectos setup codex`
- **THEN** Vectos SHALL add a Vectos MCP server entry to Codex configuration and manage a Vectos guidance block in Codex global instructions file
