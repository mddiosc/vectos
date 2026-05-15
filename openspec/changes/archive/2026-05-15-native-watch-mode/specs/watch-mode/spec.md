## ADDED Requirements

### Requirement: Watcher detects filesystem changes during serve
The system SHALL start a filesystem watcher when `vectos serve` is run with `--watch` enabled (default true). The watcher SHALL monitor the project directory for file create, modify, delete, and rename events on supported source files.

#### Scenario: File is created in watched directory
- **WHEN** a new supported source file is created inside the watched project directory
- **THEN** the watcher SHALL detect the create event and trigger an incremental reindex of the new file

#### Scenario: File is modified in watched directory
- **WHEN** an existing indexed file is modified in the watched project directory
- **THEN** the watcher SHALL detect the modify event and trigger an incremental reindex of the changed file

#### Scenario: File is deleted from watched directory
- **WHEN** an indexed file is deleted from the watched project directory
- **THEN** the watcher SHALL detect the delete event and remove the file's chunks from the index

#### Scenario: File is renamed in watched directory
- **WHEN** an indexed file is renamed within the watched project directory
- **THEN** the watcher SHALL detect the rename event, remove chunks for the old path, and trigger reindex of the new path

#### Scenario: Watcher is disabled via --watch=false
- **WHEN** `vectos serve` is run with `--watch=false`
- **THEN** no filesystem watcher SHALL be started and the server SHALL rely on manual `/reindex` calls only

### Requirement: File-hash verification prevents redundant reindexing
The system SHALL store a SHA256 hash of each indexed file's content in the database. On a filesystem change event, the system SHALL compute the current file hash and compare it against the stored hash. If the hashes match, the file SHALL NOT be reindexed.

#### Scenario: File event fires but content is unchanged
- **WHEN** a filesystem event is received for a file whose current content hash matches the stored hash
- **THEN** the system SHALL skip reindexing that file

#### Scenario: File event fires and content has changed
- **WHEN** a filesystem event is received for a file whose current content hash differs from the stored hash
- **THEN** the system SHALL trigger an incremental reindex of that file and update the stored hash

#### Scenario: File is indexed for the first time
- **WHEN** a file is indexed that has no prior hash stored
- **THEN** the system SHALL compute and store its SHA256 hash after indexing

### Requirement: Debounce batches rapid file change events
The system SHALL debounce filesystem events by collecting all events within a configurable time window (default 500ms) before triggering a reindex. The debounce timer SHALL start on the first event after a quiet period and reset on each subsequent event within the window.

#### Scenario: Multiple files saved in rapid succession
- **WHEN** several files are modified within the debounce window (e.g., a git checkout or multi-file save)
- **THEN** the system SHALL collect all changed paths and trigger a single reindex after the window expires

#### Scenario: Single file saved with editor temp-file sequence
- **WHEN** an editor saves a file using a write-then-rename pattern that generates multiple events within the debounce window
- **THEN** the system SHALL debounce the events and trigger only one reindex for the final file path

#### Scenario: Custom debounce duration
- **WHEN** `vectos serve` is run with `--watch-debounce=1000`
- **THEN** the debounce window SHALL be 1000ms instead of the default 500ms

### Requirement: Watch-ignore patterns filter monitored paths
The system SHALL support a `--watch-ignore` flag that accepts a comma-separated list of glob patterns. Directories and files matching any pattern SHALL NOT be added to the watcher's recursive directory watch list.

#### Scenario: Default ignore patterns are applied
- **WHEN** `vectos serve` is run without an explicit `--watch-ignore` flag
- **THEN** the watcher SHALL skip `.git`, `node_modules`, and `*.lock` paths by default

#### Scenario: Custom ignore patterns are applied
- **WHEN** `vectos serve` is run with `--watch-ignore=.build,*.gen.go`
- **THEN** the watcher SHALL skip paths matching `.build` or `*.gen.go` in addition to the defaults

#### Scenario: New directory created matching ignore pattern
- **WHEN** a new directory is created that matches a watch-ignore pattern
- **THEN** the watcher SHALL NOT add it to the watch list

### Requirement: Watcher lifecycle is tied to server lifecycle
The system SHALL start the watcher during server initialization after the embedding model is loaded. The watcher SHALL run in its own goroutine. On graceful server shutdown (SIGTERM/SIGINT), the watcher SHALL be stopped before closing database connections.

#### Scenario: Watcher starts with server
- **WHEN** `vectos serve` starts with `--watch` enabled
- **THEN** the watcher SHALL be initialized and begin monitoring before the server accepts its first `/reindex` request

#### Scenario: Watcher stops on server shutdown
- **WHEN** the server receives SIGTERM or SIGINT
- **THEN** the watcher SHALL stop monitoring, the current debounce batch SHALL be processed (or discarded), and the watcher goroutine SHALL exit before SQLite connections are closed

#### Scenario: Watcher does not block server readiness
- **WHEN** the watcher encounters a non-fatal error during initialization (e.g., a single directory permission error)
- **THEN** the server SHALL still start and accept requests, logging the error
