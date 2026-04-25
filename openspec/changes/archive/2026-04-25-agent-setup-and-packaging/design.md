## Context

Vectos currently supports automated setup for OpenCode only and is primarily run as a project-local binary. That is enough for development, but not for product-style adoption.

## Goals / Non-Goals

**Goals:**
- Refactor setup into reusable per-agent adapters.
- Add setup flows only for agents whose config targets can be implemented confidently in this phase.
- Make Vectos installable as a real CLI command.
- Document and support a source-based global installation path.

**Non-Goals:**
- Supporting every agent client in the ecosystem immediately.
- Building a full cross-platform release and publishing pipeline in one step.
- Claiming setup support for agent config formats that have not been validated in this phase.
- Shipping Homebrew/release automation as a required part of this phase.

## Decisions

- Setup support will use per-agent adapters behind a common interface.
- The first implementation phase will only wire agents whose config files and MCP-style integration points are validated.
- Packaging will target a global binary UX via source-based installation first.
- Release/Homebrew metadata may exist as preparatory work, but they are not the supported installation contract for this phase.

## Risks / Trade-offs

- Agent config formats may drift independently over time.
- Packaging introduces release-management overhead.
- Release/Homebrew support is valuable, but should not complicate the current source-based install story.
- Overpromising agent support is worse than shipping a smaller but reliable supported-agent set.

## Migration Plan

1. Refactor setup logic into reusable per-agent adapters.
2. Add setup targets for validated supported agents.
3. Define and document the source-based global install story.
4. Treat release/Homebrew packaging as future follow-on work if needed.
