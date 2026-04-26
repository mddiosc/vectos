## Context

The runtime behavior of Vectos is already valuable, but the execution functions are harder to isolate and test while they remain bundled inside one large file. Splitting them now reduces the cost of later feature work and makes the command layer easier to maintain.

## Goals / Non-Goals

**Goals:**
- Separate runtime command implementations from entrypoint wiring
- Group helper functions by operational concern
- Improve readability without changing command semantics

**Non-Goals:**
- Rewriting the indexing or search architecture itself
- Changing output format as part of this refactor
- Merging runtime logic into unrelated internal packages prematurely

## Decisions

### Split by operational concern

Runtime concerns should be grouped into files such as indexing, search/status, and workspace/path helpers rather than staying inside one monolithic command file.

### Keep command runtime logic in `cmd/vectos/` for now

This refactor should stop short of a larger package re-architecture. The main objective is maintainability and reviewability, not a deeper architectural move across package boundaries.

### Preserve shared helpers where they already pay for themselves

Existing helpers such as storage/scope resolution or search execution should stay shared, but be colocated more clearly with the commands that rely on them.

## Risks / Trade-offs

- Splitting helpers too aggressively may scatter related logic -> Mitigation: extract by cohesive runtime domain
- Some duplicated wiring may remain after this step -> Mitigation: optimize for clearer structure first, then deduplicate only where obvious
