## Why

`vectos setup opencode` currently skips updating an existing `AGENTS.md` when the caller is detected as non-interactive, which breaks setup in terminals such as Warp even though the guidance block is safe to append. Separately, Nx indexing is still dropping valid internal libs because project-level exclusions are too broad; the indexer should include all internal libs and let the per-file filters decide what is actually indexed.

## What Changes

- Make managed setup guidance updates idempotent and non-interactive by default, while still allowing an explicit opt-out for guidance installation.
- Add setup flags to control guidance handling without relying on terminal interactivity.
- Stop excluding Nx libs by substring heuristics like `docs`, `stories`, or `storybook`.
- Keep excluding `type: "e2e"` projects by default, with an env var override for cases where e2e projects should be indexed.
- Surface Nx graph resolution warnings and print the internal libs that were detected during indexing.

## Capabilities

### New Capabilities

- None.

### Modified Capabilities

- `agent-setup`: managed guidance updates become non-interactive by default, support an explicit no-guidance opt-out, and the guidance content must describe the current retrieval and reindex workflow.
- `workspace-awareness`: Nx project scoping must include all internal lib roots by default, preserve per-file docs/code filtering, and expose warnings when Nx graph resolution is incomplete.

## Impact

- `internal/setup/*` — setup flow and managed guidance behavior.
- `cmd/vectos/*` — CLI flags, setup dispatch, and index output.
- `internal/workspace/*` — Nx project expansion, exclusions, and warnings.
- `openspec/specs/*` — update the `agent-setup` and `workspace-awareness` contracts.
