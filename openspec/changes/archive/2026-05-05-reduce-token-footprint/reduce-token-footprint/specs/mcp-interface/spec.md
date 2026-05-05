## MODIFIED Requirements

### Requirement: MCP search failures SHALL suggest the next useful action
The system SHALL provide explicit recovery guidance when MCP search cannot return useful results because the project is missing an index or requires refresh.

#### Scenario: Project is not indexed
- **WHEN** an MCP client calls `search_code` for a project scope that has no usable index
- **THEN** the system SHALL set the `guidance` field to `IDX_MISSING` and the `next_action` field to the command required to index the project

#### Scenario: Project index is stale or incomplete
- **WHEN** an MCP client calls `search_code` and the system can determine that the available project index is stale or incomplete
- **THEN** the system SHALL set the `guidance` field to `IDX_STALE` and the `next_action` field to the command required to refresh the index

### Requirement: MCP search results SHALL include concise actionable metadata
The system SHALL return MCP search results with concise metadata that helps an agent decide whether a result is worth reading.

#### Scenario: Search result includes relevance context
- **WHEN** the system returns an MCP search result
- **THEN** each result SHALL include at least a relative file path, integer relevance score (0-100), line ranges, and function signatures — without redundant fields

#### Scenario: High-confidence results act as pointers without content preview
- **WHEN** a search result has a relevance score at or above the high-confidence threshold (default 0.90)
- **THEN** the system SHALL return only the relative file path, line ranges, and function signatures — without a truncated code preview or hint

#### Scenario: Low-confidence results include contextual hint
- **WHEN** a search result has a relevance score below the high-confidence threshold
- **THEN** the system SHALL include a concise contextual `hint` string that explains why the result was returned, in addition to path, line ranges, and signatures

### Requirement: MCP search results SHALL be returned at file-level granularity
The system SHALL group multiple chunks from the same file into a single MCP search result entry, consolidating metadata and eliminating duplication.

#### Scenario: Multiple chunks from same file in top results
- **WHEN** two or more top-ranked chunks originate from the same file and their line ranges overlap or touch within a configurable window (default 5 lines)
- **THEN** the system SHALL merge them into a single result entry with combined line ranges, collected signatures, and the highest relevance score among the merged chunks

#### Scenario: Result paths are relative to project root
- **WHEN** the system returns MCP search results
- **THEN** each result's `file_path` SHALL be relative to the project root, not an absolute path, reducing response payload size

#### Scenario: Distinct functions in the same file remain separate
- **WHEN** two top-ranked chunks originate from the same file but their line ranges are separated by more than the configured window
- **THEN** the system SHALL keep them as separate result entries

## REMOVED Requirements

### Requirement: MCP search result includes rank, file_name, language, and category fields
**Reason:** These fields are redundant or derivable by the client. `Rank` is inferable from array position. `FileName` is `filepath.Base(file_path)`. `Language` is derivable from file extension. `Category` is reconstructable via `classifyCategory(language)` on the client side.
**Migration:** Agent clients should derive these values from the remaining `file_path` and `language` fields rather than reading them directly from the response.