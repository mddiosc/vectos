## Why

Even after CLI wiring is split, `cmd/vectos/main.go` still carries large runtime functions for indexing, searching, status inspection, path classification, and workspace/storage helpers. These routines are practical but too concentrated, making future changes harder to review and test.

## What Changes

- Extract runtime command implementations such as indexing, search, and status into dedicated service-oriented files under `cmd/vectos/`.
- Group related helper functions with the command flows they support.
- Preserve current runtime behavior, outputs, and validation paths.

## Capabilities

### New Capabilities
- `runtime-command-services`: define maintainable separation between CLI wiring and command execution helpers.

### Modified Capabilities
- None.

## Impact

- Affected code: runtime command functions and helper routines currently concentrated in `cmd/vectos/main.go`
- Affected behavior: no intended user-visible behavior changes
- Dependencies: should preserve current indexing, search, status, and workspace-resolution behavior
