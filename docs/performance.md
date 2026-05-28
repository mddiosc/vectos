# Performance Guide

This guide records the pre-v1.0 validation metrics gathered for Vectos indexing and retrieval on a 100k-embedding stress shape.

## Stress Scenario

- Dataset shape: `100000` embeddings
- Embedding dimension: `8`
- Search limit: `10`
- Storage path: project-scoped SQLite database with WAL enabled
- Streaming path under test: `SQLiteStorage.ForEachEmbedding()`
- Search path under test: fallback `searchLinearScan(...)` with a result limit

## Measured Results

These metrics were measured locally on macOS (`darwin`) using the same scenario covered by `TestStreamingStress_100kEmbeddingsMemoryStable`.

| Operation | Duration | Peak heap delta |
| --- | ---: | ---: |
| Insert 100k embeddings | `~2.22 s` | not sampled in the standalone metrics run |
| Stream 100k embeddings with `ForEachEmbedding()` | `~33.89 ms` | `~3.47 MiB` |
| Semantic search fallback (`searchLinearScan`, top 10) | `~100.09 ms` | `~3.44 MiB` |

## Test Thresholds

The repository stress test enforces memory ceilings so regressions fail fast in CI:

- streaming heap delta must stay below `64 MiB`
- fallback semantic search heap delta must stay below `96 MiB`

Observed usage stayed far below both limits in the measured 100k run.

## Smoke Validation Summary

The real CLI smoke flow for pre-v1.0 validation is covered by `TestCLIRealWorldSmokeFlow` and uses a temporary project plus a mock remote embedding server.

Validated flow:

1. `vectos index <project>`
2. `vectos search "checkout-token"`
3. edit the indexed file
4. `vectos index --changed app.go <project>`
5. `vectos search "refund-token"`
6. `vectos status`

The smoke test also verifies:

- updated content replaces stale content in SQLite
- the vector index still loads after incremental reindexing
- `PRAGMA journal_mode` reports `wal`

## Hardware Acceleration

Vectos uses ONNX Runtime for embedded embedding inference. By default, inference runs on CPU. Acceleration providers can be enabled via environment variables to leverage GPU/NPU hardware.

### CoreML (macOS Apple Silicon)

Enabled by default on macOS. Offloads inference to the Apple Neural Engine, GPU, or AMX coprocessor via the CoreML execution provider.

| Variable | Default | Effect |
|----------|---------|--------|
| `VECTOS_COREML=0` | enabled | Disable CoreML, force CPU-only inference |

**Speedup on Apple Silicon:** ~10–50× vs CPU-only for jina-embeddings-v3.

### CUDA (Linux/Windows with NVIDIA GPU)

Opt-in. Requires a CUDA-capable ONNX Runtime build (`onnxruntime-gpu`), NVIDIA drivers, and CUDA toolkit installed.

| Variable | Default | Effect |
|----------|---------|--------|
| `VECTOS_CUDA=1` | disabled | Enable CUDA GPU acceleration on device 0 |

**Important:** The default ONNX Runtime shared library distributed with Vectos does NOT include CUDA. You must provide a CUDA-capable build via `ONNX_RUNTIME_LIBRARY_PATH` or place the `libonnxruntime.so` in the library search path. Without a CUDA-enabled runtime, enabling `VECTOS_CUDA=1` has no effect — ONNX silently falls back to CPU.

**Future:** `VECTOS_CUDA_DEVICE` for multi-GPU setups.

### Fallback Behavior

All acceleration providers are non-fatal: if a provider fails to initialize (missing library, unsupported hardware, etc.), ONNX Runtime automatically falls back to CPU inference. No error is surfaced to the user.

### ONNX Session Optimizations

Vectos applies these optimizations in addition to acceleration providers:

| Optimization | Value | Effect |
|--------------|-------|--------|
| Graph optimization level | `ORT_ENABLE_ALL` (99) | Constant folding, node fusion, redundant node elimination |
| Intra-op threads | `runtime.NumCPU()` | Parallelizes individual operations across all CPU cores |

These apply to all platforms and work even when no acceleration provider is active.

## Related Tests

- `internal/storage/performance_test.go`
- `internal/storage/indexed_files_test.go`
- `cmd/vectos/commands_index_test.go`
- `cmd/vectos/smoke_e2e_test.go`
