## ADDED Requirements

### Requirement: The system SHALL support repeatable retrieval benchmarks
The system SHALL allow users to run a repeatable benchmark that executes a set of retrieval queries against an indexed project.

#### Scenario: Run a benchmark against an indexed project
- **WHEN** a user runs a retrieval benchmark for a project that already has an index
- **THEN** the system SHALL execute each benchmark query against that project and produce a per-query result summary

### Requirement: Benchmark queries SHALL define expected useful targets
The system SHALL allow each benchmark query to define one or more expected useful files or chunks that count as a successful retrieval target.

#### Scenario: Query has multiple valid targets
- **WHEN** a benchmark query declares more than one expected useful target
- **THEN** the system SHALL treat the query as successful when any declared target appears within the evaluated ranking window

### Requirement: The system SHALL report positional usefulness metrics
The system SHALL report whether expected useful targets appear within configured top-ranked result windows.

#### Scenario: Expected target appears in top results
- **WHEN** a benchmark query returns an expected useful target within the configured top result window
- **THEN** the system SHALL mark that query as a successful hit for that metric window

#### Scenario: Expected target does not appear in top results
- **WHEN** a benchmark query does not return any expected useful target within the configured top result window
- **THEN** the system SHALL mark that query as unsuccessful for that metric window

### Requirement: Benchmark output SHALL be readable during normal development
The system SHALL produce benchmark output that makes it easy for a developer to inspect query names, expected targets, returned results, and aggregate hit rates.

#### Scenario: Review benchmark output after a run
- **WHEN** a benchmark run completes
- **THEN** the system SHALL present per-query and aggregate benchmark results in a format that can be compared across local iterations
