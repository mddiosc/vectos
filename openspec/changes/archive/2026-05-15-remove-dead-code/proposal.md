## Why

The Vectos codebase has accumulated dead code: a hand-rolled MCP protocol implementation superseded by `go-sdk`, stub functions like `inferGoPurpose`, and likely more orphaned artifacts. This dead code increases maintenance burden, binary size, and cognitive overhead without providing any value.

## What Changes

- Remove `internal/mcp/` directory (legacy MCP server replaced by `cmd/vectos/` using `go-sdk`)
- Remove the `inferGoPurpose` function (non-functional placeholder stub)
- Scan for and remove any other unreferenced exported functions, unused types, and orphaned test helpers
- Verify no breakage with `go vet` and `staticcheck` after each removal

## Capabilities

### New Capabilities
<!-- None — this is purely a removal/cleanup change -->

### Modified Capabilities
<!-- None — no spec-level behavior changes; existing capabilities are unaffected -->

## Impact

- **`internal/mcp/`** — entire directory deleted; no current imports exist per search
- **`inferGoPurpose`** — function and any related test code removed
- **Any additional dead code** — identified via static analysis, removed if verified unreferenced
- **Build** — `go vet` and `staticcheck` must pass after all removals
- **No behavioral changes** — existing functionality preserved; only dead code removed
