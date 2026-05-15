## ADDED Requirements

### Requirement: Vector search SHALL use an approximate nearest neighbor index
The system SHALL replace the O(n) full-table cosine scan in semantic search with a persisted approximate nearest neighbor (ANN) index that provides sub-linear lookup performance for top-K retrieval.

#### Scenario: Semantic search uses the index
- **WHEN** a semantic search query is executed with a valid query embedding
- **THEN** the system SHALL route the query through the vector index instead of scanning every chunk row for cosine similarity

#### Scenario: Fallback to linear scan when index is missing
- **WHEN** the vector index file does not exist or is stale relative to the chunk table
- **THEN** the system SHALL fall back to the existing full-table cosine scan and emit a warning

#### Scenario: Fallback to text search when both index and scan fail
- **WHEN** both the vector index lookup and the linear scan fail to produce results
- **THEN** the system SHALL fall back to text-based search as specified in the semantic-search spec

### Requirement: Vector index SHALL be persisted to disk
The system SHALL store the vector index in a binary file alongside the SQLite database, and SHALL load it on `vectos serve` startup without a full rebuild.

#### Scenario: Index survives serve restart
- **WHEN** `vectos serve` starts and a valid `.vectorindex` file exists alongside the SQLite database
- **THEN** the system SHALL load the index from disk and use it for all semantic queries without rebuilding

#### Scenario: Index is rebuilt on explicit reindex
- **WHEN** the user runs `vectos index` on an indexed project
- **THEN** the system SHALL rebuild the vector index in full and overwrite the existing `.vectorindex` file

#### Scenario: Stale index is detected
- **WHEN** the chunk table content has changed since the last index build
- **THEN** the system SHALL detect staleness via a content hash stored in the index header and treat the index as missing

### Requirement: Vector index SHALL support configurable accuracy
The system SHALL expose an `ef_search` parameter that controls the accuracy/speed trade-off during ANN search, with a default that provides >95% recall for 384-dimensional vectors.

#### Scenario: Higher ef_search improves recall
- **WHEN** `ef_search` is set to a higher value (e.g., 200)
- **THEN** the system SHALL search more nodes per query, improving recall at the cost of slower search

#### Scenario: Default ef_search is sufficient
- **WHEN** `ef_search` is not explicitly configured
- **THEN** the system SHALL use a default value of 100, providing >95% recall for typical 384-dim code embeddings

### Requirement: Vector index SHALL use cosine distance for graph edges
The HNSW graph SHALL use cosine distance (1 - cosine similarity) as its distance metric to match the existing similarity scoring.

#### Scenario: Nearest neighbors respect cosine distance
- **WHEN** the HNSW graph is constructed
- **THEN** edges SHALL connect nodes with the smallest cosine distance between their vectors

#### Scenario: Search uses cosine distance
- **WHEN** an ANN query is performed
- **THEN** the system SHALL rank candidates by cosine distance, returning the nodes with the smallest distances first
