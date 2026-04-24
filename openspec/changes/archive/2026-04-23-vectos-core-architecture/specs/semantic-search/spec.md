## ADDED Requirements

### Requirement: Code search SHALL support semantic retrieval
The system SHALL support semantic code retrieval by generating an embedding for the query and ranking indexed chunks by vector similarity.

#### Scenario: Semantic query matches indexed code
- **WHEN** a query describes code behavior without using the exact symbol name
- **THEN** the system SHALL return chunks ranked by semantic similarity to the query

#### Scenario: Semantic search returns top ranked results
- **WHEN** multiple indexed chunks are semantically related to the query
- **THEN** the system SHALL sort results by similarity score in descending order

### Requirement: Semantic search SHALL fall back to text search
The system SHALL fall back to text-based search when semantic search cannot be executed or produces no useful results.

#### Scenario: Embedding provider unavailable
- **WHEN** the system cannot generate an embedding for the query
- **THEN** it SHALL run a text search over indexed chunks instead of failing the request

#### Scenario: Semantic search returns no matches
- **WHEN** semantic retrieval yields no results
- **THEN** the system SHALL run a text search using the original query

### Requirement: Embedding input SHALL be semantically enriched
The system SHALL generate embeddings from enriched chunk content that includes structural metadata in addition to raw code.

#### Scenario: Enrich a Go function chunk
- **WHEN** the system creates an embedding for a Go function chunk
- **THEN** it SHALL include contextual metadata such as file name, language, function signature, and inferred purpose along with the raw code
