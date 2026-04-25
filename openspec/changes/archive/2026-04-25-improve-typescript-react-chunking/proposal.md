## Why

Vectos already delivers useful semantic retrieval, but its TypeScript and React chunking still relies heavily on heuristics and line windows. That limits result quality for the exact frontend repositories Vectos is meant to help with, especially when users ask for hooks, components, tests, or composition boundaries rather than exact symbol names.

## What Changes

- Improve TypeScript and React chunk extraction so components, hooks, exported functions, and test blocks are chunked as meaningful units instead of mostly line-based fragments.
- Preserve existing low-cost chunking behavior as a fallback when structural extraction cannot be derived safely.
- Improve semantic enrichment for TypeScript/React chunks so retrieval favors actionable app code over ambiguous fragments.

## Capabilities

### New Capabilities
- None.

### Modified Capabilities
- `code-indexing`: TypeScript and React files will be chunked with better structural boundaries.
- `semantic-search`: richer chunk boundaries and metadata will improve semantic retrieval quality for frontend projects.

## Impact

- Affected code: `internal/indexer/chunker.go` and related indexing logic
- Affected behavior: chunk boundaries, embedding input quality, and retrieval precision for TS/TSX/JSX-heavy projects
- Dependencies: may introduce a parser or a more structured extraction strategy if the current heuristics are insufficient
