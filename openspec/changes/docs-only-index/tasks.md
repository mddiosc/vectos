## 1. Storage Layer — Docs Database Path

- [ ] 1.1 Update `internal/storage/project_manager.go` `GetDatabasePathForName()` to accept optional variadic `suffix` parameter for docs DB path (`"-docs"`)
- [ ] 1.2 Ensure docs DB path resolves to `<project>-docs.db` under the same project directory
- [ ] 1.3 Create `internal/storage/sqlite.go` `NewSQLiteStorageForDocsProjectName()` wrapper that passes `"-docs"` suffix
- [ ] 1.4 Add `includeDocs bool` parameter to `internal/storage/sqlite.go` `SearchSemantic()`: when `true`, skip the `category NOT IN ('docs', 'dependency_metadata')` WHERE clause so docs chunks are returned

## 2. Language Detection — Documentation Types

- [ ] 2.1 Update `cmd/vectos/runtime_paths.go` `detectLanguage()`: add `.rst` → `"rst"`, `.adoc` / `.asciidoc` → `"asciidoc"`, `.tex` / `.latex` → `"latex"`, `.txt` → `"text"`. Note: `.mdx` already maps to `"markdown"` — keep it as-is to preserve structural chunking support.
- [ ] 2.2 Update `cmd/vectos/runtime_paths.go` `classifyCategory()`: add `"rst"`, `"asciidoc"`, `"latex"`, `"text"` → `"docs"` (`.mdx` already maps to `"markdown"` which is already `"docs"`)
- [ ] 2.3 Update `internal/indexer/chunker.go` `classifyCategory()`: add the same new doc language mappings (`"rst"`, `"asciidoc"`, `"latex"`, `"text"` → `"docs"`) to keep both copies in sync

## 3. Indexing — Documentation-Only Mode

- [ ] 3.1 Update `cmd/vectos/runtime_paths.go`: refactor `shouldIndexLanguage()` to accept `docsOnly bool` parameter. When `true`, return `category == "docs"`. When `false`, keep current behavior (`!= "docs" && != "dependency_metadata"`)
- [ ] 3.2 Update `cmd/vectos/runtime_paths.go` `collectIndexablePaths()` to accept and pass `docsOnly bool` to `shouldIndexLanguage()`
- [ ] 3.3 Update `cmd/vectos/cli_flags.go`: add `indexDocs *bool` flag to `cliFlags` struct and register via `indexCmd.Bool("docs", false, "Index only documentation files")`
- [ ] 3.4 Update `cmd/vectos/cli_dispatch.go` `runIndexCommand()`: pass `*app.flags.indexDocs` to `runIndex()`
- [ ] 3.5 Update `cmd/vectos/commands_index.go` `runIndex()`: accept `docsOnly bool` parameter. When `true`, use `NewSQLiteStorageForDocsProjectName` and pass `docsOnly=true` to `collectIndexablePaths`
- [ ] 3.6 Update `cmd/vectos/mcp_server.go` `indexProjectInput`: add `Docs bool \`json:"docs,omitempty"\``
- [ ] 3.7 Update `cmd/vectos/mcp_handlers.go` `makeIndexProjectHandler`: when `input.Docs == true`, open docs DB via `NewSQLiteStorageForDocsProjectName` and pass `docsOnly=true` to `collectIndexablePaths`

## 4. Search — New search_docs Tool

- [ ] 4.1 Add `searchDocsInput` struct in `cmd/vectos/mcp_server.go` with `Query`, `Path`, `Project` fields (same shape as `searchCodeInput`)
- [ ] 4.2 Register `search_docs` tool in `registerMCPTools()` in `cmd/vectos/mcp_server.go` with description `"Search through project documentation using semantic search"`
- [ ] 4.3 Create `makeSearchDocsHandler()` in `cmd/vectos/mcp_handlers.go`: opens docs DB via `NewSQLiteStorageForDocsProjectName`, calls `SearchSemantic(queryVector, limit, true)` (with `includeDocs=true`), returns same format as `search_code`
- [ ] 4.4 Add `IDX_DOCS_MISSING` guidance code and payload builder in `cmd/vectos/mcp_format.go`
- [ ] 4.5 Update `cmd/vectos/cli_flags.go`: add `searchDocs *bool` flag to `cliFlags` struct and register via `searchCmd.Bool("docs", false, "Search documentation index")`
- [ ] 4.6 Update `cmd/vectos/cli_dispatch.go` `runSearchCommand()`: pass `*app.flags.searchDocs` to `runSearch()`
- [ ] 4.7 Update `cmd/vectos/commands_search.go` `runSearch()`: accept `docsOnly bool` parameter. When `true`, open docs DB and pass `includeDocs=true` to search

## 5. Agent Guidance — Suggest Docs on Empty Results

- [ ] 5.1 Update `cmd/vectos/mcp_handlers.go` `makeSearchCodeHandler`: when search returns 0 results, check if docs DB file exists via `os.Stat` on docs DB path. If it exists, set `TRY_DOCS` guidance
- [ ] 5.2 Update `cmd/vectos/mcp_format.go`: add `TRY_DOCS` guidance code and corresponding next_action suggesting `search_docs`
- [ ] 5.3 Update setup helpers (`internal/setup/opencode.go`, `internal/setup/claude.go`, `internal/setup/codex.go`): add guidance about `search_docs` as complementary to `search_code`

## 6. CLI Help and Status

- [ ] 6.1 Update `cmd/vectos/cli_help.go`: document `search --docs` and `index --docs`
- [ ] 6.2 Update `cmd/vectos/cli_help.go`: document `search_docs` MCP tool description
- [ ] 6.3 Update `cmd/vectos/cli_flags.go`: add `statusDocs *bool` flag to `cliFlags` struct and register via `statusCmd.Bool("docs", false, "Show documentation index status")`
- [ ] 6.4 Update `cmd/vectos/cli_dispatch.go` `runStatusCommand()`: pass `*app.flags.statusDocs` to `runStatus()`
- [ ] 6.5 Update `cmd/vectos/commands_search.go` `runStatus()`: accept `docsOnly bool` parameter. When `true`, open docs DB for stats display

## 7. Testing

- [ ] 7.1 Run `go build ./...` and fix compilation errors
- [ ] 7.2 Run `go test ./... -count=1` and fix test failures
- [ ] 7.3 Manual test: `vectos index . --docs` and verify docs DB created with only documentation files
- [ ] 7.4 Manual test: `vectos search --docs "query"` and verify results come from docs DB only