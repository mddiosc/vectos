## 1. Model Registry & Config

- [ ] 1.1 Add `jina-embeddings-v2-base-code` entry to `embeddedModelAssets` map in `internal/embeddings/embedded.go` with asset specs pointing to `jinaai/jina-embeddings-v2-base-code` on HuggingFace (model.onnx, tokenizer.json, config.json)
- [ ] 1.2 Add `DefaultJinaAssetBaseURL` constant and change `DefaultEmbeddedModel` from `"bge-small-en-v1.5"` to `"jina-embeddings-v2-base-code"` in `internal/config/embedding.go`
- [ ] 1.3 Update `DefaultEmbeddingConfig()` to reflect new default model name and asset URL in `internal/config/embedding.go`
- [ ] 1.4 Add validation: ensure `model_name` in config is one of the supported models (`bge-small-en-v1.5` or `jina-embeddings-v2-base-code`)

## 2. Runtime Dimension Detection

- [ ] 2.1 Verify `detectEmbeddingSize()` in `embedded.go` correctly reads dimensions from jina ONNX output metadata (should detect 768)
- [ ] 2.2 Update `ProviderStatus` initialization to use detected dimensions from ONNX model instead of hardcoded `DefaultEmbeddedDimensions = 384`
- [ ] 2.3 Update `DefaultEmbeddedDimensions` or remove it — dimensions should always come from the loaded model's ONNX metadata
- [ ] 2.4 Handle sequence length detection: ensure `detectSequenceLength()` works for jina model (should detect 8192)

## 3. Tokenizer & Sequence Length

- [ ] 3.1 Verify that the jina tokenizer.json is compatible with the `sugarme/tokenizer` library used in `internal/embeddings/embedded.go`
- [ ] 3.2 Update `defaultSequenceLength` from 512 to 8192 when using the jina model (or make it model-specific)
- [ ] 3.3 Test tokenization with a large code chunk (>512 tokens) to verify no truncation or panic

## 4. Index Metadata & Stale Detection

- [ ] 4.1 Verify that `RequiresReindex()` in `internal/storage/sqlite.go` correctly compares provider, model, and dimensions from stored index metadata vs current embedder
- [ ] 4.2 Ensure reindex warning message includes the specific model name and dimension mismatch details
- [ ] 4.3 Test: index with bge-small → switch config to jina → search → verify stale-index warning appears

## 5. Asset Download & Validation

- [ ] 5.1 Verify jina model assets are downloadable from `https://huggingface.co/jinaai/jina-embeddings-v2-base-code/resolve/main/`
- [ ] 5.2 Update or verify `allowedDownloadContentTypes` includes any new Content-Types the jina CDN might return
- [ ] 5.3 Test auto-download flow end-to-end: fresh install → `vectos index` → model downloads → index completes
- [ ] 5.4 Test with existing bge-small model dir: switch config to jina → jina downloads to its own model directory

## 6. Search Integration

- [ ] 6.1 Verify `executeSearch()` in `cmd/vectos/search_exec.go` works with 768-dim vectors (HNSW index is dimension-agnostic)
- [ ] 6.2 Verify `trySemanticSearch()` correctly embeds queries with the jina model and searches with correct dimensions
- [ ] 6.3 Verify docs search (`executeSearchDocs`) works with the new model

## 7. Tests

- [ ] 7.1 Add unit test: jina model spec entry exists in `embeddedModelAssets` with correct asset paths
- [ ] 7.2 Add unit test: default model is jina, bge-small still accepted as valid model name
- [ ] 7.3 Add unit test: dimension detection from ONNX mock returns 768 for jina-like output shapes
- [ ] 7.4 Add unit test: `RequiresReindex` returns true when stored provider/model differs from current
- [ ] 7.5 Add integration test: index small project with default config, verify 768-dim embeddings stored
- [ ] 7.6 Run existing test suite to verify no regressions in embedding, indexing, or search

## 8. Documentation

- [ ] 8.1 Update `docs/indexing.md` to mention the code-aware model as default
- [ ] 8.2 Update `README.md` embedding section to reflect jina as default model
- [ ] 8.3 Update `config.json` examples and comments (if any)
