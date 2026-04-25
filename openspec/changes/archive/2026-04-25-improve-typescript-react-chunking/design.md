## Context

Vectos must remain useful as a standalone product. Improvements to chunking should raise retrieval quality without requiring Engram or any external orchestration layer.

## Goals

- Extract higher-signal chunks from TypeScript and React code
- Keep fallback behavior safe when syntax is unusual or unsupported
- Improve retrieval quality without exploding chunk counts

## Non-Goals

- Full language-server-grade semantic indexing
- Framework-specific static analysis for every frontend stack
- Replacing all line-based chunking for every supported language

## Approach

Use a layered chunking strategy for TypeScript and React files:

1. Prefer structural chunk boundaries for:
   - exported functions
   - React components
   - hooks
   - classes
   - test blocks such as `it()` and `test()`
2. Preserve a lightweight prelude chunk for imports and top-level declarations when useful.
3. Fall back to the current chunk-by-lines logic when structural extraction cannot be derived safely.

## Design Decisions

### Structural boundaries should be language-focused, not framework-exhaustive

The first version should recognize common TS/React patterns that materially improve retrieval quality. It does not need to understand every metaprogramming pattern.

### Chunk count must stay controlled

Improved chunking is only useful if it avoids both giant blobs and excessive fragmentation. The implementation should avoid creating a large number of tiny chunks for frontend files.

### Semantic enrichment should reflect chunk role

When a chunk is known to be a component, hook, exported function, or test block, that role should be reflected in semantic input so search can rank those chunks more accurately.

## Risks

- Over-fragmenting component files into low-value chunks
- Introducing parser complexity that slows indexing too much
- Failing on imperfect or partially invalid frontend source files

## Validation

- Run `go build ./...`
- Reindex a representative TS/React project and inspect chunk counts
- Run representative semantic searches against a small frontend project and confirm the top results improve versus current behavior
