## MODIFIED Requirements

### Requirement: Code search SHALL support semantic retrieval
The system SHALL support semantic code retrieval by generating an embedding for the query and ranking indexed chunks by vector similarity, optionally strengthened by additional hybrid ranking signals that improve top-result quality.

#### Scenario: Semantic query matches indexed code
- **WHEN** a query describes code behavior without using the exact symbol name
- **THEN** the system SHALL return chunks ranked by semantic similarity to the query

#### Scenario: Hybrid signals improve ranked results
- **WHEN** text overlap, symbol relevance, or file-level signals provide useful extra evidence for a query
- **THEN** the system SHALL be able to use those signals to improve the ranking order of the semantic candidate set

#### Scenario: Semantic search returns top ranked results
- **WHEN** multiple indexed chunks are semantically related to the query
- **THEN** the system SHALL sort results by overall ranking quality in descending order using semantic similarity as the base signal

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

#### Scenario: Enrich a TypeScript or React structural chunk
- **WHEN** the system creates an embedding for a TypeScript or React chunk with a recognized structural role
- **THEN** it SHALL include contextual metadata such as file name, language, chunk role, and detected signature when available in addition to the raw code

## ADDED Requirements

### Requirement: Top search results SHALL minimize redundant candidates
The system SHALL reduce redundant top-ranked results when multiple candidates represent the same or nearly identical code locations.

#### Scenario: Neighboring chunks compete for top results
- **WHEN** multiple highly similar candidates point to the same file region or overlapping code unit
- **THEN** the system SHALL down-rank or collapse redundant candidates so the top results cover more distinct useful options

### Requirement: Search ranking SHALL prefer actionable code entry points when possible
The system SHALL prefer more actionable code entry points when ranking otherwise similar candidates.

#### Scenario: Two candidates are similarly relevant but one is more actionable
- **WHEN** two candidates are similarly relevant to the query but one includes stronger file, symbol, or chunk-role evidence
- **THEN** the system SHALL rank the more actionable candidate higher
