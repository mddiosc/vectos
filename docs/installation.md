# Installation

## Release Install

> Experimental/internal builds. Not a stable public release.
> Supported platforms: `darwin/arm64` and `linux/amd64` only.

Install the latest release:

```sh
curl -fsSL https://github.com/mddiosc/vectos/releases/latest/download/install.sh | sh
```

The installer detects your platform, downloads the correct binary, verifies the checksum, installs `vectos` into `~/.local/bin` by default, and updates the startup file for your current shell when that directory is not already in `PATH`.

Install to a custom directory:

```sh
curl -fsSL https://github.com/mddiosc/vectos/releases/latest/download/install.sh | DEST_DIR=/usr/local/bin sh
```

Verify:

```bash
vectos version
```

First-run note: on first use, the embedded provider downloads ONNX Runtime and model assets from the internet into `~/.vectos/models/`. Subsequent runs use the cached assets.

Next step: [Agent Setup](agent-setup.md)

## Install From Source

You need `git` and `go`.

If you do not have Go installed yet:

- https://go.dev/doc/install

On macOS with Homebrew:

```bash
brew install go
```

Clone and install:

```bash
git clone <YOUR_REPO_URL>
cd vectos
./scripts/install.sh
```

By default this installs `vectos` into `~/.local/bin`. To use a different directory:

```bash
DEST_DIR=/your/bin/dir ./scripts/install.sh
```

## Manual Install Without The Helper Script

Build:

```bash
go build -o vectos ./cmd/vectos
```

System-wide install:

```bash
sudo install -m 0755 vectos /usr/local/bin/vectos
```

User-local install:

```bash
mkdir -p ~/.local/bin
install -m 0755 vectos ~/.local/bin/vectos
```

If you use the user-local install, make sure `~/.local/bin` is in your `PATH`.

## Uninstall

Remove the installed binary:

```sh
curl -fsSL https://github.com/mddiosc/vectos/releases/latest/download/install.sh | sh -s -- --uninstall
```

If you installed to a custom directory:

```sh
curl -fsSL https://github.com/mddiosc/vectos/releases/latest/download/install.sh | DEST_DIR=/usr/local/bin sh -s -- --uninstall
```

Or if you have the repository cloned:

```sh
./scripts/install.sh --uninstall
```

This removes the `vectos` binary and also removes the Vectos-managed `PATH` block from your shell startup file if one was installed by the script.

Manual cleanup targets:

| Path | Contents |
|---|---|
| `~/.vectos/` | Cached models and index databases |
| `~/.claude.json` | Claude Code user MCP config (edit, not delete) |
| `~/.claude/CLAUDE.md` | Claude Code global guidance (edit, not delete) |
| `~/.codex/config.toml` | Codex MCP config (edit, not delete) |
| `~/.codex/AGENTS.md` | Codex global guidance (edit, not delete) |
| `~/.config/opencode/opencode.json` | OpenCode MCP config (edit, not delete) |
| `~/.config/opencode/AGENTS.md` | OpenCode global guidance (edit, not delete) |

See also: [Troubleshooting](troubleshooting.md)
