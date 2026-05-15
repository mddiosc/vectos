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
- **THEN** it SHALL prefer meaningful structural chunk boundaries such as components, hooks, exported functions, classes, or test blocks when those boundaries can be derived safely

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
- **THEN** it SHALL store the extracted structural signature (if available) and inferred purpose in dedicated columns alongside the content and embedding

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
