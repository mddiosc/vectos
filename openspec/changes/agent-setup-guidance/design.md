## Context

The existing `agent-setup` capability guarantees that `vectos setup <agent>` can create a valid MCP entry for a supported client. That is necessary but not sufficient for token-efficient usage in OpenCode, because the agent also needs persistent guidance telling it to prefer Vectos retrieval before falling back to `grep`, `find`, `glob`, or broad file reads.

OpenCode already supports a global `AGENTS.md` file in `~/.config/opencode/`. That makes it the natural place to add guidance without coupling the behavior to a specific repository.

## Goals / Non-Goals

**Goals:**
- Extend setup semantics so a supported agent setup may manage both integration config and guidance config.
- Make the OpenCode setup flow able to install Vectos-first guidance globally.
- Keep the guidance idempotent and safe to re-run.
- Avoid overwriting unrelated global instructions that the user already maintains.

**Non-Goals:**
- Designing a generic prompt-management framework for all agents.
- Requiring guidance support for every agent target in the same phase.
- Replacing repository-local `AGENTS.md` rules.

## Decisions

- OpenCode guidance will be modeled as an evolution of `agent-setup`, not as a separate capability.
  Rationale: the behavior is part of what `vectos setup opencode` now does, and it changes the observable setup contract rather than introducing a standalone subsystem.
- The guidance will be stored as a managed block inside `~/.config/opencode/AGENTS.md`.
  Rationale: this lets Vectos update only its own section while preserving user-authored instructions around it.
- Setup will auto-create the managed block when no global `AGENTS.md` exists, but it will ask before appending to an existing file.
  Rationale: first-time setup should be smooth, while existing user configuration should never be silently altered.
- The guidance will explicitly instruct OpenCode to prefer `vectos_search_code`, then `vectos_index_project` when needed, and only then fall back to generic filesystem tools.
  Rationale: that matches the token-efficiency goal that motivated the change.

## Risks / Trade-offs

- Existing global instructions may conflict with the new guidance block. → Keep Vectos guidance isolated in a managed section so conflicts stay visible and editable.
- Non-interactive setup flows may not be able to confirm appending to an existing global file. → Allow setup to skip guidance changes safely instead of forcing or corrupting the file.
- This behavior is currently OpenCode-specific. → Keep the requirement scoped to supported agents that expose a stable global guidance mechanism.

## Migration Plan

1. Extend the `agent-setup` spec with the new guidance requirement.
2. Keep existing MCP setup behavior unchanged.
3. Add OpenCode-specific guidance management on top.
4. Document the new behavior and fallback expectations.

## Open Questions

- Whether future non-interactive flows should expose explicit flags like `--yes` or `--skip-guidance` is still open and can be handled in a follow-on change if needed.
