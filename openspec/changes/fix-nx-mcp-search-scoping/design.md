## Context

Vectos supports Nx monorepos by detecting `nx.json`, discovering `project.json` files to map projects, selecting a project based on path or explicit name, and expanding its internal dependencies via `nx graph --print`. The `Scope` struct carries the project name, workspace root, primary root (the selected project's directory), and the expanded set of `Roots` (including dependency libs).

The current implementation has five gaps in how scope is resolved from MCP/CLI inputs:

1. **`resolveToolScope` with `project` but no `path`** returns a bare `Scope{Name: projectName}` — missing `PrimaryRoot`, `WorkspaceRoot`, and `Roots`. This causes MCP results to show absolute paths and prevents dependency expansion.
2. **`selectNxProject` at workspace root** fails with a generic error when `path` is the workspace root and multiple projects exist, because `sameOrUnder(workspaceRoot, projectRoot)` never matches.
3. **`resolveRuntimeScope` silently swallows** the ambiguous-workspace error, returning `(nil, nil)`.
4. **`DiscoverNxProjectNames`** exists but has no callers — no MCP tool or error message uses it.
5. **Client guidance** in `guidance_content.go` doesn't mention the `project` parameter or Nx workflow.

The Nx dependency expansion itself (`resolveNxProjectRoots`) is correct and MUST NOT be changed.

## Goals / Non-Goals

**Goals:**
- Make `selectNxProject` return an actionable error with the list of available projects when `path` is the workspace root with multiple projects
- Make `resolveToolScope` resolve a complete `Scope` when `project` is given without `path`
- Eliminate silent error swallowing in `resolveRuntimeScope`
- Expose `list_projects` as an MCP tool
- Update client guidance to mention Nx workflow and the `project` parameter

**Non-Goals:**
- Changing `resolveNxProjectRoots`, `internalDependencyTargets`, `shouldExcludeNxProject`, or any dependency expansion logic
- Changing the storage layer, embedding model, chunking, or DB schema
- Adding Nx workspace detection to the client side (only server-side)
- Supporting non-Nx monorepo tools (Turborepo, Lerna, etc.)

## Decisions

### D1: Error enrichment in `selectNxProject` vs. new function
**Choice:** Modify `selectNxProject` to detect the workspace-root case and include available project names in the error message.

**Rationale:** `selectNxProject` already has the full project list. Adding detection of the workspace-root case is a small conditional addition. Creating a separate pre-check function would duplicate the project discovery call. The error message format will be: `"path is the Nx workspace root; please specify a project name. Available projects: web, ui, auth"`.

**Alternative considered:** Have the MCP handler catch the generic error and re-resolve to get project names. Rejected because it adds an extra `discoverNxProjects` call and couples MCP logic to workspace internals.

### D2: `resolveToolScope` with project-only uses `os.Getwd()`
**Choice:** When `path` is empty and `projectName` is not, call `workspace.ResolveScope(wd, projectName)` instead of constructing an incomplete `Scope`.

**Rationale:** This is consistent with `resolveRuntimeScope` which already uses `os.Getwd()`. It guarantees the caller always receives a fully-populated `Scope` with `PrimaryRoot`, `WorkspaceRoot`, and `Roots`.

### D3: `list_projects` returns empty array for non-Nx paths (not an error)
**Choice:** When called outside an Nx workspace, `list_projects` returns `{"projects": []}` rather than an error.

**Rationale:** Not being in an Nx workspace is a normal state, not an error. Returning an empty list lets agents call this tool unconditionally without error handling.

### D4: `list_projects` accepts optional `path` parameter
**Choice:** The tool accepts an optional `path` to scope discovery to a specific workspace root.

**Rationale:** Consistent with `search_code` and `index_project` which also accept `path`. When `path` is empty, defaults to `os.Getwd()`.

### D5: No project auto-selection at workspace root
**Choice:** When `path` is the workspace root and no `project` is specified, the system continues to require explicit selection (no auto-pick of first project or heuristic).

**Rationale:** Auto-selecting a project from the workspace root would be unpredictable. The improved error message gives the agent enough information to choose. This is consistent with the existing spec requirement: "Vectos SHALL require the caller to identify which Nx project scope to index or search."

## Risks / Trade-offs

- **[Risk] `resolveRuntimeScope` error propagation breaks existing CLI flows that relied on nil fallback** → Mitigation: `openStorageForScope` has its own fallback when `scope == nil`, but this path is only hit when `resolveRuntimeScope` swallows the error. After the fix, the error propagates up and the CLI exits with a clear message instead of a confusing downstream failure.

- **[Risk] `resolveToolScope` using `os.Getwd()` when the MCP server's CWD is different from the user's project** → Mitigation: The MCP server is typically launched with CWD set to the project root by the IDE/agent host. This is the same assumption already made by `resolveRuntimeScope`. If needed, the `path` parameter can override it.

- **[Risk] Large Nx workspaces with hundreds of projects produce very long error messages** → Mitigation: The project list is capped in the error at a reasonable length (~10 projects with "and N more"). `list_projects` tool returns the full list for programmatic use.

- **[Risk] Tests for `resolveNxProjectRoots` break inadvertently** → Mitigation: None of the changes touch `resolveNxProjectRoots` or its callers' logic. The existing tests `TestResolveScopeExpandsNxDependencyRoots`, `TestResolveScopeFallsBackToPrimaryRootWhenNxGraphUnavailable`, and `TestResolveScopeExcludesE2EAndDocsProjects` will be run before and after every change as regression guards.
