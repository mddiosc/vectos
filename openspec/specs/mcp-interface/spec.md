## ADDED Requirements

### Requirement: The system SHALL expose an MCP server over stdio
The system SHALL provide a Model Context Protocol server over stdio so external agents can discover and call Vectos tools.

#### Scenario: MCP client initializes successfully
- **WHEN** an MCP-compatible client connects to the server
- **THEN** the server SHALL complete the MCP initialization handshake and advertise tool capabilities

### Requirement: The MCP server SHALL expose code search and indexing tools
The system SHALL expose tools for code search and project indexing through MCP.

#### Scenario: MCP client lists available tools
- **WHEN** an MCP client requests the available tools
- **THEN** the server SHALL return at least `search_code` and `index_project`

#### Scenario: MCP client calls code search
- **WHEN** an MCP client calls `search_code` with a query
- **THEN** the server SHALL execute the search against the active project index and return the result content in MCP tool result format with enough ranking metadata for an agent to choose what to inspect next

#### Scenario: MCP client calls project indexing
- **WHEN** an MCP client calls `index_project` with a file or directory path
- **THEN** the server SHALL index the requested path and return a summary of the indexing operation

#### Scenario: MCP client calls project indexing for changed project content
- **WHEN** an MCP client calls `index_project` for a scope that already has indexed content
- **THEN** the system SHALL be able to refresh only the changed files and return a summary of the applied indexing update

#### Scenario: Agent guidance for mixed memory and code workflows
- **WHEN** an MCP-compatible agent has access to both session-memory tools and Vectos MCP tools
- **THEN** the recommended guidance SHALL prefer memory tools for prior decisions and Vectos tools for current code context without implying that Vectos depends on the presence of memory tools

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

### Requirement: Indexed chunks SHALL preserve function signatures and purpose
The system SHALL store function signatures and inferred purposes alongside chunk content so they are available in search output without re-parsing.

#### Scenario: Chunk is saved during indexing
- **WHEN** the system persists a code chunk to the database
- **THEN** it SHALL also store the extracted function signature (if any) in a `signature` column and the inferred purpose description (if any) in a `purpose` column

#### Scenario: Chunk without structural signature
- **WHEN** a chunk does not contain a recognizable structural signature
- **THEN** the `signature` column may be empty, and the system SHALL fall back to the inferred `purpose` or category for the output `hint`

### Requirement: MCP search failures SHALL suggest the next useful action
The system SHALL provide explicit recovery guidance when MCP search cannot return useful results because the project is missing an index or requires refresh.

#### Scenario: Project is not indexed
- **WHEN** an MCP client calls `search_code` for a project scope that has no usable index
- **THEN** the system SHALL indicate that indexing is required and identify the relevant indexing action

#### Scenario: Project index is stale or incomplete
- **WHEN** an MCP client calls `search_code` and the system can determine that the available project index is stale or incomplete
- **THEN** the system SHALL indicate that a refresh is recommended and identify the relevant indexing action
