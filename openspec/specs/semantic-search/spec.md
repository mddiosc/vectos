## ADDED Requirements

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

#### Scenario: Signature presence increases actionability
- **WHEN** a candidate contains a stored function signature that matches query tokens semantically or textually
- **THEN** the system SHALL treat this as stronger actionability evidence compared to a candidate with equivalent semantic score but no signature

## MODIFIED Requirements

### Requirement: Top search results SHALL minimize redundant candidates
The system SHALL reduce redundant top-ranked results when multiple candidates represent the same or nearly identical code locations.

#### Scenario: Neighboring chunks compete for top results
- **WHEN** multiple highly similar candidates point to the same file region or overlapping code unit
- **THEN** the system SHALL down-rank or collapse redundant candidates so the top results cover more distinct useful options

#### Scenario: Overlapping chunks from the same file are merged for output
- **WHEN** two or more ranked chunks from the same file have overlapping or touching line ranges within a configurable window (default 5 lines)
- **THEN** the system SHALL merge them into a single file-level output entry with combined line ranges, collected signatures, and the maximum relevance score

## ADDED Requirements

### Requirement: Semantic search candidate pool SHALL be limited to high-confidence results
The system SHALL reduce the maximum number of hybrid candidates processed from 25 to 10, prioritizing a smaller set of higher-confidence results.

#### Scenario: Query produces many candidates
- **WHEN** a semantic search yields a large number of candidate chunks
- **THEN** the system SHALL process at most 10 candidates through hybrid ranking, and the final output SHALL reflect the top ranked results from this reduced pool

### Requirement: Hybrid ranking surface SHALL be simplified
The system SHALL remove heuristic boosts and penalties that adjust ranking by less than 0.06 and rarely affect the final top-5 ordering.

#### Scenario: Ranking runs with simplified heuristics
- **WHEN** the system computes hybrid scores
- **THEN** it SHALL use only high-impact signals (exact phrase match, token overlap ratio, file name overlap, actionable code detection, source category preference, and build artifact/test penalties), and SHALL not apply fine-grained per-intent boosts

## REMOVED Requirements

### Requirement: (Implicit fine-grained intent-specific boosts)
**Reason:** The 40+ individual intent boosts (config, database, SEO, form, state, auth, data, UI, routing, etc.) produce marginal ranking improvements at the cost of code complexity and negligible agent token savings.
**Migration:** The remaining high-impact signals (exact phrase, token overlap, file name overlap, actionable code detection, category preference, artifact penalties) cover the same cases effectively with less code.
