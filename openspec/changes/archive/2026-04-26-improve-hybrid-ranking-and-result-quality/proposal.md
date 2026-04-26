## Why

Vectos phase 1 improved chunk quality, but retrieval still relies too heavily on a single ranking mode and can surface redundant or weakly actionable results. Phase 2 needs a more reliable result-quality layer so agents reach the right code with fewer exploratory reads.

## What Changes

- Add a hybrid ranking strategy that combines semantic retrieval with text-aware signals when that improves result quality.
- Reduce redundant or near-duplicate results so the top results cover more useful ground.
- Improve result ranking with lightweight structural or symbolic signals that favor actionable code over ambiguous matches.
- Preserve safe fallback behavior when semantic or hybrid ranking cannot execute.

## Capabilities

### New Capabilities
- None.

### Modified Capabilities
- `semantic-search`: retrieval behavior will support hybrid ranking, stronger result-quality heuristics, and reduced redundancy in top results.

## Impact

- Affected code: search ranking logic, result post-processing, chunk metadata usage, and CLI/MCP result presentation
- Affected behavior: better top-result precision, fewer redundant hits, and more consistent ranking for mixed semantic and exact-match queries
- Dependencies: may require additional result metadata or scoring stages, but should avoid heavyweight search infrastructure
