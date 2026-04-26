## Why

Vectos already exposes MCP search and indexing, but the current tool responses are still closer to raw plumbing than to a polished agent-facing retrieval experience. Phase 2 should improve how agents consume search results so Vectos is easier to use naturally inside real MCP workflows.

## What Changes

- Improve MCP search responses with more useful result metadata and clearer summaries.
- Expose enough context for agents to understand why a result is relevant before opening files.
- Keep indexing and search workflows aligned so agents can recover gracefully when a project is missing or stale.
- Preserve the standalone-first tool surface while making agent usage more ergonomic.

## Capabilities

### New Capabilities
- None.

### Modified Capabilities
- `mcp-interface`: MCP search and indexing responses will provide richer result summaries and clearer agent-facing ergonomics.

## Impact

- Affected code: MCP tool handlers, search result formatting, indexing summaries, and possibly guidance documentation
- Affected behavior: agents receive clearer search results, richer metadata, and more actionable indexing feedback
- Dependencies: may require exposing more internal result metadata, but should avoid widening the MCP surface unnecessarily in the first pass
