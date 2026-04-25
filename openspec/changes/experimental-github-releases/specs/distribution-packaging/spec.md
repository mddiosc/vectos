## MODIFIED Requirements

### Requirement: Vectos SHALL support installation as a real CLI
Vectos SHALL support installation workflows where users can run `vectos` directly instead of invoking a project-local binary path.

#### Scenario: Run globally installed Vectos from source
- **WHEN** the user builds Vectos from source and installs the resulting binary into their shell path
- **THEN** the `vectos` command SHALL be available globally in the shell

#### Scenario: Run Vectos from an experimental GitHub release asset
- **WHEN** the user downloads a supported experimental GitHub release asset for a validated platform and installs the binary into their shell path
- **THEN** the `vectos` command SHALL be available globally in the shell without requiring a source build on that machine

### Requirement: Vectos SHALL document the supported installation workflow clearly
Vectos SHALL document the supported installation workflows and their current constraints for both source-based and experimental release-based installation.

#### Scenario: Install from source
- **WHEN** a user follows the documented build-and-install steps
- **THEN** they SHALL be able to use `vectos` globally without invoking `./vectos`

#### Scenario: Install from experimental GitHub release
- **WHEN** a user follows the documented download-and-install steps for a supported experimental release asset
- **THEN** they SHALL be able to install and run `vectos` without cloning the repository first

## ADDED Requirements

### Requirement: Vectos SHALL publish scoped experimental GitHub release assets
Vectos SHALL publish downloadable release assets only for the explicitly validated platform targets in the current experimental release phase.

#### Scenario: Publish validated initial targets
- **WHEN** Vectos publishes an experimental/internal GitHub release in the first release phase
- **THEN** it SHALL publish assets for `darwin/arm64` and `linux/amd64`

#### Scenario: Publish release checksums
- **WHEN** Vectos publishes an experimental/internal GitHub release
- **THEN** it SHALL publish checksums for the generated release assets
