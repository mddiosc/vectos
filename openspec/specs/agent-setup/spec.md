## ADDED Requirements

### Requirement: Vectos SHALL update managed guidance blocks without interactive confirmation
Vectos SHALL create or update its managed guidance block in a supported agent's global instructions file even when the calling terminal is non-interactive, and SHALL only omit the guidance block when explicitly requested.

#### Scenario: Existing global instructions are updated automatically
- **WHEN** the user runs `vectos setup opencode`, `vectos setup claude`, or `vectos setup codex` against an existing global instructions file that does not yet contain a Vectos-managed block
- **THEN** Vectos SHALL append its managed guidance block without requiring interactive confirmation

#### Scenario: Guidance updates can be skipped explicitly
- **WHEN** the user runs `vectos setup <agent> --no-guidance`
- **THEN** Vectos SHALL leave the agent's global instructions file unchanged while still configuring the Vectos MCP entry for that agent

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
When setup manages a global guidance block for a supported agent, that guidance SHALL stay aligned with the current Vectos retrieval surface and refresh workflows, including Nx project coverage behavior.

#### Scenario: Managed guidance prefers source code, docs, and incremental refresh
- **WHEN** Vectos creates or updates a managed guidance block for a supported agent
- **THEN** that block SHALL direct the agent to prefer Vectos code search and docs search tools before broad file-search tools, and SHALL mention incremental reindex using changed file paths when the agent edits files during a task

#### Scenario: Managed guidance targets the correct global file per agent
- **WHEN** the user runs `vectos setup opencode`, `vectos setup claude`, or `vectos setup codex`
- **THEN** Vectos SHALL update only its managed guidance block inside that agent's global instructions surface (`~/.config/opencode/AGENTS.md`, `~/.claude/CLAUDE.md`, or `~/.codex/AGENTS.md` respectively) without overwriting unrelated user guidance

### Requirement: Managed guidance SHALL explain incremental reindex path semantics
The managed guidance injected into agent configuration files SHALL include explicit rules for using the `changed` parameter correctly, covering path format, renames, docs refresh, and full reindex triggers.

#### Scenario: Agent uses correct path format for changed files
- **WHEN** an agent reads the managed guidance
- **THEN** it SHALL find instructions to pass changed paths as absolute paths or as paths relative to the workspace root — not relative to an individual project or lib root

#### Scenario: Agent handles renames and moves correctly
- **WHEN** an agent reads the managed guidance
- **THEN** it SHALL find instructions to include both the old and the new path when a file is renamed or moved

#### Scenario: Agent knows when to refresh the docs index
- **WHEN** an agent reads the managed guidance
- **THEN** it SHALL find instructions to run a separate `index_project` with `docs: true` after editing documentation files

#### Scenario: Agent knows when to force a full reindex
- **WHEN** an agent reads the managed guidance
- **THEN** it SHALL find instructions to perform a full reindex after changing workspace-level configuration files such as `nx.json`, `project.json`, or lockfiles

#### Scenario: Agent understands Nx shared-lib reindex rules
- **WHEN** an agent reads the managed guidance
- **THEN** it SHALL find instructions explaining that editing a shared Nx lib may require refreshing the indexes of downstream projects that depend on it

#### Scenario: Managed guidance explains Nx lib coverage rules
- **WHEN** Vectos creates or updates a managed guidance block for an Nx workspace
- **THEN** that block SHALL state that all internal dependency libs are included by default, that only `type: "e2e"` projects are excluded by default, and that `VECTOS_NX_INCLUDE_E2E=1` overrides that exclusion
