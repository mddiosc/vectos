## Why

Vectos already has `status` output and troubleshooting docs, but users still have to piece together install health, provider readiness, and index consistency by hand when something is broken. A `doctor` command gives one direct diagnostic entry point for the most common local failure modes.

## What Changes

- Add a new `vectos doctor` command.
- Reuse existing status and provider checks where possible, and add install/runtime diagnostics for the current environment.
- Report actionable pass/warn/fail results for binary, config, provider, and index health.
- Include `doctor` in CLI help and troubleshooting guidance.

## Capabilities

- `runtime-diagnostics`: Read-only command that inspects installation, runtime readiness, embedding provider health, and index consistency.
- `cli-discoverability`: The command is visible in global and subcommand help.

## Impact

- New CLI wiring in `cmd/vectos/cli_dispatch.go` and help text in `cmd/vectos/cli_help.go`.
- Shared diagnostic logic for provider and index checks.
- Tests for command output, exit behavior, and help coverage.
- Troubleshooting docs can point users to `vectos doctor` as the first-line check.
