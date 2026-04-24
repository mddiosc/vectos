## ADDED Requirements

### Requirement: Project indexes SHALL be isolated by working project
The system SHALL store indexed code in a project-specific database derived from the current working project context.

#### Scenario: Resolve the active project database
- **WHEN** the system initializes storage for indexing or search
- **THEN** it SHALL derive the database path from the current working project and use that project-specific database

#### Scenario: Create missing project storage directories
- **WHEN** the project-specific storage directory does not exist
- **THEN** the system SHALL create the directory before opening the database

### Requirement: Search SHALL operate on the active project index only
The system SHALL query only the active project's indexed chunks unless explicitly directed otherwise.

#### Scenario: Search in one project does not leak another project's chunks
- **WHEN** a user or agent performs a search from within a project
- **THEN** the system SHALL return only results stored in that project's index
