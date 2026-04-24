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
