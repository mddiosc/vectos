# CLI Usage

## Common Commands

### Show version

```bash
vectos version
```

### Index a file or directory

```bash
vectos index .
vectos index sample_code.go
```

Refresh only changed files within the selected project scope:

```bash
vectos index . --changed src/App.tsx,src/hooks/useAuth.ts
```

Inside an Nx workspace:

```bash
vectos index --project web .
```

### Search the current project index

```bash
vectos search "checkout payment"
```

Inside an Nx workspace:

```bash
vectos search --project web "checkout"
```

`search` uses semantic retrieval first when the active index metadata matches the current embedding provider, then falls back to text search if semantic retrieval is unavailable or incompatible.

### Show index status

```bash
vectos status
```

Inside an Nx workspace:

```bash
vectos status --project web
```

Example output includes:

- active project database path
- database size
- indexed files
- indexed chunks
- chunks with embeddings
- chunks without embeddings
- provider health
- whether reindexing is required

### Start the MCP server manually

```bash
vectos mcp
```

### Help

```bash
vectos help
vectos help setup
vectos setup --help
```

See also: [Indexing And Retrieval](indexing.md)

If commands do not behave as expected, also see [Troubleshooting](troubleshooting.md).
