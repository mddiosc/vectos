## 1. Embedding Batching

- [x] 1.1 Extend `Embedder` interface with `GetEmbeddings(texts []string) ([][]float32, error)` method, providing a default loop-based implementation for backward compatibility
- [x] 1.2 Implement batched ONNX inference in `EmbeddedEmbedder` — tokenize all texts, pad to max sequence length, run single inference with shape `[N, max_seq_len]`, extract and normalize each output vector
- [x] 1.3 Implement batched request in `RemoteEmbedder` — send all texts in a single POST with all `input` entries
- [x] 1.4 Add `batch_size` configuration key (default 32) to embedding config
- [x] 1.5 Update indexing pipeline to group chunks into batches of at most `batch_size` before calling the batch embedder
- [x] 1.6 Write unit tests for batch embedding: empty batch, single-text batch, configurable batch size, ONNX output parity with single-call

## 2. Vector Index — Core HNSW Implementation

- [x] 2.1 Create `internal/vectorindex/` package with module structure
- [x] 2.2 Implement HNSW node and graph data types (node id, vector, neighbor lists per layer, max level)
- [x] 2.3 Implement HNSW graph construction algorithm (layer assignment, greedy insertion with `ef_construction`, neighbor pruning)
- [x] 2.4 Implement HNSW search algorithm (multi-layer traversal, candidate set expansion with `ef_search`, top-K return)
- [x] 2.5 Implement cosine distance metric (`1 - cosine_similarity`) for graph edges and search ranking
- [x] 2.6 Add configurable parameters: `M` (max neighbors per node), `ef_construction` (build-time exploration), `ef_search` (query-time exploration, default 100)
- [x] 2.7 Write unit tests for HNSW: construction correctness, search recall >95% on random 384-dim vectors, cosine distance ordering

## 3. Vector Index — Persistence

- [x] 3.1 Design binary file format for `.vectorindex`: version header, chunk table content hash, HNSW layer count, node vectors, neighbor edge lists
- [x] 3.2 Implement index serialization — write complete HNSW graph to binary file
- [x] 3.3 Implement index deserialization — load HNSW graph from binary file on startup
- [x] 3.4 Implement chunk table staleness detection: compute hash of chunk table metadata on index build, compare on load
- [x] 3.5 Write unit tests for serialization round-trip, staleness detection, version mismatch handling

## 4. Search Pipeline Integration

- [x] 4.1 Modify `SearchSemantic` to route queries through the vector index when available instead of full-table cosine scan
- [x] 4.2 Implement fallback logic: if index file missing or stale, fall back to existing linear scan with a warning log
- [x] 4.3 Preserve text-search fallback: when both index and linear scan fail, fall back to text-based search
- [x] 4.4 Add vector index configuration keys: `index_type`, `hnsw_m`, `hnsw_ef_construction`, `hnsw_ef_search`

## 5. Embedding Compression

- [x] 5.1 Implement 8-bit scalar quantization (SQ8): compute min/max per dimension, encode each float32 → int8, decode int8 → float32
- [x] 5.2 Add `compression` configuration key (values: `none`, `sq8`) under vector index config
- [x] 5.3 Integrate SQ8 encoding into index build path when compression is enabled
- [x] 5.4 Integrate SQ8 decoding into index search path to reconstruct vectors before distance computation
- [x] 5.5 Write unit tests for SQ8: encode/decode round-trip, query recall impact within acceptable threshold

## 6. CLI Integration

- [x] 6.1 Integrate index building as final step of `vectos index` command — after all chunks are stored in SQLite, build HNSW graph and serialize to `<db>.vectorindex`
- [x] 6.2 Show progress feedback during index build (chunks loaded, graph construction progress)
- [x] 6.3 Integrate index loading on `vectos serve` startup — load `.vectorindex` if valid, warn and fall back to linear scan if missing or stale

## 7. Testing & Validation

- [x] 7.1 Write integration tests for end-to-end semantic search using the vector index (index → load → query → verify results)
- [x] 7.2 Write performance benchmarks comparing index-based search vs linear scan across different chunk counts (1K, 10K, 50K)
- [x] 7.3 Test index staleness: modify chunk table after index build, verify fallback to linear scan
- [x] 7.4 Test index rebuild: run `vectos index` twice, verify old index is overwritten
- [x] 7.5 Test batch + compression + index in combination: build index with SQ8 from batched embeddings, verify search quality
