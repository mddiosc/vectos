## Why

The current semantic search in `SearchSemantic` performs an O(n) full-table cosine similarity scan over every chunk row, decoding each BLOB to `[]float32` in memory and computing dot products in pure Go. This is acceptable for hundreds or a few thousand chunks, but becomes prohibitively slow and memory-intensive beyond ~10K chunks — a realistic threshold for mid-sized projects. Vectos must scale its search to handle large codebases while maintaining its local-first, single-binary distribution model (no heavy C dependencies, no external services).

## What Changes

- **Vector index for approximate nearest neighbor search**: Replace the linear cosine scan with a persisted vector index (HNSW graph or similar ANN structure) stored alongside the SQLite DB, providing sub-linear search with configurable quality/speed trade-offs.
- **Batch embedding generation**: Extend the `Embedder` interface to support batch `GetEmbeddings([]string) ([][]float32, error)`, and batch chunk embeddings into groups of N during indexing to leverage ONNX Runtime batching.
- **Optional embedding compression**: Add product quantization or scalar quantization (SQ) support, configurable via `config.toml`, to reduce storage from 384 floats (1536 bytes) to ~96 bytes per vector with an opt-in quality trade-off.
- **Persist vector index to disk**: Store the index in a dedicated file (or SQLite table) so it survives `vectos serve` restarts without a full rebuild. Index metadata tracks whether the index is stale vs. the chunk table.

## Capabilities

### New Capabilities
- `vector-index`: Approximate nearest neighbor search indexed on disk, replacing the O(n) cosine scan with sub-linear lookup for semantic queries.
- `embedding-batching`: Batch interface on the embedder, allowing multiple texts to be embedded in one call to reduce per-chunk overhead during indexing.
- `embedding-compression`: Opt-in quantization of stored embeddings to reduce disk usage and memory pressure, with clear quality-vs-size configuration.

### Modified Capabilities
- `semantic-search`: The search pipeline changes to route through the vector index instead of a full table scan. The fallback to text search and hybrid ranking requirements are preserved, but the underlying retrieval mechanism is replaced.
- `embedding-provider`: The `Embedder` interface gains a batch method. Both the embedded (ONNX) and remote (HTTP) providers must implement it. The embedded provider must accept multiple tokens in one inference pass.
- `code-indexing`: The indexing pipeline shifts from single-chunk `GetEmbedding` calls to batched embedding of chunk groups. Chunk generation and embedding are decoupled into two phases: chunk all files first, then embed in batches.

## Impact

- **`internal/storage/`**: New vector index subsystem; `SearchSemantic` delegates to index lookup. Index metadata tracks staleness. New `index_metadata` columns or a separate index file.
- **`internal/embeddings/`**: `Embedder` interface extended with `GetEmbeddings([]string) ([][]float32, error)`. Both `EmbeddedEmbedder` (ONNX batching) and `RemoteEmbedder` updated.
- **`internal/indexer/`**: `SimpleChunker` decouples chunk creation from embedding; a new batching loop feeds chunks to the embedder in groups.
- **`internal/config/`**: New config keys for index type (HNSW parameters), batch size, and optional compression settings.
- **`cmd/vectos/`**: Index and serve commands aware of index file lifecycle. `vectos index` builds/rebuilds the vector index.
- **Dependencies**: A pure-Go vector index library (e.g., a minimal HNSW implementation vendored or via `github.com/coder/quartz` ecosystem, or a purpose-built Go package). Avoid CGo to preserve single-binary distribution.
