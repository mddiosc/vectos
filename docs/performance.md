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

## Related Tests

- `internal/storage/performance_test.go`
- `internal/storage/indexed_files_test.go`
- `cmd/vectos/commands_index_test.go`
- `cmd/vectos/smoke_e2e_test.go`
