## ADDED Requirements

### Requirement: Provider selection SHALL be configuration-driven
Vectos SHALL select embedding providers from explicit configuration rather than hardcoded values.

#### Scenario: Configure provider priority
- **WHEN** the user defines provider priority or fallback order
- **THEN** Vectos SHALL honor that order when choosing an embedding provider

#### Scenario: Default to embedded standalone mode
- **WHEN** the user does not explicitly configure a provider
- **THEN** Vectos SHALL choose the embedded local provider by default

### Requirement: Provider changes SHALL surface reindexing requirements
Vectos SHALL warn when changing providers may invalidate existing embeddings for a project index.

#### Scenario: Switch provider with different embedding space
- **WHEN** the user changes the embedding provider or model for an indexed project
- **THEN** Vectos SHALL indicate that the index must be rebuilt before semantic search is considered valid
