## ADDED Requirements

### Requirement: Documentation indexing SHALL exclude non-documentation content
When indexing with `docsOnly=true`, the system SHALL skip developer tool prompts (`.github/prompts/`) and agent skill files (`.agents/skills/`) that contain markdown content unrelated to project documentation.

#### Scenario: GitHub prompts directory skipped in docs mode
- **WHEN** the documentation indexer walks a project containing `.github/prompts/` files
- **THEN** those files SHALL be skipped and recorded in the skipped-files list

#### Scenario: Agent skill directory skipped in docs mode
- **WHEN** the documentation indexer walks a project containing `.agents/skills/` files
- **THEN** those files SHALL be skipped

#### Scenario: Blog content remains indexable in docs mode
- **WHEN** the documentation indexer walks a project containing `src/content/blog/` files
- **THEN** those files SHALL be indexed as documentation (but rank lower via keyword scoring boosts for docs/)
