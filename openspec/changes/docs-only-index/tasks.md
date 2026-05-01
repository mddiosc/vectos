## 1. Storage Layer — Docs Database Path

- [ ] 1.1 Update `internal/storage/project_manager.go` `GetDatabasePathForName()` to accept optional `suffix` parameter for docs DB path
- [ ] 1.2 Ensure docs DB path resolves to `<project>-docs.db` under the same project directory
- [ ] 1.3 Create `internal/storage/sqlite.go` `NewSQLiteStorageForDocsProjectName()` wrapper that passes `"-docs"` suffix
- [ ] 1.4 Verify both source and docs DBs are created under `~/.vectos/projects/<project-key>/`

## 2. Language Detection — Documentation Types

- [ ] 2.1 Update `cmd/vectos/runtime_paths.go` `detectLanguage()`: add `.mdx` → `"mdx"`, `.rst` → `"rst"`, `.adoc` → `"asciidoc"`, `.tex` → `"latex"`, `.txt` → `"text"`
- [ ] 2.2 Update `cmd/vectos/runtime_paths.go` `classifyCategory()`: ensure `"mdx"`, `"rst"`, `"asciidoc"`, `"latex"`, `"text"` return `"docs"`
- [ ] 2.3 Add tests in `cmd/vectos/runtime_paths_test.go` for new language detection

## 3. Indexing — Documentation-Only Mode

- [ ] 3.1 Update `cmd/vectos/runtime_paths.go` `collectIndexablePaths()` to accept `docsOnly bool` parameter: when `true`, only return paths where `classifyCategory(language) == "docs"`
- [ ] 3.2 Update `cmd/vectos/commands_index.go`: add `--docs` flag; pass `docsOnly=true` to `collectIndexablePaths`
- [ ] 3.3 Update `cmd/vectos/mcp_handlers.go` `makeIndexProjectHandler`: handle `docs: true` in `indexProjectInput`, open docs DB via `NewSQLiteStorageForDocsProjectName`
- [ ] 3.4 Update `cmd/vectos/commands_index.go` `runIndex()`: call docs DB storage when `--docs` is set

## 4. Search — New search_docs Tool

- [ ] 4.1 Add `searchDocsInput` struct in `cmd/vectos/mcp_server.go` with `Query`, `Path`, `Project` fields
- [ ] 4.2 Register `search_docs` tool in `registerMCPTools()` in `cmd/vectos/mcp_server.go`
- [ ] 4.3 Create `makeSearchDocsHandler()` in `cmd/vectos/mcp_handlers.go`: opens docs DB, executes search, returns same format as `search_code`
- [ ] 4.4 Update `cmd/vectos/commands_search.go`: add `--docs` flag to `runSearch()` CLI command
- [ ] 4.5 Add `IDX_DOCS_MISSING` guidance code in `cmd/vectos/mcp_format.go`

## 5. Agent Guidance — Suggest Docs on Empty Results

- [ ] 5.1 Update `cmd/vectos/mcp_handlers.go` `makeSearchCodeHandler`: check if docs DB exists and has chunks; if so and code search returns 0, set `TRY_DOCS` guidance
- [ ] 5.2 Update `cmd/vectos/mcp_format.go`: add `IDX_TRY_DOCS` guidance code and corresponding next_action
- [ ] 5.3 Update setup helpers (`setup/opencode.go`, `setup/claude.go`, `setup/codex.go`): add guidance about `search_docs` as complementary to `search_code`

## 6. CLI Help and Integration

- [ ] 6.1 Update `cmd/vectos/cli_help.go`: document `search --docs` and `index --docs`
- [ ] 6.2 Update `cmd/vectos/cli_help.go`: document `search_docs` MCP tool description
- [ ] 6.3 Update `cmd/vectos/commands_status.go`: add `--docs` flag to show docs DB stats

## 7. Testing

- [ ] 7.1 Run `go build ./...` and fix compilation errors
- [ ] 7.2 Run `go test ./... -count=1` and fix test failures
- [ ] 7.3 Manual test: `vectos index . --docs` and verify docs DB created with only documentation files
- [ ] 7.4 Manual test: `vectos search --docs "query"` and verify results come from docs DB only