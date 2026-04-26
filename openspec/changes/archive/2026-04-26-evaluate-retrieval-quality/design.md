## Context

Vectos phase 1 proved that semantic retrieval is already useful, but phase 2 needs a feedback loop that is more disciplined than manual inspection. Evaluation must stay cheap, standalone, and close to the real workflows Vectos is meant to support.

## Goals / Non-Goals

**Goals:**
- Provide a repeatable evaluation workflow for representative retrieval queries
- Measure whether useful results appear near the top of the ranked output
- Keep benchmark authoring simple enough to maintain inside the repository
- Reuse the existing search engine instead of building a separate evaluation stack

**Non-Goals:**
- Building a research-grade IR evaluation framework
- Automating benchmark generation from arbitrary repositories
- Replacing manual qualitative inspection for every search issue

## Decisions

### Use a repository-local benchmark file format

Vectos should store evaluation queries in a simple text-based fixture format that can live in the repo, be versioned, and be expanded over time. A lightweight JSON or YAML structure is preferable to an external dataset format because the main goal is maintainability, not interoperability.

### Evaluate useful hits, not only raw scores

The main product question is whether the right code appears early enough for an agent to use it with minimal extra exploration. The evaluation flow should therefore report positional usefulness metrics such as top-3 and top-5 hit success rather than only raw similarity scores.

### Reuse the normal retrieval pipeline

The benchmark command should call the same search path used by real CLI and MCP retrieval so the measured behavior matches real-world usage. A separate evaluation-only ranking path would make results less trustworthy.

### Keep output readable first, machine-readable second

The first implementation should optimize for human iteration: query name, expected files, top results, and pass/fail metrics. Structured output can be added if needed, but the first version should make it easy to compare runs during normal development.

## Risks / Trade-offs

- Small benchmark sets may overfit to a few favorite repos -> Mitigation: require representative queries across at least a few real tasks
- Evaluation may drift from real usefulness if expected targets are too rigid -> Mitigation: allow multiple valid expected files or chunks per query
- A too-heavy format could discourage benchmark upkeep -> Mitigation: prefer a compact fixture schema with minimal required fields
