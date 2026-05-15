## MODIFIED Requirements

### Requirement: Code search SHALL support semantic retrieval
The system SHALL support semantic code retrieval by executing both a vector similarity search and a BM25 full-text keyword search independently, then fusing the two result sets using Reciprocal Rank Fusion (RRF) to produce a unified ranking. The unified ranking SHALL leverage both semantic meaning and syntactic keyword matching.

#### Scenario: Semantic query matches indexed code
- **WHEN** a query describes code behavior without using the exact symbol name
- **THEN** the system SHALL return chunks ranked by fused RRF scores that incorporate both semantic similarity and keyword relevance

#### Scenario: Hybrid signals improve ranked results
- **WHEN** text overlap or keyword matches provide useful extra evidence for a query beyond semantic similarity
- **THEN** the system SHALL use those signals via the keyword search contribution to RRF, rather than via post-retrieval boosting heuristics

#### Scenario: Semantic search returns top ranked results
- **WHEN** multiple indexed chunks are semantically related to the query
- **THEN** the system SHALL sort results by RRF fusion score in descending order, combining vector rank and keyword rank contributions

### Requirement: Semantic search SHALL fall back to text search
The system SHALL fall back to text-based search when semantic search cannot be executed or produces no useful results. When the keyword search component also returns no results, the system SHALL return results from whichever search component produced results (vector-only or keyword-only).

#### Scenario: Embedding provider unavailable
- **WHEN** the system cannot generate an embedding for the query
- **THEN** it SHALL run a keyword text search over indexed chunks instead of failing the request

#### Scenario: Semantic search returns no matches but keyword search succeeds
- **WHEN** semantic retrieval yields no results but the keyword search finds matches
- **THEN** the system SHALL return the keyword search results as the final output

#### Scenario: Both searches return no matches
- **WHEN** neither semantic retrieval nor keyword search yields any results
- **THEN** the system SHALL return an empty result set without error

### Requirement: Hybrid ranking surface SHALL be simplified
The system SHALL remove heuristic boosts and penalties that adjust ranking by less than 0.06 and rarely affect the final top-5 ordering. The RRF fusion mechanism SHALL replace content-matching boosts (exact phrase, token overlap, file name). Only structural penalties (test file, build artifact, help text) SHALL remain as post-fusion adjustments.

#### Scenario: Ranking uses RRF fusion instead of heuristic boosts
- **WHEN** the system computes a unified ranking
- **THEN** it SHALL use Reciprocal Rank Fusion with k=60 to combine vector and keyword rankings, and SHALL not apply per-intent content-matching boosts

#### Scenario: Structural penalties are preserved
- **WHEN** the system applies post-fusion adjustments
- **THEN** it SHALL apply test file, build artifact, and help text penalties to the fused ranking

### Requirement: Semantic search candidate pool SHALL be limited to high-confidence results
The system SHALL retrieve up to 25 candidates from each search component (vector and keyword) before fusion, producing a fused ranking of at most 10 final results.

#### Scenario: Query produces many candidates from both sources
- **WHEN** both vector and keyword searches yield a large number of candidates
- **THEN** the system SHALL retrieve up to 25 from each source, fuse them via RRF, and return at most 10 top-ranked results
