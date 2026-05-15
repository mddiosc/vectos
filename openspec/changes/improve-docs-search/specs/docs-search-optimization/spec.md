## ADDED Requirements

### Requirement: Documentation search SHALL use the HNSW vector index when available
The system SHALL load the HNSW vector index for documentation search stores and use it for approximate nearest neighbor search when the chunk count exceeds 1000. For smaller documentation indexes, the system SHALL use linear scan for maximum accuracy.

#### Scenario: Docs search uses HNSW for large indexes
- **WHEN** a documentation index contains more than 1000 chunks and an HNSW index file exists
- **THEN** the system SHALL load the index and use HNSW approximate search for vector retrieval

#### Scenario: Docs search uses linear scan for small indexes
- **WHEN** a documentation index contains 1000 or fewer chunks
- **THEN** the system SHALL use linear scan for maximum accuracy

#### Scenario: HNSW index missing falls back to linear scan
- **WHEN** the HNSW index file does not exist or fails to load
- **THEN** the system SHALL log a warning and fall back to linear scan without failing the search request

### Requirement: Documentation search SHALL load HNSW on the same initialization path as code search
The documentation search initialization path in `commands_search.go` SHALL call `store.LoadVectorIndex()` when opening a documentation store, matching the behavior already present in the code search path.

#### Scenario: Docs search loads vector index on init
- **WHEN** a documentation search is executed via `vectos search --docs`
- **THEN** the system SHALL attempt to load the HNSW vector index before executing the search query

### Requirement: Documentation indexing SHALL exclude non-documentation content
The documentation indexing scope SHALL exclude developer tool prompts (`.github/prompts/`) and content directories (`.agents/skills/`) that contain markdown files unrelated to project documentation. Blog content directories SHALL remain indexable but receive lower relevance scores.

#### Scenario: GitHub prompts are excluded from docs index
- **WHEN** the documentation indexer walks the project directory
- **THEN** files under `.github/prompts/` SHALL be skipped and recorded in the skipped-files list

#### Scenario: Agent skill files are excluded from docs index
- **WHEN** the documentation indexer walks the project directory
- **THEN** files under `.agents/skills/` SHALL be skipped

#### Scenario: Blog content remains indexable but lower priority
- **WHEN** a documentation search query is executed
- **THEN** blog content files SHALL appear in results but rank lower than files under the `docs/` directory for the same keyword matches

### Requirement: Documentation files SHALL receive path-based relevance boosts
The keyword scoring function SHALL apply a 1.5× boost to chunks whose file path starts with `docs/` or whose filename is `README.md`, ensuring project documentation ranks above blog posts and other non-documentation markdown content.

#### Scenario: Docs directory file gets relevance boost
- **WHEN** the keyword scorer evaluates a chunk from `docs/en/DEVELOPMENT.md`
- **THEN** its score SHALL be multiplied by 1.5 before ranking

#### Scenario: Project README gets relevance boost
- **WHEN** the keyword scorer evaluates a chunk from `README.md` at the project root
- **THEN** its score SHALL be multiplied by 1.5

#### Scenario: Blog post does NOT get the docs boost
- **WHEN** the keyword scorer evaluates a chunk from `src/content/blog/en/some-post.md`
- **THEN** its score SHALL NOT receive the 1.5× docs directory boost
