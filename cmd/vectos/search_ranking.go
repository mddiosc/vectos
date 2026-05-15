package main

import (
	"math"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"vectos/internal/storage"
)

const (
	hybridCandidateLimit      = 10
	hybridResultLimitPerFile   = 2
	hybridDedupLineWindow      = 12
	hybridExactPhraseBoost     = 0.08
	hybridTokenOverlapWeight   = 0.18
	hybridFileNameBoost        = 0.06
	hybridActionableCodeBoost  = 0.04
	hybridFallbackBoost        = 0.12
	hybridBroadQueryPenalty    = 0.12
	hybridBuildArtifactPenalty = 0.25
	hybridTestFilePenalty      = 0.08
	hybridHelpTextPenalty      = 0.10
)

var tokenPattern = regexp.MustCompile(`[a-z0-9]+`)
var camelBoundaryPattern = regexp.MustCompile(`([a-z0-9])([A-Z])`)

type rankedChunk struct {
	chunk storage.CodeChunk
	score float64
}

func rerankHybridResults(query string, candidates []storage.CodeChunk, limit int) []storage.CodeChunk {
	if len(candidates) == 0 {
		return nil
	}

	queryTokens := tokenizeForRanking(query)
	ranked := make([]rankedChunk, 0, len(candidates))
	for _, candidate := range candidates {
		ranked = append(ranked, rankedChunk{
			chunk: candidate,
			score: computeHybridScore(query, queryTokens, candidate),
		})
	}

	sort.SliceStable(ranked, func(i, j int) bool {
		if math.Abs(ranked[i].score-ranked[j].score) < 0.000001 {
			return ranked[i].chunk.Score > ranked[j].chunk.Score
		}
		return ranked[i].score > ranked[j].score
	})

	filtered := dedupeRankedResults(ranked, limit)
	results := make([]storage.CodeChunk, 0, len(filtered))
	for _, rankedChunk := range filtered {
		rankedChunk.chunk.Score = rankedChunk.score
		results = append(results, rankedChunk.chunk)
	}
	return results
}

func computeHybridScore(query string, queryTokens []string, candidate storage.CodeChunk) float64 {
	score := candidate.Score
	score = applyContentBoosts(score, query, queryTokens, candidate)
	score = applyPenalties(score, queryTokens, candidate)
	return score
}

func applyContentBoosts(score float64, query string, queryTokens []string, candidate storage.CodeChunk) float64 {
	contentLower := strings.ToLower(candidate.Content)
	pathLower := strings.ToLower(filepath.ToSlash(candidate.FilePath))
	pathTokens := tokenizePathForRanking(candidate.FilePath)
	baseTokens := tokenizePathForRanking(filepath.Base(candidate.FilePath))
	contentTokens := tokenizeForRanking(contentLower)

	if queryLower := strings.ToLower(strings.TrimSpace(query)); queryLower != "" {
		if strings.Contains(contentLower, queryLower) || strings.Contains(pathLower, queryLower) {
			score += hybridExactPhraseBoost
		}
	}

	if overlap := tokenOverlapRatio(queryTokens, append(append(pathTokens, baseTokens...), contentTokens...)); overlap > 0 {
		score += overlap * hybridTokenOverlapWeight
	}

	if fileNameOverlap := tokenOverlapRatio(queryTokens, baseTokens); fileNameOverlap > 0 {
		score += fileNameOverlap * hybridFileNameBoost
	}

	if looksActionableCode(candidate) {
		score += hybridActionableCodeBoost
	}

	if looksLikeSemanticFallback(candidate) {
		score += hybridFallbackBoost
	}

	return score
}

func applyPenalties(score float64, queryTokens []string, candidate storage.CodeChunk) float64 {
	pathLower := strings.ToLower(filepath.ToSlash(candidate.FilePath))

	if isBroadImplementationQuery(queryTokens) && candidate.Category != "source" {
		score -= hybridBroadQueryPenalty
	}

	if isTestFilePath(pathLower) {
		score -= hybridTestFilePenalty
	}

	if looksLikeHelpText(candidate) {
		score -= hybridHelpTextPenalty
	}

	if isBuildArtifactPath(pathLower) {
		score -= hybridBuildArtifactPenalty
	}

	return score
}

func dedupeRankedResults(ranked []rankedChunk, limit int) []rankedChunk {
	if limit <= 0 {
		limit = len(ranked)
	}

	result := make([]rankedChunk, 0, limit)
	perFile := make(map[string]int)
	for _, candidate := range ranked {
		path := candidate.chunk.FilePath
		if perFile[path] >= hybridResultLimitPerFile {
			continue
		}
		if overlapsSelectedCandidate(candidate, result) {
			continue
		}
		result = append(result, candidate)
		perFile[path]++
		if len(result) == limit {
			break
		}
	}
	return result
}

func overlapsSelectedCandidate(candidate rankedChunk, selected []rankedChunk) bool {
	for _, existing := range selected {
		if existing.chunk.FilePath != candidate.chunk.FilePath {
			continue
		}
		if rangesOverlapOrTouch(existing.chunk.StartLine, existing.chunk.EndLine, candidate.chunk.StartLine, candidate.chunk.EndLine, hybridDedupLineWindow) {
			return true
		}
	}
	return false
}

func rangesOverlapOrTouch(aStart, aEnd, bStart, bEnd, window int) bool {
	if aStart > bEnd+window {
		return false
	}
	if bStart > aEnd+window {
		return false
	}
	return true
}

func looksActionableCode(candidate storage.CodeChunk) bool {
	content := strings.TrimSpace(candidate.Content)
	if candidate.Category == "source" {
		return true
	}
	for _, prefix := range []string{"func ", "export function ", "function ", "class ", "export class ", "type ", "const use", "function use"} {
		if strings.HasPrefix(content, prefix) {
			return true
		}
	}
	return false
}

func isTestFilePath(path string) bool {
	base := filepath.Base(path)
	return strings.HasSuffix(base, "_test.go") || strings.Contains(path, ".test.") || strings.Contains(path, "/test/") || strings.Contains(path, "\\test\\")
}

func looksLikeHelpText(candidate storage.CodeChunk) bool {
	content := strings.ToLower(candidate.Content)
	base := strings.ToLower(filepath.Base(candidate.FilePath))
	if strings.Contains(base, "help") || strings.Contains(base, "usage") {
		return true
	}
	if strings.Contains(content, "usage:") && strings.Contains(content, "examples:") {
		return true
	}
	if strings.Count(content, "fmt.println(") >= 3 || strings.Count(content, "fmt.printf(") >= 3 {
		return true
	}
	return false
}

func looksLikeSemanticFallback(candidate storage.CodeChunk) bool {
	content := strings.ToLower(candidate.Content)
	return strings.Contains(content, "searchsemantic(") && strings.Contains(content, "searchtext(")
}

func isBroadImplementationQuery(queryTokens []string) bool {
	if len(queryTokens) > 4 {
		return false
	}
	for _, token := range queryTokens {
		switch token {
		case "docs", "documentation", "config", "setup", "test", "tests", "readme":
			return false
		}
	}
	return len(queryTokens) > 0
}

func isBuildArtifactPath(path string) bool {
	for _, marker := range []string{"/dist/", "/coverage/", "/build/", "/.next/", "/playwright-report/", "/test-results/"} {
		if strings.Contains(path, marker) {
			return true
		}
	}
	return false
}

func tokenOverlapRatio(queryTokens []string, candidateTokens []string) float64 {
	if len(queryTokens) == 0 || len(candidateTokens) == 0 {
		return 0
	}
	seen := make(map[string]struct{}, len(candidateTokens))
	for _, token := range candidateTokens {
		seen[token] = struct{}{}
	}
	matches := 0
	querySeen := map[string]struct{}{}
	for _, token := range queryTokens {
		if _, done := querySeen[token]; done {
			continue
		}
		querySeen[token] = struct{}{}
		if _, ok := seen[token]; ok {
			matches++
		}
	}
	return float64(matches) / float64(len(querySeen))
}

func tokenizeForRanking(input string) []string {
	input = strings.ToLower(input)
	parts := tokenPattern.FindAllString(input, -1)
	if len(parts) == 0 {
		return nil
	}
	tokens := make([]string, 0, len(parts))
	for _, part := range parts {
		if len(part) < 2 || isStopToken(part) {
			continue
		}
		tokens = append(tokens, part)
	}
	return tokens
}

func tokenizePathForRanking(input string) []string {
	normalized := camelBoundaryPattern.ReplaceAllString(input, `$1 $2`)
	normalized = strings.NewReplacer("_", " ", "-", " ").Replace(normalized)
	return tokenizeForRanking(normalized)
}

func isStopToken(token string) bool {
	switch token {
	case "the", "and", "for", "with", "that", "this", "from", "into", "only", "part", "flow", "code":
		return true
	default:
		return false
	}
}
