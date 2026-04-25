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
- Model: `bge-small-en-v1.5`
- Cache directory: `~/.vectos/models/bge-small-en-v1.5/`

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
      "model_name": "bge-small-en-v1.5",
      "model_dir": "/Users/you/.vectos/models/bge-small-en-v1.5",
      "auto_download": true,
      "asset_base_url": "https://huggingface.co/BAAI/bge-small-en-v1.5/resolve/main",
      "timeout_seconds": 60
    },
    "remote": {
      "enabled": true,
      "base_url": "http://localhost:4000/v1",
      "model": "text-embedding-nomic-embed-text-v1.5",
      "timeout_seconds": 30
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

## Embedded Model Cache

The default embedded provider manages its own local cache under `~/.vectos/models/`.

For `bge-small-en-v1.5`, Vectos automatically downloads and caches:

- `config.json`
- `tokenizer.json`
- `model.onnx`
- a platform-specific ONNX Runtime shared library, normalized locally as:
  - `onnxruntime.dylib` on macOS
  - `onnxruntime.so` on Linux

Vectos downloads model assets from Hugging Face and extracts the ONNX Runtime shared library from the official Microsoft release for the current platform.

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
- `internal/indexer/chunker.go` — chunking and semantic enrichment
- `internal/storage/sqlite.go` — storage, stats, semantic ranking
- `internal/storage/project_manager.go` — project-aware database routing
- `AGENTS.md` — local guidance for agents using this repository
