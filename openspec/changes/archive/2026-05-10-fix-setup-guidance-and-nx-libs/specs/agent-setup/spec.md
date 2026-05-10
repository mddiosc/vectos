## ADDED Requirements

### Requirement: Vectos SHALL update managed guidance blocks without interactive confirmation
Vectos SHALL create or update its managed guidance block in a supported agent's global instructions file even when the calling terminal is non-interactive, and SHALL only omit the guidance block when explicitly requested.

#### Scenario: Existing global instructions are updated automatically
- **WHEN** the user runs `vectos setup opencode`, `vectos setup claude`, or `vectos setup codex` against an existing global instructions file that does not yet contain a Vectos-managed block
- **THEN** Vectos SHALL append its managed guidance block without requiring interactive confirmation

#### Scenario: Guidance updates can be skipped explicitly
- **WHEN** the user runs `vectos setup <agent> --no-guidance`
- **THEN** Vectos SHALL leave the agent's global instructions file unchanged while still configuring the Vectos MCP entry for that agent

## MODIFIED Requirements

### Requirement: Vectos-managed setup guidance SHALL reflect current retrieval workflows
When setup manages a global guidance block for a supported agent, that guidance SHALL stay aligned with the current Vectos retrieval surface and refresh workflows, including Nx project coverage behavior.

#### Scenario: Managed guidance prefers source code, docs, and incremental refresh
- **WHEN** Vectos creates or updates a managed guidance block for a supported agent
- **THEN** that block SHALL direct the agent to prefer Vectos code search and docs search tools before broad file-search tools, and SHALL mention incremental reindex using changed file paths when the agent edits files during a task

#### Scenario: Managed guidance targets the correct global file per agent
- **WHEN** the user runs `vectos setup opencode`, `vectos setup claude`, or `vectos setup codex`
- **THEN** Vectos SHALL update only its managed guidance block inside that agent's global instructions surface (`~/.config/opencode/AGENTS.md`, `~/.claude/CLAUDE.md`, or `~/.codex/AGENTS.md` respectively) without overwriting unrelated user guidance

#### Scenario: Managed guidance explains Nx lib coverage rules
- **WHEN** Vectos creates or updates a managed guidance block for an Nx workspace
- **THEN** that block SHALL state that all internal dependency libs are included by default, that only `type: "e2e"` projects are excluded by default, and that `VECTOS_NX_INCLUDE_E2E=1` overrides that exclusion
