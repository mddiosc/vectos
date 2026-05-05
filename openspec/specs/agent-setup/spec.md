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

### Requirement: Vectos-managed setup guidance SHALL reflect current retrieval workflows
When setup manages a global guidance block for a supported agent, that guidance SHALL stay aligned with the current Vectos retrieval surface and refresh workflows.

#### Scenario: Managed guidance prefers source code, docs, and incremental refresh
- **WHEN** Vectos creates or updates a managed guidance block for a supported agent
- **THEN** that block SHALL direct the agent to prefer Vectos code search and docs search tools before broad file-search tools, and SHALL mention incremental reindex using changed file paths when the agent edits files during a task

#### Scenario: Managed guidance targets the correct global file per agent
- **WHEN** the user runs `vectos setup opencode`, `vectos setup claude`, or `vectos setup codex`
- **THEN** Vectos SHALL update only its managed guidance block inside that agent's global instructions surface (`~/.config/opencode/AGENTS.md`, `~/.claude/CLAUDE.md`, or `~/.codex/AGENTS.md` respectively) without overwriting unrelated user guidance
