## Context

Vectos v0.2.0 reduced MCP search response tokens by ~60% using file-level output with signatures. However, three sources of token overhead remain:

1. **Guidance strings**: Error payloads include verbose English guidance (~120-150 chars when warning is present). These are useful but disproportionate for machine-to-machine protocol.

2. **Redundant metadata in search results**: Each result includes `Rank`, `FileName`, `Language`, `Category` — all derivable by the client from other fields or trivial computation.

3. **Absolute file paths**: Full paths like `/Users/name/project/foo/bar.go` are sent per result. Since the project root is already known to the agent, relative paths (`foo/bar.go`) suffice.

4. **Embedding enrichment overhead**: Every chunk gets ~31 chars of `Language:` and `Category:` prefix in the text sent to the embedding model. This is redundant signal the model can infer from code structure.

**Constraints:**
- MCP output must remain valid JSON.
- Error codes must be documented so agents can decode them.
- Embedding quality must not degrade when enrichment is reduced.

## Goals / Non-Goals

**Goals:**
1. Reduce guidance/nextAction overhead from ~120-150 chars to ~11-13 chars per warning.
2. Remove `Rank`, `FileName`, `Language`, `Category` fields from MCP search results.
3. Use relative paths from project root instead of absolute paths.
4. Remove `Language:` and `Category:` from `buildSemanticContent()` to reduce embedding input by ~31 chars/chunk.
5. Convert `Relevance` from float64 to int (0-100) to save ~4 bytes/result.

**Non-Goals:**
- Changing the MCP tool input schema (tool call parameters stay the same).
- Modifying the semantic ranking algorithm (only output format changes).
- Adding new capabilities (this is a compression pass).
- Supporting path aliases or symlink resolution in relative path computation.

## Decisions

### Decision 1: Guidance Error Codes

**Choice:** Replace verbose guidance strings with short uppercase codes. Codes are short (11-13 chars), consistent format, and map to documented meanings.

| Context | Before (chars) | After (chars) |
|---|---|---|
| Missing index | `"This project does not have a usable Vectos index yet."` | `IDX_MISSING` |
| Stale index | `"Refresh the project index before trusting semantic ranking."` | `IDX_STALE` |

The `guidance` field carries only the short error code. The `next_action` field carries the actual command to run (unchanged — keeps the existing `suggestedIndexAction` / `suggestedRefreshAction` strings).

**Rationale:** The verbose guidance is human-readable but wastes tokens in machine-to-machine protocol. Agents can decode `IDX_MISSING` as "index not ready, run index_project". The command to fix is already in `next_action`.

**Alternatives considered:**
- Keep full text (wastes tokens without adding machine-readable value).
- Use numeric codes (e.g., `E001`) — less self-documenting, requires separate documentation lookup.
- Omit guidance entirely — loses useful recovery context.

### Decision 2: Remove Redundant Fields from MCP Search Results

**Choice:** Remove `Rank`, `FileName`, `Language`, `Category` from `mcpSearchFileResult`. Keep `FilePath` (as relative), `Relevance` (as int), `LineRanges`, `Signatures`, `Hint`.

**Rationale:**
- `Rank`: Already inferable from array position (client knows index i = rank i+1).
- `FileName`: `filepath.Base(FilePath)` gives the same value.
- `Language`: Extensible from file extension (`.go` → `go`, `.ts` → `typescript`, etc.).
- `Category`: Reconstructed via `classifyCategory(language)` using the same logic as during chunking.

**Alternatives considered:**
- Keep all fields and compress JSON (less effective, still adds bytes).
- Remove only `Rank` and `FileName` but keep `Language`/`Category` (incomplete — 2 of 4 removed).
- Compute relative path but keep absolute (conflicting optimization).

### Decision 3: Relative Paths via `filepath.Rel`

**Choice:** Compute relative paths using `filepath.Rel(scope.PrimaryRoot, fr.FilePath)` in `buildMCPSearchPayload()`. Use the existing `PrimaryRoot` from the `workspace.Scope` already passed to the function.

**Rationale:** The `Scope` is available in `buildMCPSearchPayload` and contains `PrimaryRoot`. `filepath.Rel` is a pure string operation — negligible overhead. Relative paths are typically 40-100 chars shorter than absolute paths.

**Alternatives considered:**
- Compute relative path once and cache (already per-request, no caching needed).
- Send project root offset instead of relative path (adds complexity for minimal gain).
- Let client compute relative path (requires client to know project root, which they do — but would require extra field to convey it).

### Decision 4: Remove `Language:` and `Category:` from Embedding Enrichment

**Choice:** In `buildSemanticContent()`, remove the `Language:` and `Category:` prefix lines. Keep only `File:` (basename), `Signature:` (if present), `Purpose:` (if present), and `Code:\n`.

**Rationale:** The embedding model processes the full code content. The `Language:` and `Category:` labels are explicit metadata that the model could infer from code structure. Removing them saves ~31 chars per chunk with no embedding quality degradation — the model already sees the code and can determine language from syntax (e.g., `func` → Go, `def` → Python, `import` statements).

**Risk:** Low. If the model used these labels as strong signals, removing them could marginally affect ranking. Benchmark tests (see tasks) will verify no degradation.

**Alternatives considered:**
- Remove only `Category:` (saves ~17 chars — lower impact but lower risk).
- Keep both (no savings — current state).
- Replace with numeric category ID (saves ~5 chars — marginal improvement).

### Decision 5: Integer Relevance Encoding

**Choice:** Convert `Relevance` from float64 to int by multiplying by 100 and rounding. Send `87` instead of `0.8712`.

**Rationale:** JSON float encoding uses more bytes than integer for comparable precision. At 2 decimal places, int 0-100 captures same information as float 0.00-1.00.

**Alternatives considered:**
- Keep float64 (current state — more bytes per result).
- Use string (worse — adds quotes and more chars).
- Compress to 0-1000 scale (overkill — 2 decimal places sufficient).

## Risks / Trade-offs

| Risk | Mitigation |
|---|---|
| Agent clients parse guidance by exact English string | Document codes in MCP tool description; codes are self-documenting. Provide fallback decoding logic in agent SDKs. |
| Removing Language/Category degrades embedding quality | Run benchmark before/after; revert if hit rate drops >5%. |
| Relative path computation adds latency | `filepath.Rel` is a pure string operation; negligible compared to embedding lookup. Falls back to absolute path if `filepath.Rel` fails. |

## Migration Plan

1. **Update MCP tool descriptions** in `mcp_server.go` to document error codes (`IDX_MISSING`, `IDX_STALE`).
2. **Update `mcp_format.go`** with guidance codes and field removals.
3. **Update tests** in `mcp_format_test.go` and `mcp_payload_test.go` with new field expectations.
4. **Update `buildSemanticContent()`** to remove Language/Category prefix.
5. **Run benchmarks** to verify ranking quality unchanged.
6. **Deploy and monitor** — track if agent success rates change.

No database migration needed (this change affects only runtime output).

## Open Questions

1. Should `Hint` also be compressed or removed for high-confidence results? (Currently already empty for ≥0.90 — no change needed.)
2. Should the error code prefix be consistent? Currently: `IDX_*` for index-related codes. Is this sufficient or should we have broader `MCP_*` prefix?
3. Should benchmark tracking be automated so future changes are validated against hit rate baseline?