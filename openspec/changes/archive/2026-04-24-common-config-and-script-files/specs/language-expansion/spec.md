## ADDED Requirements

### Requirement: Vectos SHALL expand supported non-source project file types
Vectos SHALL support additional common non-source project file types beyond the current source and infra/config baseline.

#### Scenario: Index common script and config files
- **WHEN** a project contains supported `.json`, `.sh`, `.md`, `.toml`, `.ini`, `.xml`, `.properties`, `Makefile`, or `.gitignore` files
- **THEN** Vectos SHALL include those files in indexing and search for the project scope
