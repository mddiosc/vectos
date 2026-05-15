## ADDED Requirements

### Requirement: Documentation indexing SHALL exclude non-documentation content
When indexing with `docsOnly=true`, the system SHALL skip developer tool prompts (`.github/prompts/`) and agent skill files (`.agents/skills/`) that contain markdown content unrelated to project documentation. These exclusions SHALL be hardcoded defaults that apply even without user configuration.

#### Scenario: GitHub prompts directory skipped in docs mode
- **WHEN** the documentation indexer walks a project containing `.github/prompts/` files
- **THEN** those files SHALL be skipped and recorded in the skipped-files list

#### Scenario: Agent skill directory skipped in docs mode
- **WHEN** the documentation indexer walks a project containing `.agents/skills/` files
- **THEN** those files SHALL be skipped

### Requirement: Indexing SHALL support user-configurable exclusion patterns
The system SHALL read exclusion patterns from `vectos.config.json` (project root) and the `index` section of `~/.vectos/config.json` (global). These patterns SHALL be applied in addition to hardcoded exclusions. Patterns from project config SHALL be appended to global config patterns; neither replaces the other.

#### Scenario: Project config excludes blog directory from docs indexing
- **WHEN** `vectos.config.json` specifies `index.docs.exclude: ["src/content/blog/**"]`
- **THEN** the documentation indexer SHALL skip files matching that pattern

#### Scenario: Global config provides organization-wide defaults
- **WHEN** `~/.vectos/config.json` specifies `index.code.exclude: ["**/generated/**"]`
- **THEN** all projects on that machine SHALL skip `generated/` directories during code indexing

#### Scenario: Both configs are applied cumulatively
- **WHEN** global config excludes `[".github/**"]` and project config excludes `["src/content/**"]`
- **THEN** files matching either pattern SHALL be skipped

### Requirement: Indexing SHALL automatically respect .gitignore
The system SHALL read `.gitignore` from the project root at indexing time and automatically skip files matching its patterns. This SHALL happen regardless of whether a `vectos.config.json` exists.

#### Scenario: Gitignored build output is excluded
- **WHEN** `.gitignore` contains `dist/` and the indexer walks the project
- **THEN** files under `dist/` SHALL be skipped

#### Scenario: Gitignore with wildcard patterns works
- **WHEN** `.gitignore` contains `*.log` and `logs/*.log`
- **THEN** all `.log` files in any directory and `.log` files in `logs/` SHALL be skipped
