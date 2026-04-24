## ADDED Requirements

### Requirement: Vectos SHALL expose richer content categories
Vectos SHALL distinguish common project content into lightweight categories that help users interpret search and indexing results.

#### Scenario: Report content category
- **WHEN** a chunk is indexed from a supported file
- **THEN** Vectos SHALL store and expose an appropriate category such as `source`, `infra_config`, `scripts`, `docs`, or `dependency_metadata`

### Requirement: Vectos SHALL preserve category context in search output
Vectos SHALL keep category context visible in returned results so users can understand whether a match comes from code, config, docs, or scripts.

#### Scenario: Search mixed project content
- **WHEN** search returns results from mixed file types
- **THEN** Vectos SHALL preserve enough metadata for the caller to distinguish those result categories
