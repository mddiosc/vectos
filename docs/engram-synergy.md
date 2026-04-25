# Optional Engram Synergy

Vectos is a standalone code-context product. It does not require Engram to index projects, search code, or expose MCP tools.

When Engram is also available, the two tools complement each other well:

- Engram provides prior session memory
- Vectos provides current code context

Use them together when you want an agent to understand both what was learned before and what exists in the codebase now.

## Standalone First

Use Vectos by itself when you want:

- semantic code retrieval
- project-scoped indexing
- MCP-based code search tools for agents
- a local-first code context engine without session-memory dependencies

Typical standalone flow:

1. index the project
2. search for relevant code
3. read only the matching files or chunks

Example:

```bash
vectos index .
vectos search "checkout payment flow"
```

## Combined Workflow With Engram

When both tools are installed, the recommended workflow is:

1. recover prior memory with Engram
2. retrieve current code with Vectos
3. read only the focused files that Vectos surfaces
4. save new discoveries back to Engram if the session produces durable learnings

Conceptually:

```text
1. mem_context / mem_search
   -> what we already know

2. vectos_search_code
   -> where the relevant code lives now

3. targeted file reads
   -> confirm behavior in the current codebase

4. mem_save
   -> preserve new findings for future sessions
```

## Recommended Agent Guidance

When both systems are available, a good default agent workflow is:

```text
If prior project context may matter, check session memory first.

Then use Vectos to locate the most relevant current code before falling back to broad grep, glob, or direct file reads.

Use direct search and file reads as a fallback when Vectos does not return useful results or when exact string matching is required.

If the project is not yet indexed or the Vectos results are stale, run vectos_index_project and retry.

When a session produces a durable decision, bugfix, or pattern, save it to memory separately.
```

## Non-Goals

This workflow does not imply:

- a shared runtime between Vectos and Engram
- a shared database
- a requirement that Engram must be installed for Vectos to work
- any promise that Vectos itself manages long-term session memory

## Summary

Think of the pairing this way:

- Vectos answers: "Where is the relevant code now?"
- Engram answers: "What did we already learn before?"

They work well together, but Vectos remains complete and useful on its own.
