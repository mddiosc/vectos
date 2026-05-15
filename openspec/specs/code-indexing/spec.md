## Purpose
Define how Vectos discovers indexable project files and persists searchable code chunks.
## Requirements
### Requirement: Project code can be indexed from files and directories
The system SHALL accept a file path or project directory and index supported source files into searchable code chunks.

#### Scenario: Index a single supported file
- **WHEN** a user or agent requests indexing for a supported source file
- **THEN** the system SHALL read the file, create code chunks, generate embeddings for those chunks, and persist them in the project index

#### Scenario: Index a directory recursively
- **WHEN** a user or agent requests indexing for a directory
- **THEN** the system SHALL recursively locate supported source files, skip ignored directories, and index each discovered file

#### Scenario: Reindex a previously indexed file
- **WHEN** a file that already exists in the index is indexed again
- **THEN** the system SHALL delete the prior chunks for that file before saving the new chunks

#### Scenario: Index a TypeScript or React file with structural boundaries
- **WHEN** the system indexes a supported TypeScript, TSX, JSX, or JavaScript frontend file
- **THEN** it SHALL prefer meaningful structural chunk boundaries such as components, hooks, exported functions, classes, interfaces, type aliases, enums, or test blocks when those boundaries can be derived safely

#### Scenario: Fall back safely for unsupported frontend structure
- **WHEN** the system cannot derive safe structural boundaries for a supported TypeScript or React file
- **THEN** it SHALL fall back to its generic chunking strategy instead of failing the indexing operation

#### Scenario: Refresh changed files incrementally
- **WHEN** the system is asked to refresh a subset of changed files for an indexed project
- **THEN** it SHALL delete prior chunks for those files and persist newly generated chunks for the current file contents

#### Scenario: Remove stale chunks for deleted or excluded files
- **WHEN** a previously indexed file is deleted or no longer matches the active indexing policy
- **THEN** the system SHALL remove that file's chunks from the project index

#### Scenario: Chunks are stored with structural metadata
- **WHEN** the system persists a chunk to the database
- **THEN** it SHALL store the extracted structural signature (if available) and inferred purpose in dedicated columns alongside the content and embedding. The purpose SHALL include TypeScript-specific tags (type definition, enumeration, async function) when applicable.

### Requirement: Go code SHALL be chunked by function boundaries when possible
The system SHALL chunk Go source files by function boundaries instead of only fixed line windows whenever a function-oriented split can be derived.

#### Scenario: Chunk a Go file with multiple functions
- **WHEN** the system indexes a Go file containing multiple top-level functions
- **THEN** it SHALL create separate chunks for each function and preserve line ranges for each chunk

#### Scenario: Chunk a Go file prelude
- **WHEN** a Go file contains package or import declarations before any function
- **THEN** the system SHALL preserve that prelude as a separate chunk

#### Scenario: Go function chunks are stored with signatures
- **WHEN** a Go function chunk is created
- **THEN** the system SHALL extract and store the function signature (the `func ...` declaration line) in the `signature` column

### Requirement: File walking SHALL reject known sensitive file patterns
During directory walking for indexable file collection, the system SHALL reject files matching known sensitive-file patterns. These files SHALL be skipped and recorded in the skipped-files list.

#### Scenario: .env file is skipped
- **WHEN** the indexer walks a directory containing a file named `.env`
- **THEN** the file SHALL be skipped and added to the skipped-files list with reason "sensitive file"

#### Scenario: .env variant files are skipped
- **WHEN** the indexer encounters files named `.env.local`, `.env.production`, or `.env.development`
- **THEN** those files SHALL be skipped and recorded as sensitive

#### Scenario: SSH private key files are skipped
- **WHEN** the indexer encounters files named `id_rsa`, `id_ecdsa`, `id_ed25519`, or filenames ending with `_rsa`, `_ecdsa`, `_ed25519`
- **THEN** those files SHALL be skipped and recorded as sensitive

#### Scenario: Certificate and credential files are skipped
- **WHEN** the indexer encounters files with extensions `.pem`, `.key`, `.pfx`, `.p12`, or named `credentials.json` or `service-account.json`
- **THEN** those files SHALL be skipped and recorded as sensitive

#### Scenario: Normal source files are still indexed
- **WHEN** the indexer encounters files with standard source extensions (`.go`, `.ts`, `.py`, etc.) that do not match sensitive patterns
- **THEN** those files SHALL be indexed normally without interference from the sensitive-file filter

#### Scenario: .env.example and similar non-sensitive variants are indexed
- **WHEN** the indexer encounters a file named `.env.example` or `.env.sample`
- **THEN** the file SHALL be indexed normally (the filter SHALL use exact-name matching, not prefix matching)



<!-- Added by native-watch-mode -->

## MODIFIED Requirements

### Requirement: Project code can be indexed from files and directories
The system SHALL accept a file path or project directory and index supported source files into searchable code chunks. When a file is indexed, the system SHALL compute and store a SHA256 hash of the file's content for change detection.

#### Scenario: Index a single supported file
- **WHEN** a user or agent requests indexing for a supported source file
- **THEN** the system SHALL read the file, create code chunks, generate embeddings for those chunks, compute and store the file's SHA256 hash, and persist the chunks and hash in the project index

#### Scenario: Index a directory recursively
- **WHEN** a user or agent requests indexing for a directory
- **THEN** the system SHALL recursively locate supported source files, skip ignored directories, and index each discovered file

#### Scenario: Reindex a previously indexed file
- **WHEN** a file that already exists in the index is indexed again
- **THEN** the system SHALL delete the prior chunks for that file, compute the new SHA256 hash, store the updated hash, and save the new chunks

#### Scenario: Index a TypeScript or React file with structural boundaries
- **WHEN** the system indexes a supported TypeScript, TSX, JSX, or JavaScript frontend file
- **THEN** it SHALL prefer meaningful structural chunk boundaries such as components, hooks, exported functions, classes, interfaces, type aliases, enums, or test blocks when those boundaries can be derived safely

#### Scenario: Fall back safely for unsupported frontend structure
- **WHEN** the system cannot derive safe structural boundaries for a supported TypeScript or React file
- **THEN** it SHALL fall back to its generic chunking strategy instead of failing the indexing operation

#### Scenario: Refresh changed files incrementally
- **WHEN** the system is asked to refresh a subset of changed files for an indexed project
- **THEN** it SHALL delete prior chunks for those files, recompute file hashes, and persist newly generated chunks for the current file contents

#### Scenario: Remove stale chunks for deleted or excluded files
- **WHEN** a previously indexed file is deleted from the filesystem or no longer matches the active indexing policy
- **THEN** the system SHALL remove that file's chunks and its hash entry from the project index

#### Scenario: Chunks are stored with structural metadata
- **WHEN** the system persists a chunk to the database
- **THEN** it SHALL store the extracted structural signature (if available) and inferred purpose in dedicated columns alongside the content and embedding. The purpose SHALL include TypeScript-specific tags (type definition, enumeration, async function) when applicable.

#### Scenario: Index with different embedding model is detected as stale
- **WHEN** a project was previously indexed with one embedding model (e.g., bge-small) and the current configuration uses a different model (e.g., jina-embeddings-v3) or different dimensions
- **THEN** the system SHALL detect the provider/model/dimension mismatch and flag the index as requiring reindex for accurate results

### Requirement: Indexing SHALL support user-configurable exclusion patterns
The system SHALL read exclusion patterns from `vectos.config.json` (project root) and the `index` section of `~/.vectos/config.json` (global). Patterns SHALL be applied in addition to hardcoded exclusions. Patterns from project config SHALL be appended to global config patterns.

#### Scenario: Project config excludes blog directory from docs indexing
- **WHEN** `vectos.config.json` specifies `index.docs.exclude: ["src/content/blog/**"]`
- **THEN** the documentation indexer SHALL skip files matching that pattern

#### Scenario: Global config provides organization-wide defaults
- **WHEN** `~/.vectos/config.json` specifies `index.code.exclude: ["**/generated/**"]`
- **THEN** all projects on that machine SHALL skip `generated/` directories during code indexing

#### Scenario: Both configs are applied cumulatively
- **WHEN** global config excludes `[".github/**"]` and project config excludes `["src/content/**"]`
- **THEN** files matching either pattern SHALL be skipped

### Requirement: Indexing SHALL automatically respect .gitignore
The system SHALL read `.gitignore` from the project root at indexing time and automatically skip files matching its patterns.

#### Scenario: Gitignored build output is excluded
- **WHEN** `.gitignore` contains `dist/` and the indexer walks the project
- **THEN** files under `dist/` SHALL be skipped

## ADDED Requirements

### Requirement: File hash storage tracks indexed file content
The system SHALL maintain an `indexed_files` table in SQLite with columns `path TEXT PRIMARY KEY`, `hash TEXT NOT NULL`, and `indexed_at TIMESTAMP NOT NULL`. The table SHALL store one row per indexed file with its SHA256 content hash and last-indexed timestamp.

#### Scenario: File is indexed and hash is stored
- **WHEN** a file is successfully indexed
- **THEN** a row SHALL be inserted or upserted into `indexed_files` with the file path, its SHA256 hash, and the current timestamp

#### Scenario: File is reindexed and hash is updated
- **WHEN** a previously indexed file is reindexed
- **THEN** the existing `indexed_files` row SHALL be updated with the new hash and timestamp

#### Scenario: File is deleted and hash is removed
- **WHEN** a previously indexed file is deleted from the filesystem
- **THEN** the corresponding row in `indexed_files` SHALL be deleted along with its chunks

#### Scenario: File hash is queried for change detection
- **WHEN** the system needs to determine if a file's content has changed since last index
- **THEN** it SHALL query the `hash` column from `indexed_files` and compare to the current file's computed SHA256 hash

### Requirement: Incremental reindex can skip unchanged files by hash
The system SHALL support skipping files during incremental reindex when their content hash has not changed since the last index operation.

#### Scenario: Changed file list includes file with unchanged hash
- **WHEN** an incremental reindex is triggered with a list of changed paths that includes a file whose content hash matches the stored hash
- **THEN** the system SHALL skip that file and not regenerate its chunks or embeddings

#### Scenario: Changed file list includes file with changed hash
- **WHEN** an incremental reindex is triggered with a list of changed paths that includes a file whose content hash differs from the stored hash
- **THEN** the system SHALL reindex that file, regenerate its chunks and embeddings, and update the stored hash

#### Scenario: Changed file list includes new file with no stored hash
- **WHEN** an incremental reindex is triggered for a file that has no entry in `indexed_files`
- **THEN** the system SHALL index the file as new and store its hash
