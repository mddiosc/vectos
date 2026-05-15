## Context

`vectos serve` keeps the embedding model and SQLite connections warm, but agents must still explicitly call `POST /reindex` on every file change. Agents routinely forget, leading to stale indexes. The OpenCode plugin sends reindex requests on file changes (debounced 3s), but this requires plugin-level orchestration — the server itself has no awareness of filesystem changes.

Adding a native filesystem watcher to `vectos serve` makes the index self-maintaining: when files change, the watcher triggers incremental reindexes automatically. This removes a fragile agent responsibility and makes the warm server truly zero-maintenance for index freshness.

## Goals / Non-Goals

**Goals:**
- Automatically detect file changes (create, modify, delete, rename) in watched directories
- Verify actual content changes via file hashes before triggering reindex (debounce)  
- Trigger incremental reindex for only changed files (not full project reindex)
- Clean up chunks when files are deleted
- Configurable via `--watch`, `--watch-debounce`, `--watch-ignore` flags
- Gracefully start and stop watcher alongside server lifecycle
- Coexist with manual `/reindex` calls (no conflict)

**Non-Goals:**
- Watching multiple disjoint project directories simultaneously
- Remote filesystem watching (NFS, network mounts)
- Recursive symlink following (follows project directory symlinks but not recursive symlink trees)
- `.gitignore` parsing in the watcher itself (relies on existing `content.CollectIndexablePaths` filtering at reindex time)
- Replacing the OpenCode plugin's file-change detection (plugin can still trigger `/reindex` explicitly — they complement each other)

## Decisions

### 1. Use `github.com/fsnotify/fsnotify`

**Choice**: Integrate `github.com/fsnotify/fsnotify` for cross-platform filesystem notifications.

**Alternatives considered**:
- **inotify/kqueue directly**: Platform-specific, more code. fsnotify abstracts this.
- **Polling with `os.Stat`**: Less responsive, CPU-wasteful, error-prone timing.
- **rnotify/hnotify**: Less maintained/smaller community.

**Rationale**: fsnotify is the de-facto standard Go filesystem notification library (gopls, Hugo, etc. use it). Zero native dependencies. Supports Linux, macOS, Windows. Active maintenance.

### 2. File-hash debouncing via SHA256 in SQLite

**Choice**: Store a SHA256 hash of each indexed file's content in the database. On fsnotify event, compute hash of the changed file and compare against stored hash. Only trigger reindex if hashes differ.

**Alternatives considered**:
- **Time-based debounce only**: FS events can fire multiple times for a single save (e.g., editor temp file + atomic rename). Time-based debounce helps but doesn't catch "write same content" events.
- **mtime comparison**: mtime is unreliable (can change without content change, can be preserved by some tools).
- **In-memory hash map**: Lost on restart. DB storage persists across server restarts.

**Rationale**: Content hashing is the only reliable way to know a file actually changed. SHA256 is fast enough for source files (microseconds per file). Persisting in DB means the watcher can detect "changed since last index" even after server restart.

### 3. Debounce with timer-based batching

**Choice**: On first fsnotify event, start a timer (default 500ms). Collect all subsequent events into a set. When timer fires, evaluate all collected paths for actual content changes and trigger one reindex.

**Alternatives considered**:
- **Per-file debounce**: Each file gets its own timer. Simpler but can lead to many small reindexes during batch operations (e.g., git checkout).
- **No debounce**: Every event triggers reindex immediately. Excessive for editors that do write+rename sequences.

**Rationale**: Timer-based batching is simple, handles bursty changes (git operations, multi-file saves), and results in one reindex per burst instead of N.

### 4. Hash column in chunks table vs separate table

**Choice**: Add a `file_hash` column to a new table `indexed_files` (path TEXT PRIMARY KEY, hash TEXT, indexed_at TIMESTAMP) rather than on the chunks table.

**Alternatives considered**:
- **Add `file_hash` to chunks table**: Redundant — same hash repeated for every chunk of the same file. Wasteful and harder to query.
- **In-memory only**: Lost on restart.

**Rationale**: A separate `indexed_files` table is cleaner — one row per indexed file. Easier to query "what was the hash when we last indexed this file?" without deduplication.

### 5. Watcher handles deletions directly

**Choice**: On file deletion events, immediately remove chunks from the index without going through the debounce/reindex pipeline.

**Alternatives considered**:
- **Debounce deletions like modifications**: Deletions don't benefit from debouncing.
- **Ignore deletions**: Orphaned chunks accumulate.

**Rationale**: Deletion is unambiguous — no content to hash. Clean up immediately to avoid stale results.

### 6. Watcher initialization at server startup

**Choice**: Start the watcher as part of `runServe()` after the embedding model is loaded but before the server is marked ready. The watcher runs in its own goroutine. Stop it during graceful shutdown.

**Rationale**: Starting early ensures no changes are missed. The watcher shouldn't block server readiness — reindex requests may come before the first file change.

## Risks / Trade-offs

- **[High CPU on large monorepos]** → Mitigation: Skip `.git`, `node_modules`, `*.lock` by default. `--watch-ignore` lets users add more patterns.
- **[Missing events on network mounts]** → Mitigation: Document that watch mode requires local filesystem. Network-mounted dirs will not receive events reliably.
- **[Recursive watch depth]** → Mitigation: fsnotify requires adding each subdirectory to the watch list. On startup, walk the project tree and add all non-ignored directories. Add new directories as they're created.
- **[Race between watcher and manual `/reindex`]** → Mitigation: The server already serializes `/reindex` requests with a mutex. Watcher-triggered reindexes use the same mutex.
- **[fsnotify buffer overflow]** → Mitigation: fsnotify events are buffered. If the event channel fills, events may be dropped. The debounce timer provides a safety net — even if some events are missed, the next batch will catch them.

## Open Questions

- Should we emit a log line when watcher detects changes and triggers reindex? (Yes — useful for debugging)
- Should `--watch` default to `true` or `false`? (True — the whole point of `vectos serve` is to be a persistent, self-maintaining index server)
- What happens on `.gitignore` changes? (Watcher doesn't re-evaluate ignores in real-time — this would require re-scanning the project. Users should restart serve if `.gitignore` changes significantly)
