## Why

`vectos setup opencode` already wires the MCP entry, but OpenCode still needs separate guidance before the agent consistently prefers Vectos retrieval over generic file-search tools. Capturing that behavior in OpenSpec closes the gap between "MCP configured" and "Vectos used by default".

## What Changes

- Extend `agent-setup` so supported setup flows may also install or manage global agent guidance when that guidance is needed to make Vectos the preferred retrieval path.
- Define the OpenCode-specific behavior for adding a managed global guidance block that tells the agent to prefer `vectos_search_code` and `vectos_index_project` before broader file-search tools.
- Preserve existing user guidance by making the global guidance block optional, append-only, and safely updatable.

## Capabilities

### New Capabilities

### Modified Capabilities
- `agent-setup`: setup for supported agents now covers both MCP configuration and optional global guidance that biases the agent toward Vectos-first retrieval.

## Impact

- `internal/setup` setup flow and adapter behavior.
- OpenCode global configuration under `~/.config/opencode/AGENTS.md`.
- README/setup documentation for expected Vectos-first retrieval behavior.
