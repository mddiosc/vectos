## Why

Currently Vectos excludes documentation files (`.md`, `.mdx`, `.rst`, `.txt`, `.adoc`) from source code indexing because they introduce noise in code search results. However, agents working with documentation-heavy projects — READMEs, API docs, architecture decision records — benefit from having a separate documentation search capability that does not pollute code results. This allows agents to explicitly search documentation via a separate tool without mixing doc chunks with source code chunks.

## What Changes

- **New `search_docs` MCP tool**: A separate tool that searches only the documentation index for a project. Distinct from `search_code` which handles source code.
- **New `--docs` flag for `index_project`**: When provided, only documentation files are indexed into a separate DB (`<project>-docs.db`). Without the flag, behavior is unchanged (source code only).
- **Separate documentation database**: Documentation index stored in `<name>-docs.db` alongside the existing `<name>.db` file, both under the same project scope.
- **New documentation language detection**: Support for `.mdx`, `.rst`, `.adoc`, `.tex` as documentation file types (beyond existing markdown).
- **Updated guidance in agent setup**: When `search_code` yields no results, suggest trying `search_docs` as an alternative.

## Capabilities

### New Capabilities

- `docs-search`: A semantic search tool that operates exclusively over a documentation-specific index. Returns file paths, line ranges, section headings, and relevance scores for documentation chunks. Triggered by `search_docs` MCP tool or `vectos search --docs` CLI command.
- `docs-indexing`: An indexing mode that processes documentation files (markdown, mdx, rst, asciidoc, LaTeX, plain text) into a separate project-specific database. Triggered by `index_project --docs` or `vectos index --docs`.

### Modified Capabilities

- `mcp-interface`: Add new `search_docs` MCP tool alongside existing `search_code` and `index_project` tools.
- `code-indexing`: Add `--docs` flag to indexing operations, enabling documentation-only indexing into a separate database file.

## Impact

- **New files/APIs**: New `search_docs` tool in MCP server (`mcp_server.go`), new `--docs` flag in CLI (`commands_index.go`, `commands_search.go`).
- **Storage layer**: `GetDatabasePathForName` or new variant accepts a suffix for the docs DB. Separate database file `<project>-docs.db` created under the same project directory.
- **Language detection**: New language types (`mdx`, `rst`, `adoc`, `latex`) classified as documentation.
- **Agent guidance**: Setup helpers (`setup/opencode.go`, `setup/claude.go`, `setup/codex.go`) updated to mention `search_docs` as an option when code search returns no results.
- **No breaking changes**: Existing `search_code` and `index_project` behavior is unchanged.