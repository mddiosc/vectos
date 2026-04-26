## Why

`cmd/vectos/main.go` currently mixes global help text, subcommand flag configuration, argument normalization, command dispatch, and multiple command implementations in a single oversized file. This increases branch count, makes the CLI harder to reason about, and raises the cost of every future command change.

## What Changes

- Split command help, flag wiring, and dispatch structure into smaller CLI-focused files.
- Keep `main.go` as a thin entrypoint that initializes shared configuration and delegates command execution.
- Preserve all current CLI behavior, flags, help text, and command names.

## Capabilities

### New Capabilities
- `command-structure`: define maintainable separation between CLI entrypoint wiring and subcommand behavior.

### Modified Capabilities
- None.

## Impact

- Affected code: `cmd/vectos/main.go` and new CLI-focused files under `cmd/vectos/`
- Affected behavior: no intended user-visible behavior changes
- Dependencies: should stay compatible with current command help, flag parsing, and setup flow
