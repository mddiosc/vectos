## ADDED Requirements

### Requirement: Managed guidance SHALL explain incremental reindex path semantics
The managed guidance injected into agent configuration files SHALL include explicit rules for using the `changed` parameter correctly, covering path format, renames, docs refresh, and full reindex triggers.

#### Scenario: Agent uses correct path format for changed files
- **WHEN** an agent reads the managed guidance
- **THEN** it SHALL find instructions to pass changed paths as absolute paths or as paths relative to the workspace root — not relative to an individual project or lib root

#### Scenario: Agent handles renames and moves correctly
- **WHEN** an agent reads the managed guidance
- **THEN** it SHALL find instructions to include both the old and the new path when a file is renamed or moved

#### Scenario: Agent knows when to refresh the docs index
- **WHEN** an agent reads the managed guidance
- **THEN** it SHALL find instructions to run a separate `index_project` with `docs: true` after editing documentation files

#### Scenario: Agent knows when to force a full reindex
- **WHEN** an agent reads the managed guidance
- **THEN** it SHALL find instructions to perform a full reindex after changing workspace-level configuration files such as `nx.json`, `project.json`, or lockfiles

#### Scenario: Agent understands Nx shared-lib reindex rules
- **WHEN** an agent reads the managed guidance
- **THEN** it SHALL find instructions explaining that editing a shared Nx lib may require refreshing the indexes of downstream projects that depend on it
