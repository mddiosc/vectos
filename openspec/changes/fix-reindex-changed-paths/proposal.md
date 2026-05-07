## Why

When an MCP agent performs an incremental reindex after editing a file in an Nx dependency lib, it passes a relative path like `src/button.tsx`. The current `resolveChangedPath` only tries `WorkspaceRoot` and `PrimaryRoot` as resolution bases — it never tries the dependency lib roots in `scope.Roots`. The path fails to resolve and the change is silently ignored, leaving stale chunks in the index. Additionally, the managed agent guidance does not explain path semantics for `changed`, rename/move handling, docs-only refresh, or when to force a full reindex in Nx workspaces.

## What Changes

- **Extend `resolveChangedPath`** to try every root in `scope.Roots` as a resolution base (after `WorkspaceRoot` and `PrimaryRoot`), deduplicating and preserving priority order. Paths that resolve to a location inside `scope.Roots` are preferred.
- **Improve managed guidance** with a dedicated "Incremental Reindex" section covering: path format, renames/moves, docs-only refresh, full reindex triggers, and Nx shared-lib rules.

## Capabilities

### Modified Capabilities
- `workspace-awareness`: `resolveChangedPath` now resolves relative paths against all scope roots, not just workspace root and primary root
- `agent-setup`: managed guidance now includes explicit incremental reindex rules for both standard and Nx projects

## Impact

- `cmd/vectos/runtime_paths.go` — `resolveChangedPath` extended
- `cmd/vectos/runtime_paths_test.go` — new regression test
- `internal/setup/guidance_content.go` — new guidance section
