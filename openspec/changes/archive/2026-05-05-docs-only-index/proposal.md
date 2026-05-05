## Why

Documentation files (markdown, rst, asciidoc, latex, text) are currently excluded from code indexing because they introduce noise in code search results. However, agents frequently need to search documentation separately — architecture decision records, API usage guides, README files, and other docs contain critical context that code search cannot provide. Currently there is no way to index and search documentation independently.

## What Changes

- **Separate docs database**: `<project>-docs.db` stores documentation chunks, `<project>.db` stores code chunks. Both can exist independently for the same project.
- **`search_docs` MCP tool**: New tool that searches the docs index using the same output format as `search_code`.
- **`--docs` CLI flags**: `vectos index --docs`, `vectos search --docs`, `vectos status --docs` flags for docs-only operations.
- **New documentation file types**: `.rst` (reStructuredText), `.adoc`/`.asciidoc` (AsciiDoc), `.tex`/`.latex` (LaTeX), `.txt` (plain text). Note: `.md`/`.mdx` already maps to `markdown` language and is categorized as `docs`.
- **TRY_DOCS guidance**: When `search_code` returns 0 results and a docs index exists, the response includes `guidance: "TRY_DOCS"` to suggest the alternative tool.
- **IDX_DOCS_MISSING guidance**: When `search_docs` is called but no docs index exists, the response includes `guidance: "IDX_DOCS_MISSING"`.

## Capabilities

### New Capabilities

- `docs-search`: New capability exposing `search_docs` tool for documentation context retrieval.
- `docs-indexing`: New indexing mode that processes only documentation files into a separate database.

### Modified Capabilities

- `mcp-interface`: Extended to expose `search_docs` tool alongside existing `search_code` and `index_project`. MCP search failures SHALL suggest documentation search alternative (TRY_DOCS guidance).

## Impact

- **Storage layer**: Database path resolution now accepts a suffix parameter; docs DB uses `-docs` suffix.
- **Language detection**: New language mappings for rst, asciidoc, latex, text — all classified as `docs` category.
- **MCP server**: New `search_docs` tool registered with same input/output shape as `search_code`.
- **CLI**: New `--docs` flags for `index`, `search`, and `status` commands.
- **Search semantic**: `SearchSemantic` accepts `includeDocs` bool to bypass the `category NOT IN ('docs', 'dependency_metadata')` WHERE clause.