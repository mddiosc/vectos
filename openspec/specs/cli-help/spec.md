## ADDED Requirements

### Requirement: CLI help SHALL document expanded setup behavior
Vectos CLI help SHALL describe the validated setup targets and the uninstall form of the setup command.

#### Scenario: Show setup help
- **WHEN** the user runs `vectos help setup` or `vectos setup --help`
- **THEN** the help output SHALL mention `opencode`, `claude`, and `codex` as validated targets and document `--uninstall`

### Requirement: Vectos CLI help SHALL advertise the doctor command
The global help output SHALL list `doctor` alongside the other top-level commands.

#### Scenario: User reads global help
- **WHEN** the user runs `vectos help` or `vectos --help`
- **THEN** the output SHALL include `doctor` with a short description of the diagnostic workflow

#### Scenario: User reads doctor-specific help
- **WHEN** the user runs `vectos help doctor` or `vectos doctor --help`
- **THEN** the output SHALL describe the checks performed by the command and any supported flags
