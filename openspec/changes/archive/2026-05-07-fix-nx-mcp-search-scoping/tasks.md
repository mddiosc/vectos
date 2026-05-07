## 1. Regression guards — ensure existing Nx expansion logic still passes

- [x] 1.1 Run `go test ./internal/workspace/... -v` and confirm TestResolveScopeExpandsNxDependencyRoots, TestResolveScopeFallsBackToPrimaryRootWhenNxGraphUnavailable, TestResolveScopeExcludesE2EAndDocsProjects, and TestLoadNxGraphCachesByWorkspaceRoot all pass
- [x] 1.2 Run `go test ./cmd/vectos/... -v` and confirm existing runtime_paths tests pass
- [x] 1.3 Add regression test: `TestResolveScopeWorkspaceRootListsProjects` in `internal/workspace/workspace_test.go` — creates workspace root with multiple projects, calls ResolveScope(workspaceRoot, ""), expects error containing project names

## 2. Gap 2 (P1) — selectNxProject error enrichment at workspace root

- [x] 2.1 Modify `selectNxProject` in `internal/workspace/workspace.go` to detect when `startDir` equals the workspace root and `requestedName` is empty with multiple projects, then return an error listing available project names (format: `"path is the Nx workspace root; please specify a project name. Available projects: web, ui"`)
- [x] 2.2 Ensure single-project workspace root case continues to auto-select the project (existing behavior preserved)
- [x] 2.3 Run `go test ./internal/workspace/...` — all existing tests pass + new TestResolveScopeWorkspaceRootListsProjects passes

## 3. Gap 1 (P2) — resolveToolScope with project-only resolves full Scope

- [x] 3.1 Modify `resolveToolScope` in `cmd/vectos/runtime_paths.go` to call `resolveRuntimeScope(projectName)` when `path` is empty and `projectName` is not, instead of returning bare `Scope{Name: projectName}`
- [x] 3.2 Add test: `TestResolveToolScopeWithProjectOnlyResolvesFullScope` in `cmd/vectos/runtime_paths_test.go` — creates Nx workspace with project and dependency, calls `resolveToolScope("", "app-one")`, verifies PrimaryRoot is non-empty and Roots includes dependency
- [x] 3.3 Verify MCP search results paths are relativized: create test that verifies `buildMCPSearchPayload` produces relative paths when scope has PrimaryRoot
- [x] 3.4 Run `go test ./cmd/vectos/...` — all tests pass

## 4. Gap 5 — Propagate workspace ambiguity error in CLI

- [x] 4.1 Modify `resolveRuntimeScope` in `cmd/vectos/runtime_paths.go` to remove the `if strings.TrimSpace(projectName) == "" { return nil, nil }` block; always propagate the error from `workspace.ResolveScope`
- [x] 4.2 Add test: `TestResolveRuntimeScopePropagatesAmbiguousWorkspaceError` — verifies error is returned (not nil) when wd is workspace root with multiple projects
- [x] 4.3 Run `go test ./cmd/vectos/...` — all tests pass; verify CLI `vectos search` from workspace root without `--project` shows the enriched error message

## 5. Gap 3 — New MCP tool list_projects

- [x] 5.1 Add `listProjectsInput` struct and `listProjectsOutput` struct in `cmd/vectos/mcp_server.go`
- [x] 5.2 Add `makeListProjectsHandler` in `cmd/vectos/mcp_handlers.go` — uses `workspace.DiscoverNxProjectNames`, returns `{"projects": [...]}` or `{"projects": []}`
- [x] 5.3 Register `list_projects` tool in `registerMCPTools`
- [x] 5.4 Add test: `TestListProjectsHandlerInNxWorkspace` and `TestListProjectsHandlerOutsideNxWorkspace` in `cmd/vectos/mcp_handlers_test.go` (or new test file)
- [x] 5.5 Run `go test ./cmd/vectos/...` — handler tests pass

## 6. Gap 4 — Update client guidance for Nx monorepo workflow

- [x] 6.1 Modify `managedGuidance` in `internal/setup/guidance_content.go` to add a section explaining the `project` parameter for Nx monorepos, mentioning `list_projects` to discover available projects, and noting that `index_project` with `project` indexes dependency libs automatically
- [x] 6.2 Verify generated guidance contains `project`, `Nx`, and `list_projects` keywords

## 7. Final verification

- [x] 7.1 Run full test suite: `go test ./...` (39 passed, 11 packages)
- [x] 7.2 Run `go build ./cmd/vectos` — binary compiles
- [x] 7.3 Manual smoke test: in an Nx workspace, run `vectos mcp` and verify `list_projects` appears in tool list
- [x] 7.4 Manual smoke test: call `search_code` with only `project` (no `path`) and verify results have relative paths
