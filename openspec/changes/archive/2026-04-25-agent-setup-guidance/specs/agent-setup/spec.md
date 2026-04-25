## ADDED Requirements

### Requirement: Vectos SHALL support optional global retrieval guidance for supported agents
When a supported agent client exposes a stable global guidance file or equivalent instruction surface, Vectos SHALL be able to install or update Vectos-managed guidance that biases the agent toward Vectos retrieval before broader file-search tools.

#### Scenario: Install default guidance for OpenCode
- **WHEN** the user runs `vectos setup opencode` and no global OpenCode guidance file exists yet
- **THEN** Vectos SHALL create a Vectos-managed global guidance block that tells OpenCode to prefer `vectos_search_code` first and `vectos_index_project` before generic fallback tools

#### Scenario: Preserve existing global guidance
- **WHEN** the user runs `vectos setup opencode` and a global OpenCode guidance file already exists without a Vectos-managed block
- **THEN** Vectos SHALL ask before appending its managed guidance block

#### Scenario: Update managed guidance idempotently
- **WHEN** the user runs `vectos setup opencode` and the Vectos-managed guidance block already exists
- **THEN** Vectos SHALL update only that managed block without duplicating it or overwriting unrelated user guidance
