## ADDED Requirements

### Requirement: The MCP server SHALL expose a project listing tool
The system SHALL expose `list_projects` as an MCP tool so agents can discover available Nx project names before indexing or searching.

#### Scenario: MCP client lists available tools includes list_projects
- **WHEN** an MCP client requests the available tools
- **THEN** the server SHALL return at least `search_code`, `search_docs`, `index_project`, and `list_projects`

#### Scenario: MCP client calls list_projects in an Nx workspace
- **WHEN** an MCP client calls `list_projects` inside an Nx workspace
- **THEN** the server SHALL return a JSON object with an array of sorted project names

## MODIFIED Requirements

### Requirement: MCP search results SHALL include concise actionable metadata
The system SHALL return MCP search results with concise metadata that helps an agent decide whether a result is worth reading. When the search scope is resolved from an Nx workspace project, file paths SHALL be relative to the project's primary root.

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
- **THEN** each result's `file_path` SHALL be relative to the project's primary root, not an absolute path, reducing response payload size

#### Scenario: Distinct functions in the same file remain separate
- **WHEN** two top-ranked chunks originate from the same file but their line ranges are separated by more than the configured window
- **THEN** the system SHALL keep them as separate result entries
