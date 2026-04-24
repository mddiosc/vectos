## Why

Current AI coding agents suffer from a "context gap": they possess session memory (decisions made) via tools like Engram, but lack deep, semantic access to the codebase structure itself. This leads to hallucinations and repetitive instructions. There is a critical need for a unified "Two-Brain" architecture that merges session-based memory with code-based context using local embeddings.

## What Changes

Implement a robust, local-first semantic context engine that acts as a "second brain" for coding agents.

- **New Capability: `code-indexing`**: A background process to scan, chunk, and embed local codebases.
- **New Capability: `semantic-search`**: A retrieval mechanism using local embeddings (nomic-embed-text-v1.5) to find code by meaning, not just keywords.
- **New Capability: `multi-project-context`**: Automated management of isolated SQLite databases per project directory to ensure context purity.
- **New Capability: `mcp-server`**: An MCP-compliant interface to expose these capabilities to any agent (Opencode, Claude Code, etc.).
- **NEW** **CLI Tool**: A command-line interface to manually trigger indexing, search, and database management.

## Capabilities

### New Capabilities
- `code-indexing`: Scans local directories, performs intelligent chunking (respecting code blocks), and generates embeddings.
- `semantic-search`: Performs vector similarity searches to retrieve relevant code snippets based on natural language queries.
- `project-isolation`: Automatically detects the current working directory and switches between dedicated SQLite vector databases.
- `mcp-interface`: Provides a standardized way for LLM agents to call `search_code` or `index_project` tools.

### Modified Capabilities
- `engram-integration`: Enhanced ability for agents to combine Engram session memories with Vectos code context in a single prompt.

## Impact

- **Codebase**: New `internal/` modules for indexer, storage, and MCP server.
- **Dependencies**: Addition of SQLite with vector extensions and embedding client libraries.
- **Architecture**: Transition from a simple script-based setup to a professional, multi-component service-oriented stack.
