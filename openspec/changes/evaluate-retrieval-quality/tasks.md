## 1. Benchmark Format And CLI

- [x] 1.1 Define a lightweight benchmark fixture format for representative retrieval queries and expected useful targets
- [x] 1.2 Add a CLI entry point to run retrieval benchmarks against an indexed project
- [x] 1.3 Validate benchmark input and return actionable errors for malformed fixtures or missing project indexes

## 2. Evaluation Execution And Reporting

- [x] 2.1 Execute benchmark queries through the normal Vectos retrieval pipeline
- [x] 2.2 Compute per-query top-window hit results and aggregate hit-rate metrics
- [x] 2.3 Produce readable benchmark output that shows expected targets, returned results, and aggregate summaries

## 3. Validation And Seed Benchmarks

- [x] 3.1 Add tests for benchmark parsing and hit-rate computation
- [x] 3.2 Create an initial benchmark set from real representative queries used during Vectos validation
- [x] 3.3 Run `go build ./...` and confirm the evaluation workflow is usable in normal local iteration
