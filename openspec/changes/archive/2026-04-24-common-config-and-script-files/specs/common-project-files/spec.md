## ADDED Requirements

### Requirement: Vectos SHALL index common project files beyond source code
Vectos SHALL support indexing common script, config, metadata, and documentation files that are frequently used to understand project behavior.

#### Scenario: Index common project files
- **WHEN** a project contains supported `.json`, `.sh`, `.md`, `.toml`, `.ini`, `.xml`, `.properties`, `Makefile`, or `.gitignore` files
- **THEN** Vectos SHALL include those files in indexing for the project scope

### Requirement: Vectos SHALL avoid secret-prone env files in this phase
Vectos SHALL not include `.env`-style runtime secret files in this first implementation phase by default.

#### Scenario: Skip `.env`
- **WHEN** a repository contains `.env` files with runtime configuration
- **THEN** Vectos SHALL skip them by default in this phase
