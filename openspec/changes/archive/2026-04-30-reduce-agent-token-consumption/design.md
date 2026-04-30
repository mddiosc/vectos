## Context

Vectos is a local-first code context engine that indexes source code with embeddings and exposes search tools over MCP. Currently, every search query returns a chunk-level array where each chunk is a separate JSON object containing: `rank`, `file_path`, `file_name`, `start_line`, `end_line`, `language`, `category`, `score`, `preview`, and `reason`.

A typical successful search returns 5 chunks, each consuming roughly 100-150 tokens of MCP output. When multiple chunks come from the same file (which is common in structurally chunked codebases), metadata like `file_path`, `file_name`, `language`, and `category` is repeated 2-4 times. Previews are code fragments truncated mid-line, which convey less semantic information to the agent than a function signature would, yet consume comparable tokens.

The goal is to make the agent's token budget go further. When an agent uses native grep, it gets raw text and spends tokens "navigating" until it finds the right file. Vectos should outperform grep not by returning *more* text, but by returning *smarter pointers*: "Read `SearchSemantic` in `internal/storage/sqlite.go` at line 296." This requires restructuring the output format, extracting richer metadata during indexing, and making the preview tier adaptive to confidence.

**Constraints:**
- SQL schema can be altered (add columns), but existing `embedding` BLOB must remain compatible.
- MCP output is JSON; must stay valid and parsable by all agents.
- CLI output can change format slightly for consistency, but primary impact target is MCP.
- No new external dependencies if avoidable.

## Goals / Non-Goals

**Goals:**

1. Reduce average MCP search response size by ~60% for high-confidence queries (fewer tokens per result, less redundant metadata).
2. Shift from chunk-level output to file-level output with consolidated ranges and function signatures.
3. Implement confidence-based preview tiers: high confidence returns pointer-only (no code preview); low confidence returns contextual preview.
4. Collapse overlapping chunks from the same file into a single entry.
5. Reduce the candidate pool surface in hybrid ranking from 25 to 10 to produce fewer, higher-confidence results.
6. Store function `signature` and `purpose` during indexing so they can be returned without re-parsing at query time.

**Non-Goals:**

- Switching to a vector-native index (e.g., sqlite-vss). This change is orthogonal to output format.
- Changing the embedding model or ONNX runtime details.
- Adding batching or concurrency during index time.
- Changing the MCP tools schema / input parameters.

## Decisions

### Decision 1: File-Level Output Structure

**Choice:** Replace the per-chunk `mcpSearchResultEntry` array with a per-file `mcpSearchFileEntry` array. Each entry contains: `file_path`, `file_name`, `language`, `category`, `relevance`, `signatures` (array of strings), `line_ranges` (array of {start, end}), and an optional `hint` string.

**Rationale:** Signatures (e.g., `func SearchSemantic(...)`) are semantically dense: an LLM can infer what a function does from its signature alone in ~20 tokens. A truncated code preview of similar length often contains partial lines without context, wasting tokens. Grouping by file eliminates duplication of file metadata.

**Alternatives considered:**
- Keep chunk-level but dedupe metadata (still wasteful, signatures are still hidden inside previews).
- Return full function bodies instead of signatures (too many tokens, agent can `read` if needed).

### Decision 2: Token-Budget Preview Tiers Based on Score

**Choice:** Define a high-confidence threshold (default 0.90). Results at or above this threshold include only file path, line ranges, and signatures — no `hint` or content preview. Results below the threshold include a contextual `hint` (brief descriptive string) to help the agent understand why this result was returned.

**Rationale:** When Vectos is "sure" (score >= 0.90), the agent rarely needs a preview to decide to read the file. When uncertain, the extra context helps the agent filter and avoid wasted reads.

**Alternatives considered:**
- Always include preview (current behavior; wastes tokens).
- Dynamic threshold based on score gap between 1st and 2nd result (too complex, less predictable).

### Decision 3: Collapse Overlapping Chunks Per File

**Choice:** After ranking, fold multiple chunks from the same file into a single entry if their line ranges overlap or touch within a configurable window (default 5 lines). Their line ranges are merged into the union; signatures are collected from all source chunks (deduplicated); relevance score becomes the max of the merged chunks.

**Rationale:** Reduces entry count when a query matches multiple adjacent functions in the same file. Prevents the agent from seeing 3 results all pointing to different functions in `server.go`, which bloats output without adding decision-making value.

**Alternatives considered:**
- Only collapse exact overlap (too strict; touching functions should merge).
- Collapse by signature equality (requires reliable signature extraction, not always possible for non-structured languages).

### Decision 4: Reduce Candidate Pool from 25 to 10

**Choice:** Lower `hybridCandidateLimit` from 25 to 10. Remove approximately half of the heuristic boosts that adjust score by < 0.06 and rarely change top-5 ordering.

**Rationale:** The current 25-candidate pool with 40+ boost/penalty constants is tuned for "find the 25 best results" rather than "give the agent the 5 most useful pointers." Simplifying the ranking surface reduces variance and increases confidence in top results, enabling the token-saving tier behavior above.

**Alternatives considered:**
- Keep 25 but only return top 5 formatted differently (still wastes ranking CPU and complicates the code).
- Replace all heuristics with a learned model (overkill, adds dependency).

### Decision 5: Extend `code_chunks` Schema with `signature` and `purpose`

**Choice:** Add `signature TEXT` and `purpose TEXT` columns to `code_chunks`. Populate during indexing via the existing `extractSignature()` and `inferPurpose()` helpers in `chunker.go`.

**Rationale:** These fields are already computed during embedding enrichment (`buildSemanticContent`). Storing them avoids reparsing at query time (which would need language-specific regex per query, adding latency). SQLite TEXT column overhead is negligible.

**Alternatives considered:**
- Extract signatures on-the-fly at query time from `content` column (adds regex parsing latency per result, duplicates existing logic).
- Store only in embeddings/metadata (not directly queryable for the output format).

## Risks / Trade-offs

| Risk | Mitigation |
|---|---|
| [Risk] MCP schema change breaks agent clients that parse `preview` field | [Mitigation] The agent SDKs parse generic JSON; Vectos provides clear descriptions in MCP tool schemas. The `hint` field replaces `preview` for low-confidence results, preserving backward-compatible meaning. If an agent relied on the exact JSON keys, it already had to be robust because Vectos fields were documented but not guaranteed. |
| [Risk] Signature extraction is imperfect for less-structured languages (e.g., shell, markdown), producing empty `signatures` arrays | [Mitigation] File-level output gracefully degrades: if signatures array is empty, `hint` is always populated (derived from `purpose` or category). The output remains useful. |
| [Risk] Reducing candidate pool to 10 may miss edge cases where the correct result had rank 11-25 | [Mitigation] The semantic base score is the primary signal; heuristics only reorder within candidates. If a chunk with the correct answer had semantic score rank 15, it was already unlikely to reach top 5 after heuristic reordering. Simplifying boosts actually makes ranking more deterministic and easier to reason about. |
| [Risk] Collapsing chunks by line range may merge unrelated functions if they are close in the file | [Mitigation] The 5-line window is conservative; two unrelated functions separated by > 5 blank lines or comments stay separate. The window is a CONSTANT tunable if data shows over-merging. |
| [Trade-off] CLI output will also shift to file-level grouping. This is slightly less convenient for humans who might prefer line-by-line. | CLI `--full` flag remains available to print full content; normal CLI output is primarily consumed by scripts/agents rather than humans reading directly. |

## Migration Plan

1. **Schema migration**: The `migrate()` function in `sqlite.go` already handles `ALTER TABLE ADD COLUMN` with duplicate-column-name guards. Adding `signature` and `purpose` uses the same pattern.
2. **Reindex flag**: After deployment, `vectos status` will detect that existing chunks lack `signature`/`purpose` and recommend a reindex (already supported via `RequiresReindex` logic). Users can run `vectos index .` to backfill.
3. **Backward compatibility**: Old chunks without signatures will simply return empty `signatures` arrays; the `hint` field (from `purpose` or category) ensures the output is never blank.
4. **Rollback**: The schema change is additive only. Rolling back the code returns to reading `content` for previews instead of signatures. No data loss.

## Open Questions

1. Should the high-confidence threshold (0.90) be configurable via CLI flag or environment variable for different embedding models with different score distributions?
2. For CLI output, should we keep a `--chunks` flag to switch back to chunk-level display for debugging?
3. Should file-level collapse also apply to deduplication during indexing (store fewer overlapping chunks), or only at query time?
