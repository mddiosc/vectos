## Why

When an MCP agent (like an AI coding assistant) works inside an Nx monorepo, Vectos fails to scope searches correctly in several common scenarios: when the agent passes the workspace root as `path`, when it passes only `project` without `path`, and when the CLI encounters an ambiguous workspace. These gaps confuse the agent, produce unusable error messages, and block effective semantic search. The Nx dependency expansion logic works correctly — we need to fix how the scope is _reached_ without touching the expansion itself.

## What Changes

- **Fix `selectNxProject`** to produce an actionable error listing available projects when `path` is the workspace root with multiple projects, instead of a generic "multiple projects detected" error
- **Fix `resolveToolScope`** to resolve a complete `Scope` (with `PrimaryRoot`, `WorkspaceRoot`, `Roots`) when only `project` is given without `path`, by using `os.Getwd()` as the starting point
- **Fix `resolveRuntimeScope`** to propagate ambiguous-workspace errors to the CLI instead of silently returning `nil`
- **Add `list_projects` MCP tool** that exposes `DiscoverNxProjectNames` so agents can discover available Nx projects before indexing or searching
- **Update client guidance** (`guidance_content.go`) to mention the `project` parameter and Nx monorepo workflow
- **NO CHANGE** to `resolveNxProjectRoots` or any dependency expansion logic — the hard constraint is that indexing a project must still index all linked libs

## Capabilities

### New Capabilities
- `nx-project-listing`: Expose an MCP tool to list available Nx project names in a workspace

### Modified Capabilities
- `workspace-awareness`: Improve error guidance when the Nx workspace root is passed as path with multiple projects; fix incomplete scope resolution when project name is given without path
- `mcp-interface`: Add `list_projects` tool to the MCP server; fix `search_code`/`search_docs`/`index_project` to resolve full scopes when only `project` is provided

## Impact

- `internal/workspace/workspace.go` — `selectNxProject` returns richer errors
- `cmd/vectos/runtime_paths.go` — `resolveToolScope` and `resolveRuntimeScope` behavior changes
- `cmd/vectos/mcp_server.go` — new `listProjectsInput`, `list_projects` tool registration
- `cmd/vectos/mcp_handlers.go` — new `makeListProjectsHandler`
- `internal/setup/guidance_content.go` — Nx monorepo guidance
- Tests: `internal/workspace/workspace_test.go`, `cmd/vectos/runtime_paths_test.go` — new test cases
