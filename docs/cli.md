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

If you run the command from inside a project directory, Vectos can often infer the workspace automatically and still use the selected project scope:

```bash
cd apps/app-main
vectos index .
```

For Nx workspaces, the selected project scope can expand to include internal dependent libs when the Nx project graph is available.

### Search the current project index

```bash
vectos search "checkout payment"
```

By default, `search` prints compact snippets with file, line range, score, and a short
reason. Use `--full` only when you need the complete chunk content.

Inside an Nx workspace:

```bash
vectos search --project web "checkout"
```

This searches the selected Nx project together with any included dependency roots in the logical scope.

`search` uses semantic retrieval first when the active index metadata matches the current embedding provider, then falls back to text search if semantic retrieval is unavailable or incompatible.

### Search Documentation

To search the documentation index (separate from source code):

```bash
vectos search --docs "getting started"
vectos search --project web --docs "authentication"
```

This uses the `--docs` flag to search `<project>-docs.db` instead of the source code database.

### Show index status

```bash
vectos status
```

To see documentation index status instead:

```bash
vectos status --docs
```

Inside an Nx workspace:

```bash
vectos status --project web
```

For Nx scopes, `status` also shows the resolved project roots so you can verify which app and libs were included.

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

For a detailed explanation of Nx logical scopes, roots, exclusions, and fallback behavior, see the `Nx Scope Model` section in [Indexing And Retrieval](indexing.md).

If commands do not behave as expected, also see [Troubleshooting](troubleshooting.md).
