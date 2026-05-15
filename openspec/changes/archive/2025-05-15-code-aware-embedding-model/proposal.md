## Why

The current embedding model **BAAI/bge-small-en-v1.5** is a general-purpose English text model (384 dims) not trained on code. Benchmarks against a React/TypeScript project show ~60% precision: the model cannot distinguish TypeScript syntax patterns (`interface`, `type`, hooks, generics) from natural language text. Switching to a code-aware embedding model will give the single largest precision improvement (~20-30%) with minimal code changes.

## What Changes

- Add the **jina-embeddings-v2-base-code** model as a new embedded model option alongside bge-small
- Make the code-aware model the **new default** for indexing and search
- Support auto-download of jina model assets (model.onnx, tokenizer.json, config.json) from HuggingFace
- Preserve bge-small as a fallback/conservative option configurable via `config.json`
- **BREAKING**: Existing indexes using bge-small embeddings will be detected as stale (provider/model mismatch) and require reindexing

## Capabilities

### New Capabilities

- `code-embedding-model`: Add support for a code-aware embedding model that produces higher-quality vectors for code search, understanding TypeScript/JavaScript/Go syntax patterns in the embedding space.

### Modified Capabilities

- `embedding-provider`: The embedded provider config in `~/.vectos/config.json` gains a new valid model name (`jina-embeddings-v2-base-code`) with its own asset_base_url. The model selection logic must handle different model dimensions (768 vs 384) and different asset layouts. The default model changes from `bge-small-en-v1.5` to `jina-embeddings-v2-base-code`.
- `code-indexing`: Index metadata must track the embedding model per-project so that provider/model mismatch detection works correctly when switching models. Existing indexes using the old model must be flagged for reindex.

## Impact

- **Affected code**: `internal/embeddings/embedded.go` (new model spec, ONNX session), `internal/config/embedding.go` (default model, validation), `internal/storage/sqlite.go` (provider metadata tracking), `internal/indexer/chunker.go` (no changes needed — same chunking pipeline)
- **Affected config**: `~/.vectos/config.json` `embeddings.embedded.model_name` and `embeddings.embedded.asset_base_url`
- **Dependencies**: ONNX Runtime (already used), jina model assets from HuggingFace (~130MB for ONNX model, ~3MB for tokenizer)
- **User impact**: Requires a one-time reindex (`vectos index --force`) after upgrading. All existing users get the code-aware model by default
