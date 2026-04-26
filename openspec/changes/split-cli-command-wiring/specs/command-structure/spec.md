## ADDED Requirements

### Requirement: The CLI entrypoint SHALL delegate command wiring concerns cleanly
The system SHALL structure the CLI entrypoint so that startup, command help, flag wiring, and command dispatch are separated into maintainable responsibilities.

#### Scenario: Main entrypoint starts the CLI
- **WHEN** the CLI process starts
- **THEN** the top-level entrypoint SHALL initialize shared configuration and delegate command wiring rather than embedding all command logic inline

### Requirement: CLI refactors SHALL preserve current command behavior
The system SHALL preserve the existing command names, help text intent, and flag behavior while restructuring command wiring.

#### Scenario: User requests help for an existing command
- **WHEN** a user runs an existing help flow such as `vectos help search` or `vectos search --help`
- **THEN** the CLI SHALL continue to expose the same command and flag behavior after the refactor
