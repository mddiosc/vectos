## 1. Storage Layer — Docs DB Path Resolution

- [x] 1.1 `GetDatabasePathForName` in `project_manager.go` already accepts suffix varargs parameter
- [x] 1.2 Docs DB path resolves to `<name>-docs.db` (e.g., `myproject-docs.db`)
- [x] 1.3 `NewSQLiteStorageForDocsProjectName` wrapper in `sqlite.go` passes "-docs" suffix
- [x] 1.4 `openStorageForScope` in `runtime_paths.go` accepts `docsOnly bool` and routes to appropriate storage constructor

## 2. Language Detection — Documentation Types

- [x] 2.1 `detectLanguage` in `runtime_paths.go`: add `.rst` → `rst`
- [x] 2.2 `detectLanguage` in `runtime_paths.go`: add `.adoc`, `.asciidoc` → `asciidoc`
- [x] 2.3 `detectLanguage` in `runtime_paths.go`: add `.tex`, `.latex` → `latex`
- [x] 2.4 `detectLanguage` in `runtime_paths.go`: add `.txt` → `text`
- [x] 2.5 `classifyCategory` in `runtime_paths.go`: ensure `rst`, `asciidoc`, `latex`, `text` return `"docs"`
- [x] 2.6 `classifyCategory` in `chunker.go`: ensure same languages return `"docs"` and is synced with `runtime_paths.go`

## 3. Indexing — Docs-Only Mode

- [x] 3.1 `collectIndexablePaths` in `runtime_paths.go` accepts `docsOnly bool` parameter
- [x] 3.2 `shouldIndexLanguage` in `runtime_paths.go`: when `docsOnly=true`, return only `category == "docs"`
- [x] 3.3 `--docs` CLI flag for `index` command in `cli_flags.go`
- [x] 3.4 `runIndex` in `commands_index.go` routes to docs storage when `docsOnly=true`
- [x] 3.5 `indexProjectInput` in `mcp_server.go` adds `Docs bool` field
- [x] 3.6 `makeIndexProjectHandler` in `mcp_handlers.go` handles `docs: true` by opening docs DB

## 4. MCP Server — search_docs Tool

- [x] 4.1 `searchDocsInput` struct in `mcp_server.go` with same shape as `searchCodeInput`
- [x] 4.2 `search_docs` tool registered in `registerMCPTools`
- [x] 4.3 `makeSearchDocsHandler` in `mcp_handlers.go` opens docs DB via `openStorageForScope(pm, scope, true)`
- [x] 4.4 Handler checks for zero chunks and returns `IDX_DOCS_MISSING` guidance
- [x] 4.5 `executeSearchDocs` in `search_ranking.go` uses `includeDocs=true` to bypass category filter

## 5. CLI — search and status --docs Flags

- [x] 5.1 `--docs` CLI flag for `search` command in `cli_flags.go`
- [x] 5.2 `--docs` CLI flag for `status` command in `cli_flags.go`
- [x] 5.3 `runSearch` in `commands_search.go` routes to docs storage when `docsOnly=true`
- [x] 5.4 `runStatus` in `commands_search.go` routes to docs storage when `docsOnly=true`

## 6. Guidance — TRY_DOCS and IDX_DOCS_MISSING

- [x] 6.1 `makeSearchCodeHandler` in `mcp_handlers.go`: when results empty and docs DB has chunks, set `guidance: "TRY_DOCS"`
- [x] 6.2 `makeSearchDocsHandler` in `mcp_handlers.go`: when docs index empty, set `guidance: "IDX_DOCS_MISSING"`
- [x] 6.3 Search semantic with `includeDocs=true` bypasses WHERE clause filter

## 7. Help Text

- [x] 7.1 Update `cli_help.go` printSubcommandHelp for `index` to document `--docs` flag
- [x] 7.2 Update `printSubcommandHelp` for `search` to document `--docs` flag
- [x] 7.3 Update `printSubcommandHelp` for `status` to document `--docs` flag

## 8. Build and Test

- [x] 8.1 Run `go build ./...` and verify no compilation errors
- [x] 8.2 Run `go test ./... -count=1` and verify all tests pass
- [x] 8.3 Manual test: `vectos index . --docs` indexes documentation files into docs DB
- [x] 8.4 Manual test: `vectos search "query" --docs` searches documentation index
- [x] 8.5 Manual test: `vectos status --docs` shows documentation index stats
- [x] 8.6 Manual test: `vectos mcp` exposes `search_docs` tool in tool list (verified via code inspection: registered at mcp_server.go:77)