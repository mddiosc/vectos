## Why

Vectos already reduced MCP search response size by ~60% in v0.2.0 by switching from chunk-level to file-level output. However, there is still meaningful token overhead in error guidance strings, MCP response metadata fields, absolute file paths, and embedding enrichment text that can be trimmed further. The goal is to make Vectos consume meaningfully fewer tokens than equivalent grep/glob operations while preserving or improving result accuracy.

## What Changes

- **Replace guidance/nextAction text with short error codes**: `"This project does not have a usable Vectos index yet."` → `IDX_MISSING`. `"Refresh the project index before trusting semantic ranking."` → `IDX_STALE`. Reduces ~120-150 chars to ~11-13 chars when warnings are present.
- **Remove `Rank` field from MCP search results**: Already inferable from array position; no need to send it.
- **Remove `FileName`, `Language`, `Category` from MCP search results**: Client can derive `FileName` from `filepath.Base(FilePath)`, `Language` from file extension, and `Category` from `Language` via `classifyCategory()`.
- **Use relative paths instead of absolute paths**: Replace `/Users/name/project/foo/bar.go` with `foo/bar.go` (relative to project root). Saves ~40-100 chars per result.
- **Reduce embedding enrichment text**: Remove `Language:` and `Category:` prefix lines from `buildSemanticContent()` — the model infers these from code structure; removing saves ~31 chars per chunk with no ranking degradation.
- **Shorten `Relevance` float encoding**: Send relevance as integer 0-100 instead of float64 (e.g., `87` instead of `0.8712`). Saves ~4 bytes per result.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `mcp-interface`: Replace verbose guidance strings with short error codes; remove redundant metadata fields (Rank, FileName, Language, Category) from search response; use relative paths from project root.
- `semantic-search`: Remove `Language:` and `Category:` from embedding enrichment text.

## Impact

- **Affected modules**: `cmd/vectos/mcp_format.go`, `cmd/vectos/mcp_format_test.go`, `cmd/vectos/mcp_payload_test.go`, `internal/indexer/chunker.go`
- **Breaking change**: Agent clients that parse guidance strings by exact text will need updating to handle codes. Search result payloads change field set.
- **No schema migration needed**: This change affects only runtime output, not stored data.
- **Low risk**: Guidance codes are straightforward string replacements; fields being removed are redundant or derivable.