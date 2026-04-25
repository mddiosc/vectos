# Troubleshooting

## `vectos: command not found`

Possible causes:

- the install directory is not in your `PATH`
- your shell session has not been restarted after installation
- you installed to a custom `DEST_DIR`

Checks:

```bash
echo "$PATH"
ls ~/.local/bin/vectos
```

If needed, restart your shell or source the updated startup file.

## Installer says the platform is unsupported

Current release assets only support:

- `darwin/arm64`
- `linux/amd64`

For other platforms, use a source install:

```bash
./scripts/install.sh --from-source
```

## Permission denied during install

This usually means the target install directory is not writable by the current user.

Common fixes:

- install into the default user-local location
- choose a writable custom `DEST_DIR`
- use a system path only when you intend to manage permissions explicitly

Example user-local install:

```bash
DEST_DIR="$HOME/.local/bin" ./scripts/install.sh
```

## `vectos setup <agent>` fails

Checks:

- verify that the agent target is currently validated
- run `vectos help setup`
- confirm that `vectos` is installed globally and not only available as `./vectos`

Currently validated setup targets:

- `opencode`
- `claude`
- `codex`

Unsupported or non-validated clients should use manual MCP setup instead. See [Agent Setup](agent-setup.md).

## Manual MCP client cannot start Vectos

Use an absolute binary path in the MCP server definition when possible.

Recommended command shape:

```json
{
  "command": "/absolute/path/to/vectos",
  "args": ["mcp"]
}
```

Checks:

- the binary path exists
- the file is executable
- the client uses stdio MCP

Quick shell verification:

```bash
which vectos
vectos mcp
```

## Search quality looks stale or incorrect

You may need to reindex.

Common reasons:

- the embedding provider changed
- the embedding model changed
- the project content changed significantly
- the current index metadata no longer matches the active provider

Checks:

```bash
vectos status
```

Reindex:

```bash
vectos index .
```

## MCP tools return poor results on a fresh project

The project may not be indexed yet.

Index first:

```bash
vectos index .
```

Or, from an MCP client, call:

- `vectos_index_project`

Then retry:

- `vectos_search_code`

## First run downloads take time

This is expected for the embedded provider.

On first use, Vectos downloads:

- ONNX Runtime
- model files
- tokenizer assets

These are cached under `~/.vectos/models/` and reused on later runs.

## Uninstall removed the binary but left files behind

This is expected.

`--uninstall` removes:

- the installed `vectos` binary
- the Vectos-managed `PATH` block, if one was added by the installer

It does not automatically delete:

- `~/.vectos/`
- agent client configuration files
- managed guidance files outside the binary uninstall path

See [Installation](installation.md) for manual cleanup targets.
