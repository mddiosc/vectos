## Context

Vectos is a local-first, single-binary semantic code search tool. Embeddings are 384-dimensional float32 vectors produced by an ONNX runtime running bge-small-en-v1.5. Storage is SQLite. The current `SearchSemantic` method does a full-table scan over `code_chunks`, deserializes every embedding BLOB to `[]float32`, and computes cosine similarity in pure Go — O(n) in both time and memory. For a project with 10K chunks, this loads ~15MB of vectors and runs 10K dot products per query. At 100K chunks it becomes unusable.

The project must remain distributable as a single Go binary without C dependencies (CGo avoided where possible). The embedding runtime (ONNX) already carries native shared libraries; adding another C dependency (FAISS) would complicate cross-platform builds.

## Goals / Non-Goals

**Goals:**
- Sub-linear semantic search: O(log n) or O(k log n) approximate nearest neighbor lookup instead of O(n) scan
- Persist the vector index to disk so `vectos serve` starts with an existing index
- Batch embedding generation to reduce per-chunk ONNX session overhead during indexing
- Opt-in embedding compression to reduce storage for large projects
- Maintain single-binary distribution (no system-level C dependencies beyond what ONNX already requires)

**Non-Goals:**
- Exact nearest neighbor search (ANN trade-off is acceptable)
- Real-time index updates on chunk insertion (batch rebuild is acceptable for this release)
- Online learning or index tuning based on query patterns
- GPU acceleration for indexing or search
- Multi-node distributed indices

## Decisions

### Decision 1: Build a minimal pure-Go HNSW index

**Chosen**: Implement a minimal HNSW (Hierarchical Navigable Small World) index in `internal/vectorindex/` as a pure-Go package with cosine distance support.

**Alternatives considered**:
- **FAISS via CGo (`github.com/DataIntelligenceCrew/go-faiss`)**: Industry-standard, but requires compiling FAISS C++ library. Violates single-binary constraint and adds complex cross-platform build matrix. Rejected.
- **`github.com/james-bowman/nlp`**: Provides LSA/LDA but not general ANN. No HNSW. Rejected.
- **`github.com/coder/quartz` ecosystem**: No existing vector index library. Rejected.
- **`sqlite-vec` extension**: Requires compiling C extension into SQLite or loading as a runtime extension. Adds C dependency and platform-specific shared library loading. Rejected.
- **`github.com/sjy-dv/nnv`**: Pure-Go vector DB library. Potentially usable but pulls in many sub-dependencies for a full vector DB when we only need the index. Rejected in favor of a focused implementation.

**Rationale**: HNSW is well-documented (Malkov & Yashunin, 2016), the algorithm is ~300-400 lines of Go for a single-layer graph with construction and search. We control the implementation, avoid dependency bloat, and keep the binary self-contained. Cosine distance is computed as `1 - cosine_similarity` within the index.

### Decision 2: Separate index file alongside SQLite

**Chosen**: Store the HNSW graph in a binary file (`<db-path>.vectorindex`) in the same directory as the SQLite database. The file format is a simple binary serialization of nodes (id, vector, neighbors).

**Alternatives considered**:
- **SQLite BLOB table**: Would work but adds SQLite overhead for graph traversal (one query per node to fetch neighbors). The index needs random access to node vectors and neighbor lists during search, which is faster via direct file I/O or mmap.
- **Embedded in `code_chunks` table**: Muddles chunk storage with search index. Rejected for separation of concerns.

**Rationale**: A dedicated binary file allows the index to be mmap'd for fast random access during search. The file is rebuilt on `vectos index` and read on `vectos serve`. A hash of the chunk table is stored in the index header to detect staleness.

### Decision 3: Batch embedding via interface extension

**Chosen**: Add a `GetEmbeddings(texts []string) ([][]float32, error)` method to the `Embedder` interface. Provide a default implementation in the interface that calls `GetEmbedding` in a loop for backward compatibility.

**Implementation**:
- `EmbeddedEmbedder`: Use ONNX dynamic batching — pad tokenized sequences to the max length in the batch, run a single inference with shape `[batch_size, max_seq_len]`.
- `RemoteEmbedder`: Send all texts in the `Input` array of the request body (already supported by the API schema).

**Rationale**: The ONNX runtime is optimized for batched inference. Single-chunk embeddings incur per-call overhead (tokenization + session.Run). Batching 32 chunks in one call reduces the overhead by ~32x during indexing.

### Decision 4: Scalar quantization as first compression method

**Chosen**: Implement 8-bit scalar quantization (SQ8) as the opt-in compression. Each float32 dimension is mapped to an int8, reducing storage from 1536 bytes/vector to 384 bytes (4x reduction). A more aggressive 4-bit variant (96 bytes) can be added later.

**Alternatives considered**:
- **Product quantization (PQ)**: Better compression ratios but requires training codebooks on the dataset. Adds indexing-time cost and Go implementation complexity.
- **Binary quantization**: Extreme compression (48 bytes/vector) but too lossy for 384-dim code embeddings.

**Rationale**: SQ8 is lossy but preserves ranking for top-K retrieval well in practice. It's trivial to implement (min/max per dimension → uniform quantization). It can be enabled per-index via config and does not require training data.

### Decision 5: Index lifecycle tied to `vectos index`

**Chosen**: The vector index is built as the final step of `vectos index` (after all chunks are inserted into SQLite) and loaded on `vectos serve` startup.

**Flow**:
1. `vectos index`: Chunk files → batch-embed chunks → store chunks + embeddings in SQLite → build HNSW graph → serialize to `<db>.vectorindex`
2. `vectos serve`: Load `<db>.vectorindex` on startup. If missing or stale (chunks changed), serve falls back to linear scan with a warning.

**Rationale**: Keeps index building out of the hot path (search). The index is a derived artifact from the chunk table, rebuilt on explicit reindex. Incremental updates (add/remove single chunks) are deferred to a future iteration.

## Risks / Trade-offs

- **[Approximate results]**: HNSW returns approximate nearest neighbors, not exact top-K. Some high-similarity chunks may be missed. → Mitigation: Configurable `ef_search` parameter (higher = more accurate, slower). Default `ef_search=100` gives >95% recall on 384-dim vectors.
- **[Index rebuild cost]**: Full index rebuild on every `vectos index`. For 100K chunks, HNSW construction is ~30-60 seconds. → Mitigation: Acceptable for a batch indexing tool. Show progress feedback.
- **[Memory during indexing]**: All vectors must be in memory to build the HNSW graph. 100K × 384 × 4 = ~150MB, manageable. → Mitigation: Document memory requirements. For extreme cases (>1M chunks), add streaming build later.
- **[Two copies of vectors]**: Embeddings exist in both SQLite `code_chunks.embedding` column and in the vector index file. → Mitigation: The index file stores only vectors and graph edges, not chunk metadata. The 2x storage is acceptable given the performance gain. Compression mitigates this.
- **[Binary compatibility]**: The `.vectorindex` file format must be stable across Vectos versions. → Mitigation: Version header in the file. If version mismatch, Vectos warns and falls back to linear scan, suggesting reindex.

## Open Questions

- **HNSW construction parallelism**: Can layer assignment and graph building be parallelized in Go? Measure single-thread performance first; optimize if >5s for <50K vectors.
- **Incremental index updates**: Should `vectos index --changed file1.go` rebuild only the affected portion? Defer to a follow-up change; full rebuild is simpler and correct for now.
- **Index memory mapping**: Should the index use `mmap` for zero-copy loading? Evaluate trade-off between load time and complexity. Start with standard `os.ReadFile` and optimize if loading >200ms.
