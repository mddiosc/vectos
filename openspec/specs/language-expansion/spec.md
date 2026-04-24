## ADDED Requirements

### Requirement: Vectos SHALL index more source and infrastructure languages
Vectos SHALL support indexing additional source and infrastructure/config file types beyond the current baseline.

#### Scenario: Index supported non-Go file types
- **WHEN** a project contains supported `.java`, `Dockerfile`, `docker-compose*.yml`, `*.yml`, `*.yaml`, `BUILD`, `BUILD.bazel`, `WORKSPACE`, `MODULE.bazel`, or `*.bzl` files
- **THEN** Vectos SHALL include those files in indexing for the project scope

### Requirement: Vectos SHALL classify file types appropriately
Vectos SHALL distinguish source code files from infrastructure/config files during indexing and reporting.

#### Scenario: Report indexed file types
- **WHEN** indexing or status output is generated
- **THEN** Vectos SHALL preserve enough metadata to identify the language or file category of indexed chunks
