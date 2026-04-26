## 1. MCP Server Extraction

- [x] 1.1 Extract MCP server startup and tool registration out of `cmd/vectos/main.go`
- [x] 1.2 Extract MCP search and indexing handler logic into dedicated MCP-focused files
- [x] 1.3 Keep MCP-specific schemas and formatting helpers organized around handler responsibilities

## 2. Validation

- [x] 2.1 Validate `search_code` and `index_project` behavior after the refactor
- [x] 2.2 Run `go test ./...`
- [x] 2.3 Run `go build ./...`
