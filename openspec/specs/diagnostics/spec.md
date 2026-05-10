## ADDED Requirements

### Requirement: Vectos SHALL provide a local diagnostic command
Vectos SHALL provide a `doctor` command that performs local, read-only diagnostics for installation, runtime readiness, provider health, and index consistency.

#### Scenario: User runs the doctor command
- **WHEN** the user runs `vectos doctor`
- **THEN** Vectos SHALL print a sectioned diagnostic report with install, environment, provider, and index checks

#### Scenario: Doctor detects an unhealthy provider or model
- **WHEN** the embedding provider or model is unavailable, misconfigured, or not ready
- **THEN** `vectos doctor` SHALL mark that check as failing and include a short remediation hint

#### Scenario: Doctor detects index/provider mismatch
- **WHEN** the active index metadata does not match the current embedding provider configuration
- **THEN** `vectos doctor` SHALL report that a reindex is required

### Requirement: Vectos doctor SHALL not mutate user state
The diagnostic command SHALL not change files, indexes, environment variables, or installed binaries.

#### Scenario: User runs doctor repeatedly
- **WHEN** the user runs `vectos doctor` multiple times
- **THEN** each run SHALL produce diagnostics only and leave the workspace unchanged

### Requirement: Vectos doctor SHALL report success and failure clearly
The diagnostic command SHALL use exit codes to communicate overall health.

#### Scenario: All critical checks pass
- **WHEN** every critical diagnostic check succeeds
- **THEN** `vectos doctor` SHALL exit with code 0

#### Scenario: Any critical check fails
- **WHEN** one or more critical diagnostic checks fail
- **THEN** `vectos doctor` SHALL exit with a non-zero code
