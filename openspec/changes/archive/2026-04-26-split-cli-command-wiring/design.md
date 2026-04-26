## Context

The current CLI entrypoint has grown into a catch-all file. The highest-value maintainability improvement is to separate command wiring concerns from command execution concerns before phase 3 adds more surface area.

## Goals / Non-Goals

**Goals:**
- Reduce `main.go` size and branching pressure
- Separate help/flag/dispatch code from runtime command implementations
- Keep the refactor behavior-preserving and easy to review

**Non-Goals:**
- Renaming commands or changing CLI UX
- Redesigning the command model from scratch
- Refactoring internal indexing/search/MCP logic in this change

## Decisions

### Keep `main.go` as an entrypoint only

`main.go` should own startup concerns such as loading shared configuration and invoking a dispatch function, but not the full per-command control flow.

### Extract command help and flag wiring into CLI-specific files

Help text, `flag.FlagSet` creation, and argument normalization are related concerns and should move together rather than remaining mixed with runtime command logic.

### Preserve current command behavior exactly

This change is about maintainability, not feature changes. Flag names, help output, and command dispatch behavior should stay equivalent.

## Risks / Trade-offs

- Moving flag wiring can accidentally change help or parse order -> Mitigation: keep tests or spot checks around current help behavior
- Over-extraction could create too many tiny files -> Mitigation: split by concern, not by one-function-per-file
