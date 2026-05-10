## Context

This change fixes two user-facing regressions: setup guidance is skipped in non-interactive callers such as Warp, and Nx indexing is still losing valid lib coverage because inclusion is filtered too early at the project level instead of at the file level.

## Goals / Non-Goals

**Goals**
- Make setup guidance updates deterministic across terminals and automation.
- Keep managed guidance blocks safe and idempotent.
- Include all internal Nx libs in scope resolution by default.
- Preserve per-file code/docs filtering as the actual indexing gate.
- Surface missing Nx graph coverage instead of hiding it.

**Non-Goals**
- Add watchers or automatic background reindexing.
- Change file classification rules in `internal/content`.
- Add a new `doctor` command.
- Remove e2e exclusion entirely; it remains default behavior with an override.

## Decisions

### D1: Managed guidance updates are unconditional unless explicitly skipped
The setup flow SHALL append or replace the managed guidance block without checking terminal interactivity. The only opt-out is `--no-guidance`.

Rationale: the guidance block is fenced, reversible, and already versioned by markers. The previous TTY gate caused real setup failures in Warp and similar environments.

### D2: `setup.Run` uses an options struct
`internal/setup.Run` will accept `Options{Uninstall, SkipGuidance, AssumeYes}` instead of extra positional booleans.

Rationale: the setup command already has multiple mode switches, and an options struct keeps the call sites readable.

### D3: Nx inclusion is driven by graph relationships, not name heuristics
The project resolver will stop excluding libs by substring markers such as `docs`, `stories`, or `storybook`. The only default exclusion is `type: "e2e"`, derived from Nx metadata.

Rationale: substring filtering drops legitimate libraries. Nx graph metadata already tells us which projects are e2e.

### D4: `VECTOS_NX_INCLUDE_E2E` is the override
If the environment variable is set, e2e projects remain part of the resolved scope.

Rationale: the override must work for both CLI and MCP server usage, so an env var is the most consistent surface.

### D5: Scope warnings are returned in-band
`workspace.Scope` will gain a `Warnings []string` field. If Nx graph resolution fails, the resolver will still return a usable scope and populate a warning.

Rationale: callers can keep operating while still surfacing the degradation to the user.

### D6: Index output always prints detected internal libs
`vectos index` will print the roots detected for the selected Nx project, even without verbose mode.

Rationale: users need immediate visibility into what will be indexed, especially when diagnosing missing libs.

## Risks / Trade-offs

- Expanding all internal libs may index more files than before, but file-level filtering already limits docs/code behavior correctly.
- Removing name-based exclusion may include e2e-adjacent libs that are not typed as e2e. The trade-off is acceptable because the graph metadata is the authoritative source.
- Printing warnings in the main index flow adds a small amount of console noise, but it makes silent scope loss observable.
