## Context

The Vectos CLI entry point (`cmd/vectos/main.go`) was written iteratively with Spanish text in comments, usage strings, error messages, and success output. The rest of the codebase is already in English. The public release makes this inconsistency visible.

The CLI also lacks a proper help system. Today, help text is printed only when arguments are missing, with no `--help` flag support, no per-subcommand help, and no `help` subcommand.

The installer (`scripts/install.sh`) has no uninstall path. Users who want to remove Vectos have no documented procedure.

## Goals / Non-Goals

**Goals:**
- English-only CLI output and code comments in the main CLI layer.
- Centralized help text printed consistently for `help`, `--help`, `-h`, and per-subcommand `--help`.
- `--uninstall` in `scripts/install.sh` that removes the binary and shows manual purge guidance.

**Non-Goals:**
- `vectos uninstall` as a CLI subcommand in this phase.
- Automatic purge of `~/.vectos/` data, models, or agent config files.
- Internationalization or multi-language support.

## Decisions

- Help is implemented as a single central `printHelp()` function that prints global usage, and per-subcommand `printSubcommandHelp()` variants — avoids duplicating format strings.
- `--help` and `-h` are handled before flag parsing so they always work regardless of other argument order.
- `help <subcommand>` also works (e.g. `vectos help index`) to show subcommand-specific help.
- `--uninstall` in the script removes only the binary at `$DEST_DIR/vectos`. It does not remove data dirs automatically: instead it prints a "Manual cleanup" block listing the paths the user may want to remove.
- Uninstall detects `DEST_DIR` the same way the installer does (env var or `~/.local/bin` default), ensuring consistency.

## Risks / Trade-offs

- Per-subcommand help requires maintaining help strings in sync with actual flags — mitigated by centralizing them.
- `--uninstall` only removes from `DEST_DIR`; if the user installed to a different dir and does not pass `DEST_DIR=...`, it will look in the default. → Documented clearly in README and installer output.

## Open Questions

- Whether to add `vectos uninstall` as a CLI subcommand in a future phase (deferred).
