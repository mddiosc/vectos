## Context

`resolveChangedPath` converts a changed path (absolute or relative) into an absolute path that can be matched against the indexed file list. For relative paths it currently tries two bases in order: `WorkspaceRoot` (if different from `PrimaryRoot`) and `PrimaryRoot`. In an Nx workspace, `scope.Roots` contains the primary project root plus all internal dependency lib roots. A relative path like `src/button.tsx` that belongs to `libs/lib-ui/src/button.tsx` will never resolve correctly because `libs/lib-ui` is not in the current base list.

## Goals / Non-Goals

**Goals:**
- Relative changed paths that belong to dependency lib roots resolve correctly
- Existing resolution behavior for workspace-relative and primary-root-relative paths is preserved
- No silent no-ops: a changed path that exists in a dependency lib is matched and re-indexed

**Non-Goals:**
- File watchers, mtime/hash comparison, or automatic stale detection
- Changes to `resolveNxProjectRoots` or dependency expansion logic
- New CLI flags or MCP schema changes

## Decisions

### D1: Append `scope.Roots` to the base list, deduplicated
**Choice:** Build the base list as `[WorkspaceRoot, PrimaryRoot, ...scope.Roots]` with a seen-set to skip duplicates.

**Rationale:** `WorkspaceRoot` and `PrimaryRoot` are already in `scope.Roots` in most cases. Deduplication avoids redundant resolution attempts. Priority order is preserved: workspace-relative paths resolve first (most common case for agents), then primary-root-relative, then dependency-lib-relative.

### D2: Prefer candidates inside `scope.Roots` via `isWithinRoots`
**Choice:** Keep the existing `isWithinRoots(resolved, scope.Roots) || fileExists(resolved)` check inside the loop.

**Rationale:** `isWithinRoots` ensures the resolved path is actually within the indexed scope. `fileExists` is the fallback for paths that exist on disk but may not yet be indexed (new files). This is unchanged behavior — we just run it against more bases.

### D3: Fallback to `PrimaryRoot` unchanged
**Choice:** Keep the final fallback `filepath.Abs(filepath.Join(scope.PrimaryRoot, changed))` for paths that don't resolve against any base.

**Rationale:** Preserves existing behavior for paths that don't exist yet (new files being indexed for the first time). The caller (`filterChangedPaths`) will then check `isWithinRoots` and drop out-of-scope paths.

## Risks / Trade-offs

- **[Low] Performance**: iterating more bases per changed path. `scope.Roots` is typically small (< 20 entries). No measurable impact.
- **[Low] Ambiguity**: a relative path like `src/index.ts` could exist in both `apps/app-one/src/index.ts` and `libs/lib-ui/src/index.ts`. The first match wins (workspace root → primary root → dependency roots). This is deterministic and consistent with the existing priority.
- **[None] Regression risk**: absolute paths are returned immediately before any base iteration — that path is unchanged.
