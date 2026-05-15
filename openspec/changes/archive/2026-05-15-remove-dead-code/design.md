## Context

Vectos started with a hand-rolled MCP protocol implementation in `internal/mcp/server.go`. The project later adopted the `go-sdk` MCP library and migrated to `cmd/vectos/`. The legacy `internal/mcp/` directory was left behind. Additionally, a stub function `inferGoPurpose` exists as a placeholder with no real implementation. Other dead code patterns (unreferenced exports, orphaned test helpers) may also be present.

The codebase is a Go monorepo. Standard Go tooling (`go vet`, `staticcheck`) can detect broken imports after removal.

## Goals / Non-Goals

**Goals:**
- Remove the entire `internal/mcp/` directory
- Remove the `inferGoPurpose` function and any associated dead code
- Identify and remove other unreferenced exported symbols and orphaned test helpers
- Ensure the project builds, passes `go vet`, and passes `staticcheck` after all removals

**Non-Goals:**
- Refactoring live code (only dead code is removed)
- Changing behavior of any existing feature
- Introducing new dependencies or features
- Aggressive optimization of import graphs beyond dead-code elimination

## Decisions

**Decision 1: Verify deadness with multiple independent methods**
- `vectos_search_code` — semantic search for references to `internal/mcp` and `inferGoPurpose`
- `grep` — exact string matching for import paths and function names
- `go vet ./...` — compiler-level verification that no broken imports remain
- `staticcheck ./...` — detects unreferenced exports and additional dead code

**Decision 2: Remove `internal/mcp/` atomically rather than piecemeal**
- The entire directory is either dead or not; no value in partial removal
- Single commit reduces churn

**Decision 3: Use `inferGoPurpose` as a canary for deeper dead-code scan**
- Remove `inferGoPurpose` first
- Run `staticcheck` to surface other unreferenced symbols
- Remove additional dead code in the same change if safe

## Risks / Trade-offs

- **Risk**: A non-obvious import (e.g., `_ "internal/mcp"` for side effects) could exist
  **Mitigation**: `go vet` and `go build` will fail on broken imports. No code uses side-effect imports of internal packages in this project.

- **Risk**: `inferGoPurpose` removal could break an integration test that calls it for setup
  **Mitigation**: Grep for all occurrences first; verify function body is truly a no-op stub.

- **Risk**: Aggressive `staticcheck`-driven removal could remove something intended for WIP feature
  **Mitigation**: Only remove code that is both unreferenced AND shows clear signs of being dead (stubs, references to deleted code, obvious orphans). Flag any ambiguous cases for review rather than blindly removing.
