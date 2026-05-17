---
name: vectos
description: Semantic codebase search and indexing. Use when searching code, exploring project structure, finding files, or looking up implementations. Prefer Vectos tools over grep/glob/find for codebase exploration. Also use when grep/glob return too many or too few results.
origin: Vectos
---

# Vectos — Semantic Codebase Search

Vectos provides code-aware semantic search over your project via MCP tools. It indexes source code into project-scoped SQLite databases with embeddings for hybrid (vector + keyword) retrieval via Reciprocal Rank Fusion.

## When to Activate

Load this skill when:
- Searching for code, implementations, or patterns across the codebase
- Exploring an unfamiliar project or directory structure
- grep/glob return too many irrelevant results or miss semantically relevant code
- Working in an Nx monorepo and need dependency-aware scoping
- Need to index a project before searching
- Troubleshooting Vectos search results (stale, missing, irrelevant)

## Delegation Protocol (Critical)

When you (the orchestrator) delegate to specialist agents that perform code search (**@explorer**, **@fixer**, **@oracle**), you MUST explicitly instruct them to use Vectos tools before grep/glob. Sub-agents do not automatically receive the AGENTS.md guidance — they rely on your delegation instructions.

### Required Delegation Template

```
When searching for code, use vectos_search_code first. 
Only fall back to grep/glob when Vectos returns no useful results 
or when exact string/pattern matching is needed.
If you get IDX_MISSING, run vectos_index_project and retry.
```

### Example: Delegating to @explorer

```
@explorer: Find where authentication middleware is implemented.
Use vectos_search_code for semantic matches first.
Fall back to grep only if Vectos returns no results.
```

### Example: Delegating to @fixer

```
@fixer: Update the error handling in the payment service.
Before editing, use vectos_search_code to find all related error-handling patterns.
Use grep only for exact string replacement verification.
```

## Core Workflow

### 1. Index the Project

```
vectos_index_project()
```

First search in a session? Run this first. Vectos requires indexed code to return results. Subsequent searches in the same session don't need re-indexing unless files changed.

### 2. Search Code

```
vectos_search_code(query="authentication middleware jwt")
```

Use natural language queries. Vectos understands semantics, not just keywords. Describe what you're looking for, not just function names.

**Good queries:**
- "where is authentication middleware defined"
- "error handling pattern in payment service"
- "react component that renders user profile"

**Bad queries:**
- "auth" (too broad)
- "function xyz" (use grep for exact names)

### 3. Search Documentation

```
vectos_search_docs(query="getting started setup instructions")
```

For README files, ADRs, API docs, and other documentation. Requires separate docs index:

```
vectos_index_project(docs: true)
```

### 4. Incremental Refresh After Edits

After creating, moving, or editing files, refresh the index incrementally:

```
vectos_index_project(changed: "src/auth/login.ts,src/auth/middleware.ts")
```

Only do a full reindex when the affected scope is broad or uncertain.

## Tool Reference

### vectos_search_code

| Param | Type | Description |
|-------|------|-------------|
| `query` | string (required) | Natural language search query |
| `path` | string | Optional project path to scope the search |
| `project` | string | Nx project name (Nx workspaces only) |

**Result guidance values:**
- `IDX_MISSING` → Project not indexed. Run `vectos_index_project` and retry.
- `IDX_STALE` → Index needs refresh. Run `vectos_index_project` or incremental refresh.

### vectos_search_docs

Same params as `search_code`, but searches documentation only. Requires docs index via `index_project(docs: true)`.

### vectos_index_project

| Param | Type | Description |
|-------|------|-------------|
| `path` | string | Path to a file or directory to index |
| `project` | string | Nx project name |
| `changed` | string | Comma-separated changed file paths for incremental refresh |
| `docs` | boolean | If true, index only documentation files |

### vectos_list_projects

Lists available Nx project names. Use before searching/indexing in an Nx workspace.

## Nx Monorepo Workflow

```
# 1. Discover projects
vectos_list_projects()

# 2. Index a specific project (includes internal dependencies)
vectos_index_project(project: "app-main")

# 3. Search scoped to that project
vectos_search_code(query="shared button component", project: "app-main")
```

When you edit a shared lib, changes are reflected in all consuming projects. If downstream searches feel stale, refresh the downstream project index.

## CLI Fallback

When MCP is unavailable or the agent client doesn't support it, use the Vectos CLI:

```bash
# Index
vectos index .

# Search code
vectos search "authentication middleware"

# Search docs
vectos index . --docs
vectos search --docs "setup guide"

# Nx scoping
vectos index --project app-main .
vectos search --project app-main "shared component"

# Incremental refresh
vectos index --changed src/auth/login.ts,src/auth/middleware.ts
```

**When to prefer CLI over MCP:**
- Agent client has no MCP support
- MCP server fails to start or times out
- You're in a CI/CD pipeline or script
- Quick one-off search outside a long agent session

**When NOT to use CLI:**
- MCP is available and working
- You need structured, typed results
- You're composing searches with other tool calls

## Troubleshooting

### IDX_MISSING

The project hasn't been indexed. Run `vectos_index_project()` first. For documentation, use `docs: true`.

### IDX_STALE

The index needs refreshing. Run `vectos_index_project()` or use `changed` with the specific files you modified.

### No useful results

1. Check the query — is it semantic enough? Don't use single keywords.
2. Verify the file type is supported (check Project Status in README).
3. Try broadening the query scope.
4. Fall back to grep for exact string/pattern matching.

### Results feel stale after editing

Use incremental refresh:
```
vectos_index_project(changed: "path/to/edited/file.ts")
```

### MCP server not responding

1. Check `vectos mcp` runs without errors.
2. Run `vectos setup opencode` to re-validate the MCP entry.
3. Fall back to CLI via bash tool.

## Anti-Patterns

### Don't use Vectos when:

- **You already know the exact file path** — just read it directly.
- **You need exact string matching** — use grep (e.g., finding all callers of `getCwd`, exact regex patterns).
- **You're searching for something in a single, known file** — use read + search within the file.
- **The project is tiny (<10 files)** — direct reads are faster.

### Don't full-reindex unnecessarily:

```diff
- vectos_index_project()  # Full reindex after editing 1 file
+ vectos_index_project(changed: "src/auth/login.ts")  # Incremental refresh
```

### Don't mix code and docs in one search:

```diff
- vectos_search_code(query="README setup instructions")  # READMEs are docs
+ vectos_search_docs(query="setup instructions")
```

## Graceful Degradation

If Vectos consistently fails (index won't build, MCP down, unsupported file types):

1. Try CLI fallback first (`vectos search "query"`).
2. If CLI also fails, use grep/glob as usual.
3. Report the issue — see [troubleshooting](docs/troubleshooting.md).
