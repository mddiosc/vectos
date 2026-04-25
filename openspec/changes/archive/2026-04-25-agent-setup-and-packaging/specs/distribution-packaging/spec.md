## ADDED Requirements

### Requirement: Vectos SHALL support installation as a real CLI
Vectos SHALL support an installation workflow where users can run `vectos` directly instead of invoking a project-local binary path.

#### Scenario: Run globally installed Vectos
- **WHEN** the user builds Vectos from source and installs the resulting binary into their shell path
- **THEN** the `vectos` command SHALL be available globally in the shell

### Requirement: Vectos SHALL document the supported installation workflow clearly
Vectos SHALL document the supported source-based installation workflow and its current constraints.

#### Scenario: Install from source
- **WHEN** a user follows the documented build-and-install steps
- **THEN** they SHALL be able to use `vectos` globally without invoking `./vectos`
