## Context

The current MCP integration proves that agents can call Vectos, but the next step is to make those calls feel natural and productive. Agents need enough context to decide whether to read a file, retry with a different query, or refresh an index, without forcing the user to manually interpret sparse tool output.

## Goals / Non-Goals

**Goals:**
- Make MCP search responses easier for agents to act on directly
- Return richer metadata that helps an agent choose what to inspect next
- Make stale or missing index situations easier to recover from inside MCP workflows
- Keep the tool surface compact rather than exploding into many narrow tools

**Non-Goals:**
- Building a fully conversational assistant layer inside MCP responses
- Adding many overlapping MCP tools before the core ergonomics are validated
- Making Vectos dependent on a specific agent client or prompt format

## Decisions

### Enrich existing tools before adding many new ones

The first step should improve `search_code` and `index_project` outputs rather than multiplying tools. Better defaults are likely to deliver more value than a broader but thinner API surface.

### Return concise explanatory metadata

Useful MCP result metadata includes file path, line range, chunk role, score or rank, and a short explanation of why the result matched. This helps agents reason about next actions without reading large irrelevant files.

### Make recovery guidance explicit for missing or stale indexes

When search cannot proceed because the project is not indexed or appears stale, the tool output should make the recovery path obvious. Agents should not have to infer whether they need to index, reindex, or reformulate the query.

### Preserve compatibility with current MCP clients

Changes should fit normal MCP tool-result patterns and avoid exotic response structures that would make adoption harder.

## Risks / Trade-offs

- Too much metadata could bloat tool responses -> Mitigation: keep result metadata concise and rank-focused
- Explanatory text could become noisy or misleading -> Mitigation: keep explanations templated and grounded in actual ranking signals
- Better ergonomics in MCP could diverge from CLI output -> Mitigation: share result formatting logic where reasonable
