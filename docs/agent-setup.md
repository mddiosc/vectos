# Agent Setup

Vectos exposes MCP tools over stdio through:

```bash
vectos mcp
```

## Automatic Setup For Validated Clients

Validated setup targets in the current phase:

- `opencode`
- `claude`
- `codex`

Run:

```bash
vectos setup opencode
vectos setup claude
vectos setup codex
```

Each setup command creates or updates a Vectos MCP entry in the agent's user-wide config and may also manage a small global guidance block so the agent prefers `vectos_search_code` and `vectos_index_project` before broad file-search tools.

Configuration targets:

- `opencode` -> `~/.config/opencode/opencode.json` + `~/.config/opencode/AGENTS.md`
- `claude` -> `~/.claude.json` + `~/.claude/CLAUDE.md`
- `codex` -> `~/.codex/config.toml` + `~/.codex/AGENTS.md`

If the global guidance file for a target does not exist yet, the setup creates a managed Vectos guidance block. If it already exists, the setup asks before appending the managed Vectos section so unrelated user instructions are preserved.

Remove a configured integration:

```bash
vectos setup opencode --uninstall
vectos setup claude --uninstall
vectos setup codex --uninstall
```

This removes only the Vectos-managed MCP entry and the Vectos-managed guidance block for that agent. It does not delete unrelated user config.

## Manual MCP Setup For Other Clients

If your client supports MCP but is not one of the validated setup targets above, configure it manually by pointing it at:

```bash
vectos mcp
```

Use an absolute path to the `vectos` binary when possible.

Generic MCP command shape:

```json
{
  "command": "/absolute/path/to/vectos",
  "args": ["mcp"]
}
```

Different clients may store MCP server definitions in different JSON, TOML, or YAML structures, but the underlying command is the same.

Suggested MCP server name:

- `vectos`

Currently exposed MCP tools:

- `vectos_search_code`
- `vectos_index_project`

Recommended guidance for unsupported clients:

```text
When Vectos MCP tools are available for a project, prefer `vectos_search_code` before using `grep`, `find`, `glob`, or broad file reads.

If the project is not yet indexed or `vectos_search_code` returns no useful results, run `vectos_index_project` and retry `vectos_search_code`.

Use `grep`, `glob`, and direct file reads only as a fallback when Vectos has no useful results or when you need exact pattern matching.
```

## Unsupported Setup Targets

Current explicit non-validated target for this phase:

- `gemini`

Running `vectos setup <agent>` for a non-validated target currently fails with an explicit error instead of pretending support exists.

For unsupported clients, manual MCP configuration is the intended path.

See also: [CLI Usage](cli.md)

If setup fails or the client cannot launch Vectos, also see [Troubleshooting](troubleshooting.md).
