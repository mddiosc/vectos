## MODIFIED Requirements

### Requirement: Vectos SHALL support logical project scoping in Nx workspaces
Vectos SHALL support indexing and searching a logical project composed of multiple filesystem roots when that project is resolved from Nx workspace metadata. Vectos SHALL include all internal dependency libs in the scope by default, and file-level docs/code filtering SHALL decide what content is indexed from those roots.

#### Scenario: Index a selected Nx project with multiple roots
- **WHEN** the selected Nx project resolves to multiple paths
- **THEN** Vectos SHALL index all included paths as one project scope

#### Scenario: Internal libs with docs-like names are still included
- **WHEN** an internal dependency lib name or path contains substrings such as `docs`, `stories`, or `storybook`
- **THEN** Vectos SHALL still include that lib in the resolved scope when Nx reports it as an internal dependency

## ADDED Requirements

### Requirement: Vectos SHALL expose Nx graph resolution warnings and e2e overrides
When Nx graph resolution is incomplete or when the workspace contains e2e projects, Vectos SHALL surface the limitation clearly and SHALL support an explicit override for indexing e2e projects.

#### Scenario: Nx graph failure surfaces a warning
- **WHEN** Vectos cannot resolve the Nx graph for a workspace
- **THEN** it SHALL retain the primary project root and surface a warning that Nx graph expansion was incomplete

#### Scenario: E2E projects are excluded by default
- **WHEN** Nx reports a dependency project whose type is `e2e`
- **THEN** Vectos SHALL exclude that project from the default resolved scope

#### Scenario: E2E projects can be included with an environment override
- **WHEN** the environment variable `VECTOS_NX_INCLUDE_E2E=1` is set
- **THEN** Vectos SHALL include e2e projects in the resolved scope instead of excluding them by default
