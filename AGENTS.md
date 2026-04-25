# Vectos Agent Rules

When you need to search this codebase semantically, prefer the `vectos_search_code` MCP tool before using `grep` or broad file reads.

If the requested code is not yet indexed or the MCP search returns no useful result, use `vectos_index_project` and then retry `vectos_search_code`.

Use `grep` and direct file reads as a fallback when MCP search is insufficient or when exact string matching is required.

If session-memory tools such as Engram are also available, use them for prior decisions, durable learnings, and historical context. Use Vectos for current code retrieval. Treat them as complementary, not coupled.
