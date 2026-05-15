## MODIFIED Requirements

### Requirement: Runtime command logic SHALL be organized by operational concern
The system SHALL organize command runtime logic so indexing, search/status, serving, setup, and shared runtime helpers are separated into maintainable groupings. The `serve` and `doctor` commands SHALL be added alongside existing commands as new operational flows.

#### Scenario: A developer modifies command runtime behavior
- **WHEN** a developer changes indexing, search, status, serving, setup, or diagnostic runtime logic
- **THEN** the relevant operational flow SHALL be locatable without navigating a single oversized command file

#### Scenario: User runs the serve command
- **WHEN** a user runs `vectos serve`
- **THEN** the system SHALL start an HTTP server on port 7438 (or configured port) that keeps the embedding model in memory, starts a filesystem watcher by default (with `--watch`, `--watch-debounce`, `--watch-ignore` flags), and exposes `/health` and `/reindex` endpoints

#### Scenario: User runs the doctor command
- **WHEN** a user runs `vectos doctor`
- **THEN** the system SHALL print a read-only diagnostic report without mutating indexes or configuration

### Requirement: Runtime refactors SHALL preserve existing command semantics
The system SHALL preserve existing command behavior while runtime functions are extracted into smaller files. The new commands SHALL not affect existing commands.

#### Scenario: User runs an existing command after the refactor
- **WHEN** a user runs an existing command such as `vectos index`, `vectos search`, `vectos status`, `vectos mcp`, `vectos serve`, or `vectos setup`
- **THEN** the command SHALL continue to behave equivalently after the addition of new commands

## ADDED Requirements

### Requirement: Serve command supports watch configuration flags
The `vectos serve` command SHALL accept `--watch`, `--watch-debounce`, and `--watch-ignore` CLI flags to configure filesystem watching behavior.

#### Scenario: Serve command accepts --watch flag
- **WHEN** a user runs `vectos serve --watch=false`
- **THEN** the server SHALL start without the filesystem watcher

#### Scenario: Serve command accepts --watch-debounce flag
- **WHEN** a user runs `vectos serve --watch-debounce=1000`
- **THEN** the watcher debounce window SHALL be set to 1000ms

#### Scenario: Serve command accepts --watch-ignore flag
- **WHEN** a user runs `vectos serve --watch-ignore=.build,*.gen.go`
- **THEN** the watcher SHALL ignore paths matching `.build` or `*.gen.go`

#### Scenario: Serve command with default watch flags
- **WHEN** a user runs `vectos serve` without any watch flags
- **THEN** the watcher SHALL start with `--watch=true`, `--watch-debounce=500`, and `--watch-ignore=.git,node_modules,*.lock`
