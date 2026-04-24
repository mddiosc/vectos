## 1. Core Engine (Foundations)

- [x] 1.1 Implement `internal/storage` with SQLite and support for hybrid search
- [x] 1.2 Implement `internal/indexer` with intelligent chunking
- [x] 1.3 Create a CLI command to run the pipeline
- [x] 1.4 Add unit tests for chunking logic and storage persistence

## 2. Semantic Intelligence

- [x] 2.1 Integrate `embeddings.Embedder` interface (Abstraction)
- [x] 2.2 Implement `RemoteEmbedder` for LiteLLM/LM Studio integration
- [x] 2.3 Implement `internal/search` for semantic similarity queries (Hybrid Readiness)
- [x] 2.4 Integrate embeddings into the indexing pipeline (Vector storage)

## 3. MCP Integration & Multi-Project

- [x] 3.1 Implement "Project Awareness": auto-detect working directory and switch SQLite databases
- [x] 3.2 Implement MCP Server in `internal/mcp`
- [x] 3.3 Expose `search_code` and `index_project` as MCP tools
- [x] 3.4 Create `vectos setup <agent>` command to automate MCP configuration

## 4. Refinement & Polishing

- [x] 4.1 Implement `vectos status` to check DB size and index health
- [x] 4.2 Add support for multiple languages in chunking logic
- [x] 4.3 Final documentation and README update
