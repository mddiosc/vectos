## Context

Documentation files are excluded from code indexing to prevent them from appearing in code search results. However, agents working in codebases that heavily use documentation (ADRs, architecture guides, API documentation) need a way to search this content separately. The codebase already has the storage, language detection, and MCP infrastructure needed — this change adds the documentation-specific paths and guidance.

**Constraints:**
- Docs and code must use separate database files to avoid cross-contamination.
- The `search_docs` tool must produce output in the same format as `search_code` for consistency.
- Documentation file detection must be correct so only docs are indexed in docs-only mode.
- TRY_DOCS guidance must appear only when it is genuinely useful (code returns 0 and docs exist).

## Goals / Non-Goals

**Goals:**
1. Separate docs DB (`<project>-docs.db`) from code DB (`<project>.db`).
2. New `search_docs` MCP tool that opens the docs DB and returns results in the same format as `search_code`.
3. `index_project` with `docs: true` parameter indexes only documentation files.
4. `--docs` CLI flags for `index`, `search`, and `status` commands.
5. TRY_DOCS guidance when code search returns 0 and a docs index has chunks.
6. IDX_DOCS_MISSING guidance when `search_docs` is called on an empty docs index.
7. Language detection for new doc types: `.rst`, `.adoc`, `.asciidoc`, `.tex`, `.latex`, `.txt`.

**Non-Goals:**
- Changing the code indexing behavior (docs remain excluded from code index).
- Supporting documentation-specific chunking strategies (uses same chunker as code).
- Merging docs and code search results in a single call.
- Modifying the semantic ranking algorithm for docs.

## Decisions

### Decision 1: Separate DB Files with Suffix

**Choice:** Docs database uses `<project>-docs.db` filename suffix. Code database uses `<project>.db` (no suffix).

```go
// project_manager.go
func (pm *ProjectManager) GetDatabasePathForName(projectName string, suffix ...string) (string, error) {
    projectKey := normalizeProjectName(projectName)
    dbSuffix := ""
    if len(suffix) > 0 {
        dbSuffix = suffix[0]
    }
    dbName := fmt.Sprintf("%s%s.db", projectKey, dbSuffix)
    return filepath.Join(pm.baseDir, projectKey, dbName), nil
}

// sqlite.go
func NewSQLiteStorageForDocsProjectName(pm *ProjectManager, projectName string) (*SQLiteStorage, error) {
    dbPath, err := pm.GetDatabasePathForName(projectName, "-docs")
    // ...
}
```

**Rationale:** Separate files allow independent lifecycle management. Code and docs can be reindexed separately. The suffix is descriptive and consistent with existing naming patterns.

**Alternatives considered:**
- Single DB with category column filter (would require schema change and mix concerns).
- Different directory for docs (adds complexity; suffix is simpler).

### Decision 2: Language Detection for Doc Types

**Choice:** New language mappings in `detectLanguage()` and `classifyCategory()`:

| Extension | Language | Category |
|-----------|----------|----------|
| `.rst` | `rst` | `docs` |
| `.adoc`, `.asciidoc` | `asciidoc` | `docs` |
| `.tex`, `.latex` | `latex` | `docs` |
| `.txt` | `text` | `docs` |
| `.md`, `.mdx` | `markdown` | `docs` (already implemented) |

**Rationale:** These are the primary documentation formats used in software projects. LaTeX uses `.tex` or `.latex` extension. Plain text `.txt` is included as a catch-all for documentation without markup.

**Note:** `.mdx` extension maps to `markdown` language (not `mdx`) to preserve structural chunking behavior — the markdown chunker is used for `.mdx` files.

### Decision 3: `--docs` Flag Behavior

**Choice:** `index_project` accepts `docs: true` parameter. CLI has `--docs` flags for `index`, `search`, and `status` commands.

```go
// mcp_server.go - input struct
type indexProjectInput struct {
    Path    string `json:"path"`
    Project string `json:"project,omitempty"`
    Changed string `json:"changed,omitempty"`
    Docs    bool   `json:"docs,omitempty"`  // NEW
}

// cli_flags.go
indexDocs: indexCmd.Bool("docs", false, "Index only documentation files into a separate docs database")
searchDocs: searchCmd.Bool("docs", false, "Search documentation index instead of source code")
statusDocs: statusCmd.Bool("docs", false, "Show documentation index status instead of source index")
```

**Rationale:** The bool parameter is optional and defaults to false, preserving backward compatibility. CLI flags provide discoverable documentation-only mode.

### Decision 4: New `search_docs` Tool

**Choice:** `search_docs` MCP tool has the same input shape as `search_code`:

```go
type searchDocsInput struct {
    Query   string `json:"query"`
    Path    string `json:"path,omitempty"`
    Project string `json:"project,omitempty"`
}
```

Handler opens docs DB and calls `executeSearchDocs`:

```go
func makeSearchDocsHandler(projectBaseDir string, embedConfig config.EmbeddingConfig) func(context.Context, *mcpSDK.CallToolRequest, searchDocsInput) (*mcpSDK.CallToolResult, any, error) {
    return func(ctx context.Context, req *mcpSDK.CallToolRequest, input searchDocsInput) (*mcpSDK.CallToolResult, any, error) {
        store, err := openStorageForScope(pm, scope, true)  // true = docsOnly
        // ...
        searchRun, err := executeSearchDocs(store, embedConfig, input.Query, 5)
        // ...
    }
}
```

**Rationale:** Same input shape allows agents to use either tool interchangeably. Same output format (`mcpSearchResultPayload`) ensures consistent handling downstream.

### Decision 5: TRY_DOCS Guidance

**Choice:** When `search_code` returns 0 results, check if docs DB has chunks. If so, set `guidance: "TRY_DOCS"` and `next_action: "Try search_docs tool instead, or run index_project with docs: true to index documentation."`

```go
// mcp_handlers.go in makeSearchCodeHandler
if len(payload.Results) == 0 {
    docsStore, docsErr := storage.NewSQLiteStorageForDocsProjectName(pm, scope.Name)
    if docsErr == nil {
        defer docsStore.Close()
        docsStats, statsErr := docsStore.Stats()
        if statsErr == nil && docsStats.ChunkCount > 0 {
            payload.Guidance = "TRY_DOCS"
            payload.NextAction = "Try search_docs tool instead, or run index_project with docs: true to index documentation."
        }
    }
}
```

**Rationale:** Zero results from code search when a docs index exists strongly suggests the user is looking for documentation, not code. The guidance is actionable and specific.

### Decision 6: IDX_DOCS_MISSING Guidance

**Choice:** When `search_docs` is called and the docs DB has zero chunks, set `guidance: "IDX_DOCS_MISSING"` and `next_action: "Use index_project with docs: true to index documentation files first."`

```go
// mcp_handlers.go in makeSearchDocsHandler
if stats.ChunkCount == 0 {
    payload := buildMCPSearchPayload(scope, input.Query, searchRun{Results: []storage.CodeChunk{}})
    payload.Guidance = "IDX_DOCS_MISSING"
    payload.NextAction = "Use index_project with docs: true to index documentation files first."
    // ...
}
```

**Rationale:** Symmetric with IDX_MISSING for code search. Makes it clear the docs index needs to be populated before search can work.

## Risks / Trade-offs

| Risk | Mitigation |
|---|---|
| Docs indexed by mistake when code intended | `--docs` flag is explicit; default behavior unchanged |
| Large docs DB increases storage | Docs DB is separate; users can delete it independently |
| `markdown` language includes both code comments and actual docs | Structural chunking works for both; noise is limited |

## Migration Plan

1. **Update `project_manager.go`**: Add suffix parameter to `GetDatabasePathForName` (already done).
2. **Add `NewSQLiteStorageForDocsProjectName`**: Wrapper that passes "-docs" suffix (already done).
3. **Update `detectLanguage`**: Add `.rst`, `.adoc`, `.asciidoc`, `.tex`, `.latex`, `.txt` mappings (already done).
4. **Update `classifyCategory`**: Ensure new languages return "docs" (already done in chunker.go and runtime_paths.go).
5. **Update `SearchSemantic`**: Add `includeDocs` bool parameter to bypass category filter (already done).
6. **Add `search_docs` MCP tool**: Register in `registerMCPTools` (already done).
7. **Add `makeSearchDocsHandler`**: Opens docs DB, calls `executeSearchDocs` (already done).
8. **Update `collectIndexablePaths`**: Accept `docsOnly` param, use `shouldIndexLanguage` (already done).
9. **Add `--docs` CLI flags**: For index, search, status commands (already done).
10. **Add TRY_DOCS and IDX_DOCS_MISSING guidance**: In handlers (already done).
11. **Update help text**: Document new flags (already done).
12. **Test**: `go build`, `go test`, manual verification of index/search/status with `--docs`.