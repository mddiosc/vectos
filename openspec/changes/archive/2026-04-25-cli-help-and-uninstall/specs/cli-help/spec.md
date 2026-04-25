## ADDED Requirements

### Requirement: Vectos CLI SHALL expose help for all subcommands
Vectos SHALL print usage and subcommand descriptions when invoked with no arguments, `help`, `--help`, or `-h`.

#### Scenario: Global help with no arguments
- **WHEN** the user runs `vectos` with no arguments
- **THEN** Vectos SHALL print global usage including all available subcommands and exit

#### Scenario: Global help via flag
- **WHEN** the user runs `vectos --help` or `vectos -h`
- **THEN** Vectos SHALL print the same global usage output and exit with code 0

#### Scenario: Global help via subcommand
- **WHEN** the user runs `vectos help`
- **THEN** Vectos SHALL print global usage and exit with code 0

#### Scenario: Per-subcommand help
- **WHEN** the user runs `vectos <subcommand> --help` or `vectos help <subcommand>`
- **THEN** Vectos SHALL print usage specific to that subcommand including its flags and exit with code 0

### Requirement: Vectos CLI SHALL use English for all user-visible output
All user-visible CLI output including usage text, error messages, success messages, and progress output SHALL be written in English.

#### Scenario: English error messages
- **WHEN** a CLI command encounters an error
- **THEN** the error message shown to the user SHALL be in English

#### Scenario: English success and progress output
- **WHEN** a CLI command completes successfully
- **THEN** progress and result messages SHALL be in English
