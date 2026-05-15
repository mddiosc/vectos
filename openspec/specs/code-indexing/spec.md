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
- **THEN** it SHALL prefer meaningful structural chunk boundaries such as components, hooks, exported functions, classes, or test blocks when those boundaries can be derived safely

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
- **THEN** it SHALL store the extracted structural signature (if available) and inferred purpose in dedicated columns alongside the content and embedding

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

