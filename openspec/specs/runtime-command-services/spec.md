## ADDED Requirements

### Requirement: Runtime command logic SHALL be organized by operational concern
The system SHALL organize command runtime logic so indexing, search/status, and shared runtime helpers are separated into maintainable groupings.

#### Scenario: A developer modifies command runtime behavior
- **WHEN** a developer changes indexing, search, or status runtime logic
- **THEN** the relevant operational flow SHALL be locatable without navigating a single oversized command file

### Requirement: Runtime refactors SHALL preserve existing command semantics
The system SHALL preserve existing command behavior while runtime functions are extracted into smaller files.

#### Scenario: User runs an existing command after the refactor
- **WHEN** a user runs an existing command such as `vectos index`, `vectos search`, or `vectos status`
- **THEN** the command SHALL continue to behave equivalently after the refactor
