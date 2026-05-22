package main

import (
	"sort"
	"strings"

	"vectos/internal/storage"
)

const (
	rrfConstant           = 40.0 // lower k = more weight to top-ranked results
	rrfVectorLimit        = 35
	rrfKeywordLimit       = 15
	rrfFinalLimit         = 10
	rrfResultLimitPerFile = 2
)

// fuseResults fuses vector and keyword search results using Reciprocal Rank Fusion.
// k is the RRF constant (default 60). Returns up to limit fused results.
func fuseResults(vectorResults, keywordResults []storage.CodeChunk, k float64, limit int) []storage.CodeChunk {
	if limit <= 0 {
		limit = rrfFinalLimit
	}

	// Handle edge cases: one or both lists empty
	if len(vectorResults) == 0 && len(keywordResults) == 0 {
		return nil
	}
	if len(vectorResults) == 0 {
		return limitResults(keywordResults, limit)
	}
	if len(keywordResults) == 0 {
		return limitResults(vectorResults, limit)
	}

	// Compute RRF scores: score = 1/(k + rank) summed across both lists
	rrfScores := make(map[int64]float64)
	chunkMap := make(map[int64]storage.CodeChunk)

	for rank, chunk := range vectorResults {
		rrfScores[chunk.ID] += 1.0 / (k + float64(rank) + 1)
		if _, exists := chunkMap[chunk.ID]; !exists {
			chunkMap[chunk.ID] = chunk
		}
	}

	for rank, chunk := range keywordResults {
		rrfScores[chunk.ID] += 1.0 / (k + float64(rank) + 1)
		if _, exists := chunkMap[chunk.ID]; !exists {
			chunkMap[chunk.ID] = chunk
		}
	}

	// Sort by RRF score descending
	type rrfEntry struct {
		id    int64
		score float64
	}
	entries := make([]rrfEntry, 0, len(rrfScores))
	for id, score := range rrfScores {
		entries = append(entries, rrfEntry{id: id, score: score})
	}
	sort.SliceStable(entries, func(i, j int) bool {
		return entries[i].score > entries[j].score
	})

	// Build result list with scores
	results := make([]storage.CodeChunk, 0, limit)
	for _, entry := range entries {
		chunk := chunkMap[entry.id]
		chunk.Score = entry.score
		results = append(results, chunk)
		if len(results) >= limit {
			break
		}
	}
	return results
}

// applyFusionPenalties applies structural penalties to fused results.
// Only test file, build artifact, and help text penalties remain post-fusion.
// Content-matching boosts are NOT applied — BM25 keyword search handles that natively.
func applyFusionPenalties(results []storage.CodeChunk) []storage.CodeChunk {
	for i := range results {
		pathLower := strings.ToLower(results[i].FilePath)

		if isTestFilePath(pathLower) {
			results[i].Score *= 0.05 // test files are noise for most queries
		}
		if isImportPrelude(results[i]) {
			results[i].Score *= 0.4
		}
		if isBuildArtifactPath(pathLower) {
			results[i].Score *= 0.2
		}
		if looksLikeHelpText(results[i]) {
			results[i].Score *= 0.5
		}
	}
	// Re-sort after penalties — a penalized top hit may now rank below an
	// unpenalized one that was originally second.
	sort.SliceStable(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})
	return results
}

// dedupeByFile limits results to maxPerFile entries per file path and removes
// overlapping chunks from the same file. Returns up to limit results.
func dedupeByFile(results []storage.CodeChunk, limit, maxPerFile int) []storage.CodeChunk {
	if limit <= 0 {
		limit = len(results)
	}
	if maxPerFile <= 0 {
		maxPerFile = 2
	}

	out := make([]storage.CodeChunk, 0, limit)
	perFile := make(map[string]int)
	for _, c := range results {
		path := c.FilePath
		if perFile[path] >= maxPerFile {
			continue
		}
		if overlapsSelected(c, out) {
			continue
		}
		out = append(out, c)
		perFile[path]++
		if len(out) >= limit {
			break
		}
	}
	return out
}

func overlapsSelected(candidate storage.CodeChunk, selected []storage.CodeChunk) bool {
	for _, existing := range selected {
		if existing.FilePath != candidate.FilePath {
			continue
		}
		if rangesOverlapOrTouch(existing.StartLine, existing.EndLine, candidate.StartLine, candidate.EndLine, 12) {
			return true
		}
	}
	return false
}
