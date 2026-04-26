## 1. MCP Server Extraction

- [ ] 1.1 Extract MCP server startup and tool registration out of `cmd/vectos/main.go`
- [ ] 1.2 Extract MCP search and indexing handler logic into dedicated MCP-focused files
- [ ] 1.3 Keep MCP-specific schemas and formatting helpers organized around handler responsibilities

## 2. Validation

- [ ] 2.1 Validate `search_code` and `index_project` behavior after the refactor
- [ ] 2.2 Run `go test ./...`
- [ ] 2.3 Run `go build ./...`
