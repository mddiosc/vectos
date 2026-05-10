## Purpose
Define how Vectos CLI runtime command logic is organized and preserved as commands are split by operational concern.

## Requirements

### Requirement: Runtime command logic SHALL be organized by operational concern
The system SHALL organize command runtime logic so indexing, search/status, serving, and shared runtime helpers are separated into maintainable groupings. The `serve` command SHALL be added alongside existing commands.

#### Scenario: A developer modifies command runtime behavior
- **WHEN** a developer changes indexing, search, status, or serving runtime logic
- **THEN** the relevant operational flow SHALL be locatable without navigating a single oversized command file

#### Scenario: User runs the serve command
- **WHEN** a user runs `vectos serve`
- **THEN** the system SHALL start an HTTP server on port 7438 (or configured port) that keeps the embedding model in memory and exposes `/health` and `/reindex` endpoints

### Requirement: Runtime refactors SHALL preserve existing command semantics
The system SHALL preserve existing command behavior while runtime functions are extracted into smaller files. The new `serve` command SHALL not affect existing commands.

#### Scenario: User runs an existing command after the refactor
- **WHEN** a user runs an existing command such as `vectos index`, `vectos search`, `vectos status`, or `vectos mcp`
- **THEN** the command SHALL continue to behave equivalently after the addition of the `serve` command
