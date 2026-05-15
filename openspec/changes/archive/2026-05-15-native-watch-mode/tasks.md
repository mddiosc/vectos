## 1. Database Schema

- [x] 1.1 Add `indexed_files` table to SQLite schema (path TEXT PRIMARY KEY, hash TEXT NOT NULL, indexed_at TIMESTAMP NOT NULL)
- [x] 1.2 Write database migration to create `indexed_files` table on existing databases
- [x] 1.3 Add upsert/get/delete operations for `indexed_files` rows in storage layer

## 2. File Hash Tracking in Indexer

- [x] 2.1 Compute SHA256 hash of file content during indexing and store in `indexed_files` table
- [x] 2.2 Delete `indexed_files` row when a file's chunks are removed (deletion or reindex)
- [x] 2.3 Add `HasFileChanged(path string) (bool, error)` method that compares current file hash against stored hash
- [x] 2.4 Add `RemoveDeletedFile(path string)` method to clean up chunks + hash row for deleted files

## 3. Add fsnotify Dependency

- [x] 3.1 Add `github.com/fsnotify/fsnotify` to go.mod (`go get github.com/fsnotify/fsnotify`)
- [x] 3.2 Verify build succeeds with new dependency on all target platforms (darwin, linux)

## 4. CLI Flags for Watch Mode

- [x] 4.1 Add `--watch` (bool, default true), `--watch-debounce` (duration, default 500ms), `--watch-ignore` (string, default `.git,node_modules,*.lock`) flags to CLI flag definitions
- [x] 4.2 Wire watch flags through serve command dispatch to server initialization

## 5. Watcher Core Implementation

- [x] 5.1 Create `internal/watcher/` package with `Watcher` struct wrapping fsnotify
- [x] 5.2 Implement `NewWatcher(rootPath string, ignorePatterns []string, debounce time.Duration, onChange func([]string)) (*Watcher, error)`
- [x] 5.3 Implement recursive directory watching: on start, walk the project tree and add all non-ignored directories to fsnotify
- [x] 5.4 Implement ignore-pattern matching (glob) for paths before adding to watch list
- [x] 5.5 Handle new directory creation events by adding new directories to the watch list
- [x] 5.6 Implement debounce: collect events into a set, start/reset timer on each event, fire onChange callback with collected paths after timer expires
- [x] 5.7 Implement `Start(ctx context.Context)` method that runs the event loop in a goroutine
- [x] 5.8 Implement `Stop()` method that cancels the context, stops the watcher, and cleans up

## 6. File Deletion Handling

- [x] 6.1 On file delete/rename events, immediately remove chunks from index (do not wait for debounce)
- [x] 6.2 Remove `indexed_files` row when chunks are deleted
- [x] 6.3 Log deletion events for observability

## 7. Server Integration

- [x] 7.1 In `runServe()`, after embedding model is loaded, initialize watcher with configured flags
- [x] 7.2 Wire watcher's onChange callback to trigger incremental reindex using existing reindex mutex/serialization
- [x] 7.3 Pass file hashes to reindex flow so unchanged files can be skipped
- [x] 7.4 Add watcher shutdown to graceful server shutdown sequence (stop watcher before closing DB connections)
- [x] 7.5 Add log output when watcher detects changes and triggers reindex

## 8. Testing

- [x] 8.1 Write unit tests for `indexed_files` CRUD operations in storage layer
- [x] 8.2 Write unit tests for file hash computation and change detection
- [x] 8.3 Write unit tests for watcher ignore-pattern matching
- [x] 8.4 Write unit tests for debounce timer behavior
- [x] 8.5 Write integration test: create temp directory, index files, start watcher, modify file, verify chunks updated
- [x] 8.6 Write integration test: delete file while watcher is running, verify chunks removed
- [x] 8.7 Write integration test: watcher with `--watch=false` does not trigger auto-reindex

## 9. Documentation

- [x] 9.1 Update `vectos serve --help` text to document `--watch`, `--watch-debounce`, `--watch-ignore` flags
- [x] 9.2 Document watch mode behavior and limitations in README (local filesystem only, default ignore patterns)
