## 1. Model Registry & Config

- [x] 1.1 Add `jina-embeddings-v3` entry to `embeddedModelAssets` map in `internal/embeddings/embedded.go` with asset specs pointing to `jinaai/jina-embeddings-v3` on HuggingFace (model.onnx, tokenizer.json, config.json)
- [x] 1.2 Add `DefaultJinaAssetBaseURL` constant and change `DefaultEmbeddedModel` from `"bge-small-en-v1.5"` to `"jina-embeddings-v3"` in `internal/config/embedding.go`
- [x] 1.3 Update `DefaultEmbeddingConfig()` — uses `DefaultEmbeddedModel` dynamically, model dir auto-reflects jina-v3
- [x] 1.4 Add validation: ensure `model_name` in config is one of the supported models (`bge-small-en-v1.5` or `jina-embeddings-v3`)

## 2. Runtime Dimension Detection

- [x] 2.1 Verify `detectEmbeddingSize()` in `embedded.go` correctly reads dimensions from jina ONNX output metadata (auto-detects 1024 from Dims[2])
- [x] 2.2 Update `ProviderStatus` initialization — Dimensions overridden by ONNX detection at session creation (line 451-453)
- [x] 2.3 `DefaultEmbeddedDimensions = 384` kept as fallback; actual dimensions come from ONNX metadata
- [x] 2.4 Handle sequence length detection: `detectSequenceLength()` auto-detects 8192 from ONNX input Dims[1]

## 3. Tokenizer & Sequence Length

- [x] 3.1 jina tokenizer.json uses standard HuggingFace format, compatible with `sugarme/tokenizer` library
- [x] 3.2 `defaultSequenceLength` kept at 512 as safe fallback; actual value detected from ONNX metadata at runtime
- [~] 3.3 Test tokenization with large code chunk — requires actual model download (manual verification)

## 4. Index Metadata & Stale Detection

- [x] 4.1 `RequiresReindex()` correctly compares provider, model, and dimensions from stored index metadata vs current embedder
- [x] 4.2 Reindex warning message includes specific model name and dimension details
- [~] 4.3 End-to-end stale-index test requires actual model indices (manual verification)

## 5. Asset Download & Validation

- [x] 5.1 Verified jina-v3 ONNX model available at `https://huggingface.co/jinaai/jina-embeddings-v3/resolve/main/onnx/model.onnx`
- [x] 5.2 `allowedDownloadContentTypes` already covers octet-stream, gzip, x-gzip, x-tar — sufficient for HuggingFace CDN
- [~] 5.3 Auto-download end-to-end requires fresh install (manual verification)
- [~] 5.4 Existing bge-small dir coexistence (manual verification)

## 6. Search Integration

- [x] 6.1 `executeSearch()` works with any dimensions — HNSW index is dimension-agnostic
- [x] 6.2 `trySemanticSearch()` correctly embeds queries and searches with detected dimensions
- [x] 6.3 Docs search (`executeSearchDocs`) uses same pipeline

## 7. Tests

- [x] 7.1 Add unit test: jina model spec entry exists in `embeddedModelAssets` with correct asset paths
- [x] 7.2 Add unit test: default model is jina-v3, bge-small still accepted as valid model name
- [x] 7.3 Add unit test: `detectEmbeddingSize` fallback returns `DefaultEmbeddedDimensions`
- [x] 7.4 Add unit test: `RequiresReindex` returns true when stored provider/model/dimensions differs from current
- [~] 7.5 Integration test with actual index requires model download (manual)
- [x] 7.6 Run existing test suite — 204/204 pass, no regressions

## 8. Documentation

- [x] 8.1 Update `docs/indexing.md` to mention the code-aware model as default
- [x] 8.2 Update `README.md` embedding section to reflect jina as default model
- [x] 8.3 Update config examples
