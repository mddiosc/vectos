## ADDED Requirements

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

### Requirement: Go code SHALL be chunked by function boundaries when possible
The system SHALL chunk Go source files by function boundaries instead of only fixed line windows whenever a function-oriented split can be derived.

#### Scenario: Chunk a Go file with multiple functions
- **WHEN** the system indexes a Go file containing multiple top-level functions
- **THEN** it SHALL create separate chunks for each function and preserve line ranges for each chunk

#### Scenario: Chunk a Go file prelude
- **WHEN** a Go file contains package or import declarations before any function
- **THEN** the system SHALL preserve that prelude as a separate chunk
