## MODIFIED Requirements

### Requirement: MCP search results SHALL include concise actionable metadata
The system SHALL return MCP search results with concise metadata that helps an agent decide whether a result is worth reading.

#### Scenario: Search result includes relevance context
- **WHEN** the system returns an MCP search result
- **THEN** each result SHALL include at least the file path and enough concise metadata, such as rank, line range, chunk role, or short match context, to support agent decision-making

#### Scenario: High-confidence results act as pointers without content preview
- **WHEN** a search result has a relevance score at or above the high-confidence threshold (default 0.90)
- **THEN** the system SHALL return only the file path, line ranges, function signatures, and a relevance score — without a truncated code preview

#### Scenario: Low-confidence results include contextual hint
- **WHEN** a search result has a relevance score below the high-confidence threshold
- **THEN** the system SHALL include a concise contextual `hint` string that explains why the result was returned, in addition to path, line ranges, and signatures

### Requirement: MCP search results SHALL be returned at file-level granularity
The system SHALL group multiple chunks from the same file into a single MCP search result entry, consolidating metadata and eliminating duplication.

#### Scenario: Multiple chunks from same file in top results
- **WHEN** two or more top-ranked chunks originate from the same file and their line ranges overlap or touch within a configurable window (default 5 lines)
- **THEN** the system SHALL merge them into a single result entry with combined line ranges, collected signatures, and the highest relevance score among the merged chunks

#### Scenario: Distinct functions in the same file remain separate
- **WHEN** two top-ranked chunks originate from the same file but their line ranges are separated by more than the configured window
- **THEN** the system SHALL keep them as separate result entries

## ADDED Requirements

### Requirement: MCP search output format SHALL support file-level entries
The system SHALL use a per-file result structure for MCP search output that replaces the previous per-chunk structure.

#### Scenario: MCP client receives search results
- **WHEN** an MCP client calls `search_code`
- **THEN** the returned JSON SHALL contain a `results` array where each element represents a file with `file_path`, `file_name`, `language`, `category`, `relevance`, `line_ranges`, `signatures`, and an optional `hint`

#### Scenario: File-level entry contains consolidated signatures
- **WHEN** a file-level result entry is constructed from multiple source chunks
- **THEN** it SHALL include a `signatures` array containing the signature or first line of each distinct structural unit (e.g., function, class, exported symbol) found in the source chunks, deduplicated

### Requirement: Indexed chunks SHALL preserve function signatures and purpose
The system SHALL store function signatures and inferred purposes alongside chunk content so they are available in search output without re-parsing.

#### Scenario: Chunk is saved during indexing
- **WHEN** the system persists a code chunk to the database
- **THEN** it SHALL also store the extracted function signature (if any) in a `signature` column and the inferred purpose description (if any) in a `purpose` column

#### Scenario: Chunk without structural signature
- **WHEN** a chunk does not contain a recognizable structural signature
- **THEN** the `signature` column may be empty, and the system SHALL fall back to the inferred `purpose` or category for the output `hint`
