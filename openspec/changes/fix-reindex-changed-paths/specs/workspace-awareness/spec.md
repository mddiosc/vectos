## ADDED Requirements

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
