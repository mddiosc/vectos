## ADDED Requirements

### Requirement: CLI help SHALL document expanded setup behavior
Vectos CLI help SHALL describe the validated setup targets and the uninstall form of the setup command.

#### Scenario: Show setup help
- **WHEN** the user runs `vectos help setup` or `vectos setup --help`
- **THEN** the help output SHALL mention `opencode`, `claude`, and `codex` as validated targets and document `--uninstall`
