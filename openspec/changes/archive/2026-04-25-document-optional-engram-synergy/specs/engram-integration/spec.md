## MODIFIED Requirements

### Requirement: Vectos results SHALL be usable alongside session memory tools
The system SHALL expose code-context results in a form that can be combined with session-memory tools in agent workflows.

#### Scenario: Agent combines memory and code context when both systems are available
- **WHEN** an agent uses both Engram session-memory tools and Vectos code search in the same task
- **THEN** the recommended workflow SHALL treat memory retrieval and code retrieval as complementary context sources without making either one a hard dependency of the other

#### Scenario: Vectos used without Engram
- **WHEN** Vectos is installed without Engram
- **THEN** Vectos SHALL remain fully usable for indexing and semantic code retrieval as a standalone product
