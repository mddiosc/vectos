## Purpose
Define semantic retrieval, text-search fallback, and ranking behavior for code search.
## Requirements
### Requirement: Code search SHALL support semantic retrieval
The system SHALL support semantic code retrieval by executing both a vector similarity search and a term-frequency keyword search independently, then fusing the two result sets using Reciprocal Rank Fusion (RRF) to produce a unified ranking. The unified ranking SHALL leverage both semantic meaning and syntactic keyword matching.

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

### Requirement: Embedding input SHALL be semantically enriched
The system SHALL generate embeddings from enriched chunk content that includes structural metadata in addition to raw code. The enrichment SHALL include TypeScript-specific structural tags (type definition, enumeration, async function) when the chunk content contains those constructs.

#### Scenario: Enrich a Go function chunk
- **WHEN** the system creates an embedding for a Go function chunk
- **THEN** it SHALL include contextual metadata such as file name, language, function signature, and inferred purpose along with the raw code

#### Scenario: Enrich a TypeScript interface chunk
- **WHEN** the system creates an embedding for a TypeScript chunk containing an interface declaration
- **THEN** it SHALL include the "type definition" tag in the chunk purpose and include it in the embedding input text

#### Scenario: Enrich a TypeScript enum chunk
- **WHEN** the system creates an embedding for a TypeScript chunk containing an enum declaration
- **THEN** it SHALL include the "enumeration" tag in the chunk purpose and include it in the embedding input text

#### Scenario: Enrich a TypeScript or React structural chunk
- **WHEN** the system creates an embedding for a TypeScript or React chunk with a recognized structural role (component, hook, type definition, enumeration, async function, etc.)
- **THEN** it SHALL include contextual metadata such as file name, language, chunk role, and detected signature when available in addition to the raw code

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

### Requirement: Semantic search candidate pool SHALL be limited to high-confidence results
The system SHALL retrieve up to 25 candidates from each search component (vector and keyword) before fusion, producing a fused ranking of at most 10 final results.

#### Scenario: Query produces many candidates from both sources
- **WHEN** both vector and keyword searches yield a large number of candidates
- **THEN** the system SHALL retrieve up to 25 from each source, fuse them via RRF, and return at most 10 top-ranked results

### Requirement: Hybrid ranking surface SHALL be simplified
The system SHALL remove heuristic boosts and penalties that adjust ranking by less than 0.06 and rarely affect the final top-5 ordering. The RRF fusion mechanism SHALL replace content-matching boosts (exact phrase, token overlap, file name). Only structural penalties (test file, build artifact, help text) SHALL remain as post-fusion adjustments.

#### Scenario: Ranking uses RRF fusion instead of heuristic boosts
- **WHEN** the system computes a unified ranking
- **THEN** it SHALL use Reciprocal Rank Fusion with k=60 to combine vector and keyword rankings, and SHALL not apply per-intent content-matching boosts

#### Scenario: Structural penalties are preserved
- **WHEN** the system applies post-fusion adjustments
- **THEN** it SHALL apply test file, build artifact, and help text penalties to the fused ranking

### Requirement: Text search SHALL escape LIKE wildcards in user input
When performing a text-based fallback search using SQL LIKE, the system SHALL escape SQL LIKE wildcard metacharacters (`%` and `_`) in user-provided search terms to prevent unintended wildcard injection and resource-exhaustion attacks.

#### Scenario: Normal search term passes through
- **WHEN** a text search is performed with query `auth middleware`
- **THEN** the search SHALL construct a LIKE pattern `%auth middleware%` and return matching results normally

#### Scenario: Wildcard characters in query are escaped
- **WHEN** a text search is performed with query `100% coverage`
- **THEN** the `%` character SHALL be escaped to `\%` before constructing the LIKE clause, so the search matches literal `100%` text rather than treating `%` as a wildcard

#### Scenario: Underscore in query is escaped
- **WHEN** a text search is performed with query `my_func`
- **THEN** the `_` character SHALL be escaped to `\_` before constructing the LIKE clause, so the search matches literal `my_func` rather than treating `_` as a single-character wildcard

#### Scenario: Denial-of-service pattern is neutralized
- **WHEN** a text search is performed with a query consisting entirely of repeated `%` characters (e.g., `%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%`)
- **THEN** all `%` characters SHALL be escaped to `\%`, preventing the LIKE engine from expanding the pattern combinatorially

#### Scenario: Escape character itself is escaped
- **WHEN** a text search is performed with query containing a backslash (e.g., `path\to\file`)
- **THEN** the backslash character SHALL be escaped to `\\` before constructing the LIKE clause, preventing escape-chain bypass

