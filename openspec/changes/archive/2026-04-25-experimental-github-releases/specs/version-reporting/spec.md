## ADDED Requirements

### Requirement: Vectos SHALL expose build version metadata consistently
Vectos SHALL expose release version metadata from a single build-time source so users and clients can identify the exact build they are running.

#### Scenario: Report version from CLI
- **WHEN** a user runs the version-reporting command for Vectos
- **THEN** Vectos SHALL report at least the release version, commit identifier, and build date for the running binary

#### Scenario: Report version from MCP metadata
- **WHEN** an MCP client initializes against a Vectos server
- **THEN** the server metadata SHALL report the same release version that was injected into the build
