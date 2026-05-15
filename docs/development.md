# Development

## Architecture

Vectos currently uses this stack:

- Storage: SQLite
- Embeddings: embedded-first provider strategy
- Embedded runtime: in-process ONNX + local tokenizer
- Remote fallback: OpenAI-compatible embeddings API
- MCP server: `github.com/modelcontextprotocol/go-sdk/mcp`
- Index isolation: one database per active project

Default embedded configuration:

- Provider: `embedded`
- Model: `jina-embeddings-v3` (1024-dim, code+text+multilingual, 8192-token context)
- Cache directory: `~/.vectos/models/jina-embeddings-v3/`
- Supported models: `jina-embeddings-v3`, `bge-small-en-v1.5`

Default fallback configuration:

- Provider order: `embedded`, then `remote`
- Remote provider: disabled by default
- Remote model: `text-embedding-nomic-embed-text-v1.5`
- Remote base URL: user-provided when enabled

## Embedding Configuration

Embedding configuration is loaded from `~/.vectos/config.json`.

Example:

```json
{
  "embeddings": {
    "default_provider": "embedded",
    "fallback_order": ["embedded", "remote"],
    "embedded": {
      "enabled": true,
      "model_name": "jina-embeddings-v3",
      "model_dir": "/Users/you/.vectos/models/jina-embeddings-v3",
      "auto_download": true,
      "asset_base_url": "https://huggingface.co/jinaai/jina-embeddings-v3/resolve/main",
      "timeout_seconds": 60,
      "batch_size": 32
    },
    "remote": {
      "enabled": true,
      "base_url": "http://localhost:4000/v1",
      "model": "text-embedding-nomic-embed-text-v1.5",
      "timeout_seconds": 30
    },
    "vector_index": {
      "index_type": "hnsw",
      "hnsw_m": 16,
      "hnsw_ef_construction": 200,
      "hnsw_ef_search": 200,
      "compression": "none"
    }
  }
}
```

Notes:

- `embedded.enabled: false` cleanly disables the local provider.
- `remote.enabled: false` disables remote fallback.
- `remote.base_url` is intentionally not hardcoded by default; set it to your own OpenAI-compatible endpoint only if you want remote fallback.
- `fallback_order` controls provider resolution explicitly.
- `default_provider` controls which provider Vectos tries first.
- `batch_size` controls how many texts are embedded in a single ONNX inference call (default 32).
- `vector_index` controls the approximate nearest neighbor index (see Vector Index Configuration below).

## Vector Index Configuration

The `vector_index` section in `~/.vectos/config.json` controls the HNSW approximate nearest neighbor index used for semantic search.

| Key | Default | Description |
|-----|---------|-------------|
| `index_type` | `hnsw` | Index type (`hnsw` or empty for linear scan) |
| `hnsw_m` | 16 | Max connections per node in the HNSW graph |
| `hnsw_ef_construction` | 200 | Search depth during index construction (higher = better quality, slower build) |
| `hnsw_ef_search` | 200 | Search depth during query (higher = better recall, slower query) |
| `compression` | `none` | Embedding compression: `none` or `sq8` (4x storage reduction, lossy) |

The vector index is built automatically during `vectos index` and persisted as a `.vectorindex` file alongside the SQLite database. It is rebuilt when the content hash changes or when the index file is missing.

SQ8 compression quantizes 32-bit float vectors to 8-bit integers, reducing storage from 1536 bytes/vector (384-dim) or 4096 bytes/vector (1024-dim) to roughly 1/4 of that. Recall typically remains above 80% but may degrade for some queries.

## Embedded Model Cache

The default embedded provider manages its own local cache under `~/.vectos/models/`.

For `jina-embeddings-v3`, Vectos automatically downloads and caches:

- `config.json`
- `tokenizer.json`
- `model.onnx`
- a platform-specific ONNX Runtime shared library, normalized locally as:
  - `onnxruntime.dylib` on macOS
  - `onnxruntime.so` on Linux

Vectos downloads model assets from Hugging Face and extracts the ONNX Runtime shared library from the official Microsoft release for the current platform.

To switch to the older model, set `model_name` to `bge-small-en-v1.5` in the embedded config section. Vectos will automatically sync `model_dir` and `asset_base_url` to match the selected model.

## Local Build

```bash
go build -o vectos ./cmd/vectos
```

Minimal helper:

```bash
make build
```

## Relevant Project Files

- `cmd/vectos/main.go` — CLI, MCP server, setup automation
- `cmd/vectos/commands_serve.go` — serve command with watch mode and HTTP server
- `internal/indexer/chunker.go` — chunking and semantic enrichment
- `internal/storage/sqlite.go` — storage, stats, semantic ranking
- `internal/storage/project_manager.go` — project-aware database routing
- `internal/vectorindex/` — HNSW index, SQ8 compression, binary persistence
- `internal/server/` — HTTP API (search, metrics, status, reindex)
- `internal/watcher/` — filesystem watcher for serve mode
- `internal/config/embedding.go` — embedding and vector index configuration
- `AGENTS.md` — local guidance for agents using this repository
