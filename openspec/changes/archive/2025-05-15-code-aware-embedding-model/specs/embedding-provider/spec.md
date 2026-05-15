## MODIFIED Requirements

### Requirement: Vectos SHALL support multiple embedding provider types
Vectos SHALL support at least one embedded local provider and one remote URL-based provider for embeddings. The embedded provider SHALL support multiple model options including a code-aware model (`jina-embeddings-v2-base-code`) and a general-purpose model (`bge-small-en-v1.5`), with the code-aware model as the default.

#### Scenario: Use embedded provider by default
- **WHEN** Vectos starts without an explicit provider override
- **THEN** it SHALL attempt to use the embedded provider first, defaulting to the code-aware model (`jina-embeddings-v2-base-code`)

### Requirement: Vectos SHALL support standalone local embeddings
Vectos SHALL be usable for indexing and semantic search without requiring the user to configure or operate an external embeddings provider. The default embedded model SHALL be `jina-embeddings-v2-base-code`, a code-aware model producing 768-dimensional embeddings.

#### Scenario: Index project with no external provider configured
- **WHEN** the user runs Vectos with default configuration and no remote embeddings endpoint configured
- **THEN** Vectos SHALL use its embedded local embeddings runtime with the code-aware model to generate vectors

#### Scenario: Search indexed project with embedded provider
- **WHEN** a project index was created with the embedded provider
- **THEN** Vectos SHALL support semantic search against that index without requiring a remote embeddings endpoint

#### Scenario: Use remote provider by configuration
- **WHEN** the user configures a remote provider endpoint
- **THEN** Vectos SHALL generate embeddings through that endpoint instead of the embedded provider

#### Scenario: User explicitly selects bge-small model
- **WHEN** the user sets `model_name: "bge-small-en-v1.5"` in the embedded configuration
- **THEN** Vectos SHALL use the bge-small model with 384-dimensional embeddings instead of the code-aware default
