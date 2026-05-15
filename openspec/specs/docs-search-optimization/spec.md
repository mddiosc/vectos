## Purpose
Define how documentation search is optimized for speed and relevance, supporting configurable index exclusions and path-based relevance boosts.

## Requirements

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

### Requirement: Users SHALL be able to configure index exclusions via project-level config
The system SHALL support a `vectos.config.json` file in the project root directory that allows users to specify glob-based exclusion patterns for documentation and code indexing. Patterns SHALL be additive to global defaults and hardcoded exclusions.

#### Scenario: Project config excludes blog content from docs
- **WHEN** `vectos.config.json` contains `{"index": {"docs": {"exclude": ["src/content/blog/**"]}}}`
- **THEN** files under `src/content/blog/` SHALL be skipped during documentation indexing

#### Scenario: Project config excludes mocks from code index
- **WHEN** `vectos.config.json` contains `{"index": {"code": {"exclude": ["**/__mocks__/**"]}}}`
- **THEN** files under any `__mocks__` directory SHALL be skipped during code indexing

#### Scenario: Project without config uses defaults
- **WHEN** no `vectos.config.json` exists in the project root
- **THEN** the system SHALL index normally using only hardcoded exclusions and global config defaults

### Requirement: Users SHALL be able to configure global index exclusion defaults
The global configuration file `~/.vectos/config.json` SHALL support an `index` section with `docs.exclude` and `code.exclude` arrays. These patterns SHALL apply to all projects unless a project config adds additional patterns.

#### Scenario: Global config excludes GitHub prompts from all docs indexes
- **WHEN** `~/.vectos/config.json` contains `{"index": {"docs": {"exclude": [".github/prompts/**"]}}}`
- **THEN** every project's documentation index SHALL skip files under `.github/prompts/`

#### Scenario: Global and project configs are merged cumulatively
- **WHEN** global config excludes `[".github/**"]` and project config excludes `["src/content/**"]`
- **THEN** both `.github/` and `src/content/` files SHALL be excluded from the index

### Requirement: Indexing SHALL automatically respect .gitignore patterns
The system SHALL read the project's `.gitignore` file at indexing time and automatically exclude files matching its patterns. This behavior SHALL be applied by default and SHALL NOT require configuration.

#### Scenario: Gitignored files are excluded from indexing
- **WHEN** `.gitignore` contains `*.log` and the indexer encounters `app.log`
- **THEN** `app.log` SHALL be skipped and recorded in the skipped-files list

#### Scenario: Gitignored directories are excluded from indexing
- **WHEN** `.gitignore` contains `dist/` and the indexer encounters `dist/bundle.js`
- **THEN** `dist/bundle.js` SHALL be skipped

#### Scenario: Negated gitignore patterns still exclude (initial implementation)
- **WHEN** `.gitignore` contains `!important.log` (negation)
- **THEN** the system SHALL skip the negation for now (future enhancement) and follow only positive exclusion patterns

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

### Requirement: All recent features SHALL be documented
The project documentation (README.md and docs/) SHALL be updated to reflect all features implemented since the last documentation update: jina-embeddings-v3 default model, RRF keyword-vector fusion, TypeScript structural tagging, configurable index exclusions, lockfile/config-file exclusions, and HNSW tuning parameters.

#### Scenario: README reflects current default model
- **WHEN** a new user reads the README
- **THEN** it SHALL mention jina-embeddings-v3 as the default embedding model with 1024 dimensions

#### Scenario: Indexing docs explain exclusion config
- **WHEN** a user reads `docs/indexing.md`
- **THEN** it SHALL document the `vectos.config.json` format, global config `index` section, and `.gitignore` integration
