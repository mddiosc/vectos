## MODIFIED Requirements

### Requirement: Runtime command logic SHALL be organized by operational concern

The system SHALL organize command runtime logic so indexing, search/status, serving, setup, and shared runtime helpers are separated into maintainable groupings. The `doctor` command SHALL be added alongside existing commands as a read-only diagnostic flow.

#### Scenario: A developer modifies command runtime behavior
- **WHEN** a developer changes indexing, search, status, serving, setup, or diagnostic runtime logic
- **THEN** the relevant operational flow SHALL be locatable without navigating a single oversized command file

#### Scenario: User runs the doctor command
- **WHEN** a user runs `vectos doctor`
- **THEN** the system SHALL print a read-only diagnostic report without mutating indexes or configuration

### Requirement: Runtime commands SHALL preserve existing command semantics

The system SHALL preserve existing command behavior while runtime functions are extracted into smaller files. The new `doctor` command SHALL not affect existing commands.

#### Scenario: User runs an existing command after the refactor
- **WHEN** a user runs an existing command such as `vectos index`, `vectos search`, `vectos status`, `vectos mcp`, `vectos serve`, or `vectos setup`
- **THEN** the command SHALL continue to behave equivalently after the addition of `doctor`
