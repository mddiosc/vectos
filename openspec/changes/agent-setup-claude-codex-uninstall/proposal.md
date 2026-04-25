## Why

Vectos currently validates only `opencode` for setup, even though Codex and Claude Code both support MCP configuration and already have established config formats. This leaves two major agent clients unsupported and forces users to configure them by hand.

Vectos also has no uninstall path for agent integrations. Once MCP entries or global guidance are installed, users have no built-in way to remove them cleanly.

## What Changes

- Extend `agent-setup` to validate and support `claude` and `codex` in addition to `opencode`.
- Add uninstall support to agent setup so users can remove the Vectos MCP entry and managed global guidance for supported agents.
- Update CLI help and docs to reflect the expanded supported-agent matrix and the uninstall workflow.

## Capabilities

### Modified Capabilities
- `agent-setup`: validated support expands to `opencode`, `claude`, and `codex`; setup can now also uninstall managed integration state for those agents.
- `cli-help`: help text must document the expanded setup command behavior and supported agents.

## Impact

- `internal/setup/` adapters and setup orchestration
- CLI help text in `cmd/vectos/main.go`
- README setup and uninstall documentation
