# Retrieval Benchmark

This report compares Vectos against traditional agent tools for the same intent-driven queries.

**Current retrieval stack**: jina-embeddings-v3 (1024-dim, code+text+multilingual) + HNSW vector index + BM25 text search fused via Reciprocal Rank Fusion (RRF).

## Multi-Tool Benchmark (Latest)

A comprehensive benchmark comparing Vectos against **all tools available to AI coding agents** (grep, glob, ast-grep, Read file, Read directory) on 10 natural-language queries against a React/TypeScript project with Tailwind CSS, i18n, routing, and theme support.

For the full analysis with per-query breakdowns, see [Token Efficiency Analysis](token-efficiency.md).

### Summary

| Tool | Avg Tokens/Query | Can Answer Alone? | Total Workflow Cost |
|------|----------------:|-----------------:|-------------------:|
| **Vectos** | **~139** | Often | **~489** |
| grep | ~5,183 | Never | ~8,183 |
| glob | ~299 | Never | ~3,299 |
| ast-grep | ~248 | Sometimes | ~1,748 |
| Read file | ~3,397 | Yes (must know path) | ~3,397 |

### Key Ratios

| Comparison | Ratio |
|-----------|------:|
| Vectos vs grep (search only) | **37×** fewer tokens |
| Vectos vs grep (full workflow) | **17×** fewer tokens |
| Vectos vs glob+read workflow | **7×** fewer tokens |
| Vectos vs ast-grep+read workflow | **4×** fewer tokens |
| Best case (Tailwind + generic terms) | **56×** fewer tokens vs grep |
| Worst case (unique specific terms) | **3.5×** fewer tokens vs grep |

### Method

- Execute 10 natural-language queries with each tool against the same project.
- For grep: decompose query into reasonable keywords (`rg "keyword1|keyword2|..."`).
- For glob: use filename patterns (`**/*{keyword1,keyword2}*`).
- For ast-grep: use structural patterns (`useTheme($$$)`, `lazy($$$)`, `<Link $$$>`).
- For Read file: measure the byte size of files that correctly answer the query.
- Estimate tokens as `output_bytes / 4`.

---

## Self-Benchmark (Vectos Repository)

Vectos benchmarked against itself using the fixture at `benchmarks/retrieval/token-efficiency.json`:

| Query                         | Vectos tokens | Vectos files | First response | rg tokens | rg files | Verdict |
| ----------------------------- | ------------: | -----------: | -------------- | --------: | -------: | ------- |
| Fallback to text search       |           404 |            5 | Yes            |       425 |        7 | Vectos  |
| Docs and metadata exclusion   |           354 |            5 | Partial        |       396 |        5 | Vectos  |
| TypeScript and React chunking |           402 |            4 | No             |       458 |        2 | Vectos  |
| Hybrid ranking logic          |           397 |            3 | Yes            |       295 |        3 | rg      |
| Nx logical scope roots        |           408 |            3 | Yes            |      1298 |        9 | Vectos  |

### Readout

- Vectos hits the right result within the top 3 for all 5 benchmark queries.
- Vectos wins on raw token output in 4 of 5 queries.
- The biggest win comes from compact search output plus an adaptive preview, not from changing the underlying retrieval model.
- The strongest result is Nx scope resolution, where Vectos returns the right area with far fewer tokens than `rg`.
- The remaining weak spot is the hybrid-ranking query, where `rg` is still shorter.
- **Note**: This self-benchmark shows modest gains (~1.2×) because the Vectos codebase is small and uses specific terms. The multi-tool benchmark above shows the real-world advantage on larger projects with CSS frameworks and generic terminology.

## Retrieval Stack Evolution

| Version | Embedding Model | Search Method | Key Improvement |
|---------|----------------|---------------|-----------------|
| v0.1.x | bge-small-en-v1.5 (384-dim) | Linear scan cosine | Baseline retrieval |
| v0.1.7–0.1.9 | bge-small-en-v1.5 | Hybrid semantic + text reranking | Better ranking heuristics |
| v0.6.0 | jina-embeddings-v3 (1024-dim) | HNSW + BM25 + RRF | Faster search, better code-aware embeddings |

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

## Larger Project Validations

### Next.js Project (Vectos vs rg only)

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

### React/TypeScript + Tailwind Project (All tools)

Full multi-tool benchmark with 10 queries comparing Vectos, grep, glob, ast-grep, and Read file. See [Token Efficiency Analysis](token-efficiency.md) for complete per-query data.

Key findings:
- Vectos advantage grows dramatically with CSS utility frameworks (Tailwind `dark:*` classes cause 335 grep matches for a "dark mode" query)
- Generic terms ("error", "form", "test") produce 50× more grep tokens than Vectos
- ast-grep is precise but narrow — cannot answer conceptual questions
- glob is cheap but blind — always requires follow-up reads
- Vectos' signatures and hints eliminate ~50% of follow-up file reads
