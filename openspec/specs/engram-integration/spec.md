## ADDED Requirements

### Requirement: Vectos results SHALL be usable alongside session memory tools
The system SHALL expose code-context results in a form that can be combined with session-memory tools in agent workflows.

#### Scenario: Agent combines memory and code context
- **WHEN** an agent uses both session-memory tools and Vectos code search in the same task
- **THEN** the Vectos result SHALL provide file-localized code context that can be combined with memory-derived decisions
