## Why

The MCP layer has grown from a simple integration point into a meaningful product surface. Even after extracting output formatting, `runMCP` in `cmd/vectos/main.go` still embeds server startup, input schemas, handler registration, indexing/search behavior, and protocol logging in one place.

## What Changes

- Isolate MCP server construction and tool registration into dedicated files.
- Separate MCP search and indexing handlers from generic command runtime code.
- Preserve current MCP tool names, request shapes, and response behavior.

## Capabilities

### New Capabilities
- `mcp-server-structure`: define maintainable separation between MCP server setup and handler behavior.

### Modified Capabilities
- None.

## Impact

- Affected code: `runMCP`, MCP tool schemas, handler registration, and MCP-oriented helpers in `cmd/vectos/`
- Affected behavior: no intended protocol or tool-surface changes
- Dependencies: should preserve the current `search_code` and `index_project` behavior and tool discovery
