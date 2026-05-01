## Context

Vectos currently excludes documentation files from source code indexing via `shouldIndexLanguage()`, which filters out `category == "docs"` and `category == "dependency_metadata"`. This prevents documentation chunks from polluting code search results, but it also means documentation is completely unindexable.

The goal is to provide a separate documentation index that:
1. Coexists with the source code index under the same project scope
2. Is searchable only via an explicit `search_docs` tool
3. Does not modify existing `search_code` behavior

**Constraints:**
- MCP output format stays JSON and parsable by agents
- No changes to existing tool input schemas for `search_code` and `index_project`
- Both indexes share the same `~/.vectos/projects/<project>/` directory
- New languages must not conflict with existing language detection

## Goals / Non-Goals

**Goals:**
1. Add `--docs` flag to `index_project` that indexes only documentation files into `<name>-docs.db`
2. Create a new `search_docs` MCP tool that searches only the docs database
3. Support new documentation file types: `.mdx`, `.rst`, `.adoc`, `.tex`
4. Provide guidance suggesting `search_docs` when `search_code` returns no results
5. Keep both indexes under the same project scope with separate database files

**Non-Goals:**
- Adding PDF support (binary, not chunkable)
- Mixing docs and source in the same database
- Auto-detecting when to use docs vs code search
- Changing existing `search_code` or `index_project` behavior

## Decisions

### Decision 1: Separate Database Files with Suffix

**Choice:** Use `<name>-docs.db` for the documentation index. Source remains `<name>.db`.

```
~/.vectos/projects/<project-key>/
  <project>.db        ← source code (unchanged)
  <project>-docs.db   ← documentation only
```

**Rationale:** Clean separation without schema changes. Each DB is independent and can be reindexed independently. No migration needed.

**Alternatives considered:**
- Single DB with a `is_documentation` boolean column (pollutes schema, requires migration)
- Subdirectory per index type (more complex path handling)
- Table suffix instead of file suffix (requires more SQLite complexity)

### Decision 2: Language Detection for Documentation Types

**Choice:** Extend `detectLanguage()` to recognize new file types and update `classifyCategory()` to return `"docs"` for all documentation languages.

**New mappings:**

| Extension | Language | Category |
|---|---|---|
| `.md` | markdown | docs |
| `.mdx` | mdx | docs |
| `.rst` | rst | docs |
| `.adoc` / `.asciidoc` | asciidoc | docs |
| `.tex` / `.latex` | latex | docs |
| `.txt` | text | docs |

**Rationale:** Consistent with existing pattern — `classifyCategory()` maps language → category. Changing `shouldIndexLanguage()` to optionally invert the filter (docs only vs source only) keeps the logic centralized.

**Alternatives considered:**
- Separate `shouldIndexDocs()` function (duplicates logic)
- Language allowlist in `collectIndexablePaths()` (more error-prone)

### Decision 3: `--docs` Flag Behavior

**Choice:** `index_project` takes a `docs bool` field. When `true`, only files with `category == "docs"` are indexed. When `false` (default), existing behavior (source only, excludes docs).

**Rationale:** Minimal change to existing API. `docs` field is optional and defaults to `false`, preserving backward compatibility.

**Alternatives considered:**
- Separate `index_docs` tool (adds complexity, two tools doing similar things)
- Flag in path: `index_docs` (too implicit)

### Decision 4: New `search_docs` Tool

**Choice:** Create a new `search_docs` MCP tool with the same input shape as `search_code` (`query`, `path`, `project`), but opening the docs database instead.

**Rationale:** Explicit tool for docs search. Agents choose which tool to call. Clean separation of concerns.

**Alternatives considered:**
- Add `type` field to `search_code` input (mixes concerns, more complex handler)
- Single tool with `search_type` parameter (same downside)

### Decision 5: Guidance Suggestion

**Choice:** When `search_code` returns zero chunks AND the docs database exists and has chunks, include `TRY_DOCS: <hint>` in the guidance field.

**Rationale:** Agents get contextual hint without automatic fallback behavior. Explicit is better than implicit.

**Alternatives considered:**
- Auto-query docs DB when code search fails (unpredictable, hides failures)
- Return two separate result arrays (confusing, mixes concerns)

## Risks / Trade-offs

| Risk | Mitigation |
|---|---|
| User indexes docs without `--docs` flag accidentally | `shouldIndexLanguage` already filters docs out by default |
| Docs DB grows too large | Separate DB means it can be deleted independently with `vectos index --docs --clear` (future) |
| Agent doesn't know to use `search_docs` | Guidance on empty results suggests it; setup helpers updated |

## Migration Plan

1. No database migration needed — new files created as needed
2. `index_project` with `docs: true` creates `<name>-docs.db` if not exists
3. `search_docs` returns error if docs DB doesn't exist (guides user to index with `--docs`)
4. No rollback concerns — no existing behavior changes

## Open Questions

1. Should `vectos status` show both source and docs DB stats? (Yes — add `--docs` flag to `status`)
2. Should `index --docs` delete the docs DB first, or merge? (Merge — consistent with incremental behavior)
3. Should we index `node_modules/README.md` files? (No — `shouldSkipDir` already excludes `node_modules`)