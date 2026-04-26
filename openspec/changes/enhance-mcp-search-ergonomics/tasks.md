## 1. MCP Search Result Ergonomics

- [ ] 1.1 Define the concise result metadata that MCP search responses should always include
- [ ] 1.2 Update `search_code` MCP responses to return richer actionable summaries without bloating output unnecessarily
- [ ] 1.3 Reuse or align formatting logic with existing CLI result presentation where it improves consistency

## 2. Recovery And Indexing Guidance

- [ ] 2.1 Detect missing-index search situations and return explicit indexing guidance through MCP
- [ ] 2.2 Detect stale or incomplete index situations when feasible and return explicit refresh guidance through MCP
- [ ] 2.3 Ensure `index_project` summaries give agents enough feedback to know whether indexing or refresh succeeded

## 3. Validation

- [ ] 3.1 Add tests for MCP result metadata and recovery guidance behavior
- [ ] 3.2 Validate the ergonomics with a real MCP client flow against an indexed project and an unindexed project
- [ ] 3.3 Run `go build ./...` and confirm MCP initialization and tool discovery remain unchanged
