## Why

Vectos already returns useful results, but phase 2 needs a way to measure whether retrieval quality is actually improving across real repositories and representative queries. Without a built-in evaluation workflow, ranking and chunking changes will continue to rely too heavily on intuition and one-off spot checks.

## What Changes

- Add a retrieval evaluation capability that can run a repeatable set of benchmark queries against an indexed project.
- Define a lightweight benchmark format for storing expected useful files or chunks for real-world queries.
- Report simple retrieval quality signals such as whether relevant results appear in the top ranked positions.
- Keep the evaluation flow standalone-first so it can be run from the CLI without requiring external orchestration.

## Capabilities

### New Capabilities
- `retrieval-evaluation`: Run repeatable retrieval benchmarks against an indexed project and report quality metrics for representative queries.

### Modified Capabilities
- None.

## Impact

- Affected code: CLI surface, search execution path, benchmark parsing, and reporting output
- Affected behavior: Vectos gains a built-in way to validate retrieval quality before and after ranking or indexing changes
- Dependencies: may introduce a small benchmark fixture format, but should avoid heavy external evaluation frameworks
