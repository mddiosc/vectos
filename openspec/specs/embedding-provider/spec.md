## ADDED Requirements

### Requirement: Vectos SHALL support multiple embedding provider types
Vectos SHALL support at least one embedded local provider and one remote URL-based provider for embeddings.

#### Scenario: Use embedded provider by default
- **WHEN** Vectos starts without an explicit provider override
- **THEN** it SHALL attempt to use the embedded provider first

### Requirement: Vectos SHALL support standalone local embeddings
Vectos SHALL be usable for indexing and semantic search without requiring the user to configure or operate an external embeddings provider.

#### Scenario: Index project with no external provider configured
- **WHEN** the user runs Vectos with default configuration and no remote embeddings endpoint configured
- **THEN** Vectos SHALL use its embedded local embeddings runtime to generate vectors

#### Scenario: Search indexed project with embedded provider
- **WHEN** a project index was created with the embedded provider
- **THEN** Vectos SHALL support semantic search against that index without requiring a remote embeddings endpoint

#### Scenario: Use remote provider by configuration
- **WHEN** the user configures a remote provider endpoint
- **THEN** Vectos SHALL generate embeddings through that endpoint instead of the embedded provider

### Requirement: Remote providers SHALL use an OpenAI-compatible embeddings API
Vectos SHALL treat remote URL providers as OpenAI-compatible embeddings endpoints in the initial implementation.

#### Scenario: Connect to local or remote compatible endpoint
- **WHEN** the configured endpoint implements the OpenAI-compatible embeddings contract
- **THEN** Vectos SHALL use it as a valid remote provider
