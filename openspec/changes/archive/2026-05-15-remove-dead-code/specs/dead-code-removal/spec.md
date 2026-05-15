## REMOVED Requirements

### Requirement: Legacy MCP server (internal/mcp/)
**Reason**: Replaced by `go-sdk` MCP implementation in `cmd/vectos/`. The hand-rolled MCP protocol in `internal/mcp/server.go` is no longer used.
**Migration**: No migration needed — no active code references this package.

### Requirement: inferGoPurpose function
**Reason**: Non-functional placeholder stub with no real implementation. Dead code.
**Migration**: No migration needed — the function performs no useful work.

## ADDED Requirements

<!-- No new capabilities — this is a pure removal/cleanup change -->
