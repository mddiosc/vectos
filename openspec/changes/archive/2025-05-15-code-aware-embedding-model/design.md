## Context

Vectos currently ships with a single embedded model: **BAAI/bge-small-en-v1.5** (384 dimensions, 512 token limit). This is a general-purpose English text embedding model. Benchmarks show it achieves ~60% precision on TypeScript/React code search because its embedding space doesn't capture programming language syntax or structural patterns.

The code-aware alternative **jina-embeddings-v2-base-code** (jinaai/jina-embeddings-v2-base-code) is a 768-dimension model trained on code, supporting 8192 tokens. It's available on HuggingFace with ONNX weights.

The existing `embeddedModelAssets` map in `internal/embeddings/embedded.go` maps a single model name to its asset specs. The `embedded.go` file uses ONNX Runtime with hardcoded 384-dimension expectations in some paths. The config layer (`internal/config/embedding.go`) defaults to bge-small.

## Goals / Non-Goals

**Goals:**
- Add jina-embeddings-v2-base-code as a supported embedded model alongside bge-small
- Make jina the new default model
- Support auto-download of jina model assets from HuggingFace
- Handle different model dimensions (768) correctly through the entire pipeline
- Detect provider/model mismatch for existing indexes to flag reindex need

**Non-Goals:**
- Supporting arbitrary/custom ONNX models (still fixed set)
- Changing the chunking strategy
- Supporting multiple models simultaneously per index
- Remote provider changes

## Decisions

### Decision 1: Add jina as a new entry in the model registry, not a separate code path

The `embeddedModelAssets` map already supports multiple models by name. We add a "jina-embeddings-v2-base-code" entry with its own asset specs pointing to `jinaai/jina-embeddings-v2-base-code` on HuggingFace.

**Alternative considered**: Create a separate EmbeddedEmbedder subtype for code models. Rejected because the ONNX inference pipeline is identical — only model weights, tokenizer, and dimensions differ. A registry-based approach keeps the codebase simpler.

### Decision 2: Make jina the default, keep bge-small as fallback

`DefaultEmbeddedModel` changes from `"bge-small-en-v1.5"` to `"jina-embeddings-v2-base-code"`. Users who want the old model set `model_name: "bge-small-en-v1.5"` in config. This is a **BREAKING** change — existing indexes will be flagged as stale and require reindex.

**Alternative considered**: Keep bge-small default, let users opt into jina. Rejected because the whole point is improving search quality for all users. The one-time reindex cost is acceptable.

### Decision 3: Auto-detect dimensions from ONNX model outputs, not hardcode

Currently `DefaultEmbeddedDimensions = 384` is used in `ProviderStatus` initialization. The code already has `detectEmbeddingSize()` that reads dimensions from ONNX output metadata — but it's only called after session creation. The status should use the detected value once available, not the hardcoded default.

### Decision 4: No code changes to chunking or search

The chunker (`buildSemanticContent`) produces the same enriched text regardless of model. The search pipeline (`executeSearch`, `trySemanticSearch`) works with whatever dimensions the embedder provides. Dimensions are stored per-chunk in the DB already. The HNSW index is dimension-agnostic.

## Risks / Trade-offs

- **[Model size]** jina ONNX model is ~130MB vs bge-small ~45MB → mitigated by auto-download and caching in `~/.vectos/models/`
- **[Inference latency]** jina has 2x dimensions (768 vs 384) → ONNX inference is already fast (<10ms per chunk on Apple Silicon); batch embedding amortizes cost
- **[Backward compatibility]** Existing indexes become stale → users must reindex; the stale-index warning pathway already exists and will handle this automatically
- **[HuggingFace availability]** jina model must be downloadable → same download infrastructure as bge-small; if download fails, bge-small remains as configured fallback
- **[ONNX compatibility]** jina model may require specific ONNX opset → tested during implementation; same ONNX Runtime 1.25.0 works for BGE, likely compatible
