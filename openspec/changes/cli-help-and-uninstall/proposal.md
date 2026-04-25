## Why

Vectos CLI output and code comments are still partially in Spanish, which is inconsistent with the project's target audience and public release posture. Additionally, the CLI lacks a real help system and the installer has no uninstall path, leaving users with no documented way to remove the tool cleanly.

## What Changes

- Translate all user-visible CLI text and code comments in the main CLI layer to English.
- Add a consistent help system so users can discover commands via `vectos help`, `vectos --help`, and per-subcommand `--help`.
- Extend `scripts/install.sh` with `--uninstall` to remove the installed binary and show manual purge instructions for data and config directories.

## Capabilities

### Modified Capabilities
- `distribution-packaging`: installer now supports `--uninstall` in addition to install.

### New Capabilities
- `cli-help`: consistent help output for all subcommands reachable via `help`, `--help`, `-h`.

## Impact

- `cmd/vectos/main.go` — English-only output and centralized help.
- `scripts/install.sh` — `--uninstall` support.
- `README.md` — updated uninstall instructions.
