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

Each setup command creates or updates a Vectos MCP entry in the agent's user-wide config, manages a global guidance block, and installs a Vectos skill (where supported) so the agent prefers Vectos search tools before broad file-search tools.

Flags:

```
--no-guidance    Skip global guidance updates (configure MCP, install skill and plugin)
--yes, -y        Answer yes to all prompts (non-interactive mode)
```

Configuration targets:

- `opencode` -> `~/.config/opencode/opencode.json` + `~/.config/opencode/AGENTS.md` + `~/.config/opencode/plugins/vectos.ts` + `~/.agents/skills/vectos/SKILL.md`
- `claude` -> `~/.claude.json` + `~/.claude/CLAUDE.md`
- `codex` -> `~/.codex/config.toml` + `~/.codex/AGENTS.md` + `~/.codex/skills/vectos/SKILL.md`

If the global guidance file for a target does not exist yet, the setup creates a managed Vectos guidance block. If it already exists, the setup appends or replaces the managed Vectos block automatically. Guidance updates are unconditional — use `--no-guidance` to skip them.

The managed guidance currently teaches the agent to:

- prefer `vectos_search_code` / `search_code` for source-code lookups
- prefer `vectos_search_docs` / `search_docs` for README files, ADRs, API docs, and other documentation
- run `vectos_index_project` / `index_project` when the project is not indexed yet
- use `docs: true` when it needs a dedicated documentation index
- use incremental reindex with `changed` file paths after editing files, instead of defaulting to a full reindex
- instruct specialist sub-agents to use Vectos search tools before `grep`/`glob` when delegating

For OpenCode and Codex, a companion Vectos skill is also installed so agents can load detailed usage patterns, troubleshooting, and delegation guidance on demand.

Remove a configured integration:

```bash
vectos setup opencode --uninstall
vectos setup claude --uninstall
vectos setup codex --uninstall
```

To reinstall without touching the guidance block:

```bash
vectos setup opencode --no-guidance
```

Setting `--no-guidance` skips the AGENTS.md/CLAUDE.md guidance block update but still installs the Vectos skill (where supported), the MCP entry, and any plugins. Use this to reconfigure MCP (for example after a binary upgrade) without altering your guidance block.

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
- `vectos_search_docs`
- `vectos_index_project`
- `vectos_list_projects`

Recommended guidance for unsupported clients:

```text
When Vectos MCP tools are available for a project, prefer Vectos search tools before using `grep`, `find`, `glob`, or broad file reads.

For source code and implementation lookups, use `vectos_search_code`.

For README files, API docs, ADRs, and other documentation, use `vectos_search_docs`.

If the project is not yet indexed or results are not useful, run `vectos_index_project` and retry. When you need documentation retrieval, index docs separately.

If you create, move, or edit files while working, prefer an incremental refresh with `changed` paths before retrying search. Use a full reindex only when the affected scope is broad or uncertain.

Use `grep`, `glob`, and direct file reads only as a fallback when Vectos has no useful results or when you need exact pattern matching.

When delegating to specialist agents that perform code search, explicitly instruct them to use Vectos search tools before `grep`/`glob`. The main agent is the enforcement point. If a sub-agent returns without using Vectos, remind it in the next delegation.
```

When session memory tools such as Engram are also available, use them for prior decisions and durable learnings first, then use Vectos for current code retrieval. Vectos does not require those tools to function.

## Unsupported Setup Targets

Current explicit non-validated target for this phase:

- `gemini`

Running `vectos setup <agent>` for a non-validated target currently fails with an explicit error instead of pretending support exists.

For unsupported clients, manual MCP configuration is the intended path.

See also: [CLI Usage](cli.md)

See also: [Optional Engram Synergy](engram-synergy.md)

If setup fails or the client cannot launch Vectos, also see [Troubleshooting](troubleshooting.md).
