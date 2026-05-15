## MODIFIED Requirements

### Requirement: Project code can be indexed from files and directories
The system SHALL accept a file path or project directory and index supported source files into searchable code chunks. When a file is indexed, the system SHALL compute and store a SHA256 hash of the file's content for change detection. The system SHALL also record the embedding provider name, model name, and embedding dimensions used for the index in project metadata for stale-index detection.

#### Scenario: Index a single supported file
- **WHEN** a user or agent requests indexing for a supported source file
- **THEN** the system SHALL read the file, create code chunks, generate embeddings for those chunks using the configured embedding model, compute and store the file's SHA256 hash, record the embedding provider metadata (provider, model, dimensions), and persist the chunks and hash in the project index

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

#### Scenario: Index with different embedding model is detected as stale
- **WHEN** a project was previously indexed with one embedding model (e.g., bge-small) and the current configuration uses a different model (e.g., jina) or different dimensions
- **THEN** the system SHALL detect the provider/model/dimension mismatch and flag the index as requiring reindex for accurate results
