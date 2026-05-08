## ADDED Requirements

### Requirement: Vectos SHALL support logical project scoping in Nx workspaces
Vectos SHALL support indexing and searching a logical project composed of multiple filesystem roots when that project is resolved from Nx workspace metadata.

#### Scenario: Index a selected Nx project with multiple roots
- **WHEN** the selected Nx project resolves to multiple paths
- **THEN** Vectos SHALL index all included paths as one project scope

### Requirement: Vectos SHALL detect Nx project boundaries
Vectos SHALL detect Nx project boundaries and roots in the initial monorepo implementation.

#### Scenario: Resolve an Nx project
- **WHEN** the current repository contains Nx workspace metadata and the user selects a project
- **THEN** Vectos SHALL resolve the project roots from Nx workspace configuration

### Requirement: Vectos SHALL require explicit project selection when needed
Vectos SHALL expose explicit logical-project selection for Nx workspaces when the workspace contains multiple candidate projects.

#### Scenario: Select project from Nx workspace
- **WHEN** the current repository contains multiple Nx projects
- **THEN** Vectos SHALL require the caller to identify which Nx project scope to index or search
## Requirements
### Requirement: Vectos SHALL resolve a complete Scope when project name is given without path
When an MCP tool or CLI command receives only a project name (no path), the system SHALL resolve a complete `Scope` that includes `PrimaryRoot`, `WorkspaceRoot`, and `Roots` by using the current working directory as the starting point for workspace detection.

#### Scenario: search_code called with project but no path
- **WHEN** an MCP client calls `search_code` with `project: "app-one"` and no `path`
- **THEN** the server SHALL resolve a `Scope` with a non-empty `PrimaryRoot` corresponding to the "app-one" project root, and search results SHALL use paths relative to that `PrimaryRoot`

#### Scenario: index_project called with project but no path
- **WHEN** an MCP client calls `index_project` with `project: "app-two"` and no `path`
- **THEN** the server SHALL resolve a `Scope` that includes the "app-two" project root and all its internal dependency roots, and index all of them

### Requirement: Vectos SHALL provide actionable error messages when the Nx project scope is ambiguous
When the system cannot determine which Nx project to use because the path is the workspace root and multiple projects exist, it SHALL return an error message that includes the list of available project names.

#### Scenario: ResolveScope called with workspace root and multiple projects
- **WHEN** `ResolveScope` is called with a path equal to the Nx workspace root and no project name, and the workspace contains multiple projects
- **THEN** the system SHALL return an error whose message identifies that a specific project is needed and lists the available project names

#### Scenario: ResolveScope called with workspace root and a single project
- **WHEN** `ResolveScope` is called with a path equal to the Nx workspace root and no project name, and the workspace contains exactly one project
- **THEN** the system SHALL automatically select that project (existing behavior preserved)

### Requirement: Vectos SHALL require explicit project selection when needed
Vectos SHALL expose explicit logical-project selection for Nx workspaces when the workspace contains multiple candidate projects.

#### Scenario: Select project from Nx workspace
- **WHEN** the current repository contains multiple Nx projects
- **THEN** Vectos SHALL require the caller to identify which Nx project scope to index or search

#### Scenario: Ambiguous selection returns actionable error
- **WHEN** the current repository contains multiple Nx projects and the requested path does not uniquely identify one
- **THEN** Vectos SHALL return an error that lists the available project names so the caller can retry with an explicit selection

### Requirement: Vectos SHALL resolve changed paths against all scope roots
When resolving a relative changed path for incremental reindex, the system SHALL try every root in the project scope as a resolution base, not only the workspace root and primary root.

#### Scenario: Relative changed path resolves against a dependency lib root
- **WHEN** `resolveChangedPath` is called with a relative path (e.g. `src/button.tsx`) and the file exists under a dependency lib root in `scope.Roots` (e.g. `/workspace/libs/lib-ui/src/button.tsx`)
- **THEN** the system SHALL return the absolute path under that dependency lib root

#### Scenario: Workspace-relative paths still resolve first
- **WHEN** `resolveChangedPath` is called with a path that is relative to the workspace root (e.g. `libs/lib-ui/src/button.tsx`)
- **THEN** the system SHALL resolve it against the workspace root as before, without change

#### Scenario: Primary-root-relative paths still resolve correctly
- **WHEN** `resolveChangedPath` is called with a path relative to the primary project root (e.g. `src/main.ts`) and the file exists under the primary root
- **THEN** the system SHALL return the absolute path under the primary root, unchanged from current behavior

#### Scenario: Absolute paths are returned unchanged
- **WHEN** `resolveChangedPath` is called with an absolute path
- **THEN** the system SHALL return `filepath.Clean(path)` without attempting any base resolution

