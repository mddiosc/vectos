## Context

The current setup layer uses a small adapter abstraction and only validates `opencode`. The implementation already proved the basic pattern: create or update an agent-specific config file, add a Vectos MCP entry, and optionally manage a small guidance block in the agent's global instructions file.

Two additional agents are now good targets for the same pattern:

- **Codex CLI** stores MCP configuration in `~/.codex/config.toml` under `[mcp_servers.<name>]`.
- **Claude Code** stores user-scoped MCP configuration in `~/.claude.json` and supports a project/user `.mcp.json` format with `mcpServers` entries. For a user-wide setup flow, `~/.claude.json` is the right target for a `user`-scope server.

Both clients also have global instruction files that can host a small managed Vectos guidance block:

- Codex: `~/.codex/AGENTS.md`
- Claude Code: `~/.claude/CLAUDE.md`

## Goals / Non-Goals

**Goals:**
- Validate and support `claude` and `codex` in the same setup UX as `opencode`.
- Add uninstall support for all validated agents.
- Keep config updates idempotent and safely scoped to Vectos-managed entries/blocks.
- Update help and docs so the new behavior is discoverable.

**Non-Goals:**
- Supporting `gemini` in this phase.
- Purging all agent data or Vectos caches during uninstall.
- Replacing agent-native setup CLIs (`claude mcp add`, etc.) with a generalized wrapper beyond the minimal config updates needed here.

## Decisions

- Setup uninstall will be exposed as `vectos setup <agent> --uninstall`.
  Rationale: uninstall is the inverse lifecycle of setup and fits naturally under the same command rather than creating a new top-level command.
- `claude` setup will write a user-scoped `mcpServers.vectos` entry into `~/.claude.json`.
  Rationale: Anthropic documents user-scope MCP servers in `~/.claude.json`, which avoids requiring a project-local `.mcp.json`.
- `codex` setup will manage `[mcp_servers.vectos]` in `~/.codex/config.toml`.
  Rationale: this is the existing user-wide Codex configuration file and already contains other MCP servers.
- Global guidance for Codex and Claude will be managed as delimited blocks in their existing instruction files.
  Rationale: it preserves unrelated user content and allows clean uninstall of just the Vectos-managed section.
- Uninstall is idempotent: if the MCP entry or guidance block does not exist, the command should still complete without destructive side effects.
  Rationale: removal should be safe to rerun.

## Risks / Trade-offs

- TOML mutation for Codex introduces a new config format to manage. -> Use a TOML library rather than hand-editing strings.
- Claude Code config shape is broad and may contain unrelated project state. -> Touch only top-level `mcpServers.vectos` and leave all other fields intact.
- Guidance files may already contain user instructions. -> Use managed block markers and update/remove only that block.

## Open Questions

- Whether `gemini` should follow the same adapter pattern later is deferred to a future change.
