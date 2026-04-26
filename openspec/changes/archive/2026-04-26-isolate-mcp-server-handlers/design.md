## Context

MCP is now important enough to deserve its own maintainable structure. Keeping all MCP setup and handlers inline inside the main CLI file will make the next protocol or ergonomics change harder to implement safely.

## Goals / Non-Goals

**Goals:**
- Move MCP server setup and handler registration out of `cmd/vectos/main.go`
- Separate search and indexing handler logic into clearer MCP-focused units
- Preserve the current tool surface and response behavior

**Non-Goals:**
- Renaming MCP tools or changing request shapes
- Replacing the current MCP SDK or transport model
- Redesigning runtime indexing/search behavior beyond what is needed for extraction

## Decisions

### Keep MCP setup together, but outside `main.go`

Server initialization, logging setup, and tool registration belong together, but not in the central CLI entrypoint file.

### Keep handler-specific formatting close to MCP handlers

Formatting and guidance helpers that exist specifically for MCP should stay near MCP handler code rather than be merged back into generic CLI presentation logic.

### Preserve protocol compatibility as a hard constraint

This change should be behavior-preserving from the perspective of agents already using `search_code` and `index_project`.

## Risks / Trade-offs

- Moving schemas and handlers can accidentally change MCP registration -> Mitigation: keep end-to-end MCP validation in scope
- Too much separation can make handler flow harder to follow -> Mitigation: split by MCP concern, not by tiny helper alone
