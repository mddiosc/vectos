## ADDED Requirements

### Requirement: Vectos SHALL support setup for multiple agent clients
Vectos SHALL provide setup automation through agent-specific adapters for supported clients.

#### Scenario: Configure a supported agent
- **WHEN** the user runs `vectos setup <agent>` for a supported agent
- **THEN** Vectos SHALL create or update that agent's configuration with a valid MCP entry for Vectos

#### Scenario: Reject unvalidated agent target
- **WHEN** the user requests setup for an agent target that is not validated in the current implementation phase
- **THEN** Vectos SHALL fail with a clear unsupported-or-unvalidated-agent error

#### Scenario: Reject unsupported agent target
- **WHEN** the user requests setup for an unsupported agent
- **THEN** Vectos SHALL fail with a clear unsupported-agent error
