## Purpose
Define how Vectos fuses independent vector and keyword search results using Reciprocal Rank Fusion to produce a unified ranking that leverages both semantic meaning and syntactic text matching.

## Requirements

### Requirement: Search SHALL fuse vector and keyword results using Reciprocal Rank Fusion
The system SHALL execute both a semantic vector search and a term-frequency keyword search independently for each query. The two result sets SHALL be fused into a single ranking using Reciprocal Rank Fusion (RRF) with k=60. This fusion SHALL produce a final ranking that leverages both semantic relevance and syntactic keyword matching.

#### Scenario: Vector and keyword searches agree on top result
- **WHEN** a query produces a chunk that ranks highly in both the vector and keyword result sets
- **THEN** that chunk SHALL receive a high RRF score and appear at or near the top of the fused ranking

#### Scenario: Keyword search finds exact symbol match missed by vector search
- **WHEN** a query contains an exact symbol name (e.g., `mergeProjectData`) that the keyword search finds but the vector search ranks low
- **THEN** the keyword-matched chunk SHALL be elevated in the fused ranking by its keyword rank contribution to the RRF score

#### Scenario: Vector search finds semantic match missed by keyword search
- **WHEN** a query describes behavior in natural language (e.g., "form validation logic") that the vector search captures but keyword terms don't appear verbatim
- **THEN** the semantically matched chunk SHALL be elevated in the fused ranking by its vector rank contribution to the RRF score

#### Scenario: Chunk appears in only one result set
- **WHEN** a chunk appears in the vector search results but not in the keyword search results (or vice versa)
- **THEN** it SHALL receive an RRF score based solely on its rank in the contributing result set, without penalization for absence from the other set

### Requirement: Keyword search SHALL return ranked results with scores
The system SHALL execute keyword searches using term-frequency relevance scoring. Results SHALL be returned with their relevance scores for use in the fusion pipeline.

#### Scenario: Keyword search returns scored results
- **WHEN** the system executes a keyword search for query `custom React hooks`
- **THEN** each result SHALL include a relevance score reflecting term frequency in the chunk content and filename

#### Scenario: Keyword search returns empty for no matches
- **WHEN** a keyword search finds no chunks containing any query terms
- **THEN** the system SHALL return an empty result set without error, and the fusion SHALL rely entirely on vector search results

### Requirement: Vector and keyword searches SHALL execute with independent candidate pools
The system SHALL execute vector search and keyword search as independent operations, each returning its own top-ranked candidates. Neither search SHALL constrain or filter the other's results before fusion.

#### Scenario: Both searches return the maximum candidate limit
- **WHEN** both vector and keyword searches find many relevant candidates
- **THEN** each SHALL return up to its configured candidate limit (default 25) independently, and the fused ranking SHALL be computed from the union of both result sets

#### Scenario: One search returns fewer candidates than the limit
- **WHEN** the vector search returns only 8 candidates but the keyword search returns 25
- **THEN** the fused ranking SHALL include contributions from all 33 unique candidates (after deduplication by chunk ID)

### Requirement: Post-fusion penalties SHALL be limited to structural signals
After RRF fusion, the system SHALL apply only structural penalties (test file, build artifact path, help text) to the fused ranking. Content-matching boosts (exact phrase, token overlap, file name) SHALL be removed since keyword search captures that signal natively.

#### Scenario: Test file is penalized in fused ranking
- **WHEN** a fused result comes from a test file (e.g., `*_test.go`, `*.test.ts`)
- **THEN** its score SHALL be reduced by the test file penalty factor before final ranking

#### Scenario: Build artifact is penalized
- **WHEN** a fused result comes from a build artifact path (e.g., `/dist/`, `/.next/`)
- **THEN** its score SHALL be reduced by the build artifact penalty factor

#### Scenario: No content-matching boosts are applied
- **WHEN** a fused result contains an exact phrase match or token overlap with the query
- **THEN** no additional boost SHALL be applied beyond what RRF already captures from the keyword search contribution
