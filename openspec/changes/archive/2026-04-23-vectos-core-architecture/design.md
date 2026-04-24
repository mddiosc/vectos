## Context

Current AI coding agents suffer from a "context gap". While tools like Engram provide session-based memory (decisions, preferences), they lack a deep, semantic understanding of the actual codebase. This gap leads to hallucinations, repetitive instructions, and an inability to reference specific code structures accurately.

## Goals / Non-Goals

**Goals:**
- Create a "Two-Brain" architecture: Session Memory (Engram) + Code Context (Vectos).
- Enable semantic search (search by meaning, not just keywords) over local codebases.
- Implement multi-project isolation so each project has its own dedicated vector context.
- Provide a standardized MCP interface for any agent to consume this knowledge.

**Non-Goals:**
- Building a full IDE or code editor.
- Replacing existing session memory tools like Engram (Vectos complements them).
- Providing a cloud-based solution (Vectos is strictly local-first).

## Decisions

- **Storage Engine**: SQLite with `sqlite-vss` (or similar vector extension). 
    - *Rationale*: Single-file portability, extreme reliability, and allows for a hybrid approach (SQL text search + Vector similarity search) in a single query.
- **Embedding Model**: `nomic-embed-text-v1.5` via local inference (LM Studio/LiteLLM).
    - *Rationale*: High performance, open-weights, and allows for a completely offline workflow.
- **Architecture Pattern**: MCP (Model Context Protocol) Server.
    - *Rationale*: Ensures interoperability with any modern agent (Claude Code, Opencode, etc.) and standardizes tool definitions.
- **Chunking Strategy**: Intelligent block-based chunking.
    - *Rationale*: Avoiding arbitrary character limits by attempting to respect code boundaries (functions, classes) to maintain semantic integrity.

## Risks / Trade-offs

- **Resource Consumption** $\rightarrow$ *Mitigation*: Use lightweight embedding models and optimized SQLite indexing to minimize CPU/RAM footprint.
- **Index Staleness** $\rightarrow$ *Mitigation*: Implement a file-watcher or manual trigger to re-index when code changes.
- **Connectivity Dependency** $\rightarrow$ *Mitigation*: Design the MCP server to degrade gracefully to pure text search if the embedding provider (LiteLLM) is unavailable.

## Migration Plan

1. **Phase 1 (Foundations)**: Build the core indexing engine and SQLite storage (Completed/In-Progress).
2. **Phase 2 (Intelligence)**: Integrate embeddings and implement hybrid search.
3. **Phase 3 (Integration)**: Implement the MCP server and CLI.
4. **Phase 4 (Refinement)**: Multi-project management and performance optimization.

## Open Questions

- Should we support automatic re-indexing via `inotify`/`fsevents` or keep it manual to save resources?
- How to handle extremely large repositories (e.g., monorepos) without overwhelming the local SQLite file?
