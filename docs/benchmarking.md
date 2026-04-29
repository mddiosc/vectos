# Retrieval Benchmark

This report compares Vectos against a simple `rg`-based baseline for the same intent-driven queries.

Method:

- Reindex the current repository first.
- Run `vectos benchmark benchmarks/retrieval/token-efficiency.json`.
- Measure baseline output with `rg -n` using equivalent search intent.
- Estimate tokens as `output_bytes / 4`.

## Results

| Query                         | Vectos tokens | Vectos files | First response | rg tokens | rg files | Verdict |
| ----------------------------- | ------------: | -----------: | -------------- | --------: | -------: | ------- |
| Fallback to text search       |           404 |            5 | Yes            |       425 |        7 | Vectos  |
| Docs and metadata exclusion   |           354 |            5 | Partial        |       396 |        5 | Vectos  |
| TypeScript and React chunking |           402 |            4 | No             |       458 |        2 | Vectos  |
| Hybrid ranking logic          |           397 |            3 | Yes            |       295 |        3 | rg      |
| Nx logical scope roots        |           408 |            3 | Yes            |      1298 |        9 | Vectos  |

## Readout

- Vectos now hits the right result within the top 3 for all 5 benchmark queries.
- Vectos now wins on raw token output in 4 of 5 queries.
- The biggest win comes from compact search output plus an adaptive preview, not from changing the underlying retrieval model.
- The strongest result is Nx scope resolution, where Vectos returns the right area with far fewer tokens than `rg`.
- The remaining weak spot is the hybrid-ranking query, where `rg` is still shorter.

## Benchmark Fixture

The reusable fixture lives at `benchmarks/retrieval/token-efficiency.json`.

## MCP Payload Check

Using the same adaptive preview heuristic inside MCP, a representative 3-result payload
measured about `1242 bytes` in this repo for each of the 5 benchmark queries.
That is the payload shape agents actually consume, so the compact preview path matters
there at least as much as in the CLI.

### Payload Growth

| Results | Payload bytes |
| ------: | ------------: |
|       3 |          1242 |
|       5 |          1966 |
|      10 |          3818 |

The payload scales linearly with result count, so keeping the default search window at 5
is a sensible balance for MCP. Ten results roughly doubles the payload relative to five.

## mywebsite Check

Benchmarking the larger Next.js project at `/Users/mddiosc/develop/personal/mywebsite` gave:

| Query                   | Vectos tokens | Vectos files | rg tokens | rg files | Verdict |
| ----------------------- | ------------: | -----------: | --------: | -------: | ------- |
| Mobile menu toggle      |           353 |            5 |       586 |       12 | Vectos  |
| Work experience loading |           416 |            4 |      1002 |       15 | Vectos  |
| GitHub API client       |           378 |            5 |       339 |        9 | rg      |
| Navbar links            |           383 |            4 |       603 |       12 | Vectos  |
| JWT helpers             |           398 |            5 |       773 |       12 | Vectos  |

Readout:

- Vectos hit top-3 and top-5 on all 5 queries after the ranking adjustments.
- Vectos stayed smaller than `rg` on 4 of 5 queries and was close on the GitHub API client query.
- The strongest win was the work-experience query, where `rg` had much more noise.
