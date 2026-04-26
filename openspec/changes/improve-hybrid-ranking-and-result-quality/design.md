## Context

Vectos is already useful, but current retrieval can still produce top results that are semantically related yet not the most actionable entry points for an agent. Phase 2 should improve not only whether something relevant appears somewhere in the list, but whether the top few results are the best places to start reading.

## Goals / Non-Goals

**Goals:**
- Improve top-ranked result quality for real code-navigation queries
- Blend semantic and exact-match signals when useful
- Reduce repeated or redundant results from the same file or neighboring chunks
- Keep ranking behavior understandable enough to debug with the new evaluation workflow

**Non-Goals:**
- Building a full search engine with complex learning-to-rank infrastructure
- Guaranteeing perfect ranking for every repository shape or programming language
- Replacing semantic retrieval with pure keyword search

## Decisions

### Use a staged hybrid ranking approach

The simplest reliable model is a staged pipeline: gather semantic candidates, compute text-aware and structural boosts, then rerank. This keeps the main architecture close to the existing system while making improvements traceable.

### Prefer lightweight boosts over opaque ranking logic

Useful early boosts include exact symbol mentions, file-name relevance, chunk-role relevance, and strong text overlap. These are easier to reason about than an opaque learned ranking layer and fit Vectos' experimental-but-practical posture.

### Deduplicate neighboring or near-identical results

Top results lose value when several entries point to nearly the same file region or chunk family. The ranking pipeline should collapse or down-rank redundant candidates so the first few results cover more useful alternatives.

### Keep semantic fallback intact

Hybrid ranking should strengthen normal retrieval, not make the system brittle. If hybrid features are unavailable or do not improve ranking, Vectos must preserve current semantic-plus-text fallback behavior.

## Risks / Trade-offs

- Too many heuristics could make ranking hard to reason about -> Mitigation: keep boosts explicit and validate against benchmarks
- Deduplication could hide genuinely distinct useful chunks -> Mitigation: only collapse highly overlapping or near-identical candidates
- File-name or symbol boosts could over-favor exact-match queries -> Mitigation: keep semantic similarity as the base ranking signal
