## Why

Reindexing an entire project on every meaningful code change is slower than necessary and makes it harder to keep indexes fresh during normal development. Vectos needs an incremental path so standalone usage remains fast enough to feel natural in daily workflows.

## What Changes

- Add an incremental reindexing path that updates only changed files instead of rebuilding the whole project by default in every workflow.
- Define how changed files are detected and how removed or no-longer-indexable files are purged.
- Provide a standalone-friendly workflow that can later be automated by hooks or wrappers without making hooks mandatory.

## Capabilities

### New Capabilities
- None.

### Modified Capabilities
- `code-indexing`: indexing behavior will support targeted refresh of changed files and cleanup of removed/stale chunks.
- `mcp-interface`: MCP indexing flows may return summaries that reflect incremental updates instead of only full reindex behavior.

## Impact

- Affected code: CLI indexing flow, MCP indexing path, storage cleanup behavior, and project index maintenance
- Affected behavior: faster refresh after changes and clearer semantics for stale-chunk cleanup
- Dependencies: may optionally integrate with git-based changed-file detection, but must not require git hooks to be useful
