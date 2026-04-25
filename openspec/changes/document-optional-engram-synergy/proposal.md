## Why

Vectos should stand on its own as a code-context product, but its workflow becomes more powerful when paired with Engram session memory. That relationship needs to be documented and formalized as optional synergy, not product coupling.

## What Changes

- Document a recommended workflow where agents use Engram memory and Vectos code retrieval together when both are available.
- Clarify that Vectos remains fully useful standalone and does not depend on Engram.
- Define the MCP and documentation expectations that make the combined workflow easier for agents to follow.

## Capabilities

### New Capabilities
- None.

### Modified Capabilities
- `engram-integration`: the current lightweight interoperability requirement will be expanded into an explicit optional workflow contract.
- `mcp-interface`: documentation/guidance may clarify how Vectos tools should be used in mixed-memory agent sessions.

## Impact

- Affected code: likely docs and guidance first, with optional future MCP/helper changes
- Affected behavior: clearer agent workflows when both systems are installed
- Dependencies: none required for standalone Vectos behavior
