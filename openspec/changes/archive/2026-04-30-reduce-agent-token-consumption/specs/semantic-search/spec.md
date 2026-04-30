## MODIFIED Requirements

### Requirement: Top search results SHALL minimize redundant candidates
The system SHALL reduce redundant top-ranked results when multiple candidates represent the same or nearly identical code locations.

#### Scenario: Neighboring chunks compete for top results
- **WHEN** multiple highly similar candidates point to the same file region or overlapping code unit
- **THEN** the system SHALL down-rank or collapse redundant candidates so the top results cover more distinct useful options

#### Scenario: Overlapping chunks from the same file are merged for output
- **WHEN** two or more ranked chunks from the same file have overlapping or touching line ranges within a configurable window (default 5 lines)
- **THEN** the system SHALL merge them into a single file-level output entry with combined line ranges, collected signatures, and the maximum relevance score

### Requirement: Search ranking SHALL prefer actionable code entry points when possible
The system SHALL prefer more actionable code entry points when ranking otherwise similar candidates.

#### Scenario: Two candidates are similarly relevant but one is more actionable
- **WHEN** two candidates are similarly relevant to the query but one includes stronger file, symbol, or chunk-role evidence
- **THEN** the system SHALL rank the more actionable candidate higher

#### Scenario: Signature presence increases actionability
- **WHEN** a candidate contains a stored function signature that matches query tokens semantically or textually
- **THEN** the system SHALL treat this as stronger actionability evidence compared to a candidate with equivalent semantic score but no signature

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
