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
	hybridCandidateLimit       = 25
	hybridResultLimitPerFile   = 2
	hybridDedupLineWindow      = 12
	hybridExactPhraseBoost     = 0.08
	hybridTokenOverlapWeight   = 0.18
	hybridFileNameBoost        = 0.06
	hybridActionableCodeBoost  = 0.04
	hybridFallbackBoost        = 0.12
	hybridSourcePathBoost      = 0.08
	hybridConfigIntentBoost    = 0.10
	hybridConfigPathBoost      = 0.08
	hybridApiRoutePenalty      = 0.05
	hybridDbIntentBoost        = 0.09
	hybridDbPathBoost          = 0.10
	hybridGenericConfigPenalty = 0.05
	hybridUiIntentBoost        = 0.08
	hybridSeoIntentBoost       = 0.08
	hybridSeoHeadBoost         = 0.10
	hybridSeoPagePenalty       = 0.04
	hybridFormIntentBoost      = 0.08
	hybridStateIntentBoost     = 0.08
	hybridAuthIntentBoost      = 0.08
	hybridDataIntentBoost      = 0.08
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

	if isBroadImplementationQuery(queryTokens) {
		if candidate.Category == "source" {
			score += hybridSourcePathBoost
		} else {
			score -= hybridBroadQueryPenalty
		}
	}

	if isRoutingQuery(queryTokens) {
		score += routingNameSignalBoost(candidate.FilePath)
	}

	if intentBoost := genericIntentBoost(queryTokens, candidate); intentBoost > 0 {
		score += intentBoost
	}

	if configBoost := configSpecificBoost(queryTokens, candidate); configBoost != 0 {
		score += configBoost
	}

	if dbBoost := databaseSpecificBoost(queryTokens, candidate); dbBoost != 0 {
		score += dbBoost
	}

	if seoBoost := seoSpecificBoost(queryTokens, candidate); seoBoost != 0 {
		score += seoBoost
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

func isRoutingQuery(queryTokens []string) bool {
	for _, token := range queryTokens {
		switch token {
		case "routing", "router", "routes", "route", "navigate", "navigation":
			return true
		}
	}
	return false
}

func routingNameSignalBoost(path string) float64 {
	best := 0.0
	for _, token := range tokenizePathForRanking(path + " " + filepath.Base(path)) {
		switch token {
		case "router", "routers":
			best = math.Max(best, 0.16)
		case "routes":
			best = math.Max(best, 0.12)
		case "navigation", "navigate", "navigations":
			best = math.Max(best, 0.10)
		case "route":
			best = math.Max(best, 0.03)
		}
	}
	return best
}

func genericIntentBoost(queryTokens []string, candidate storage.CodeChunk) float64 {
	if len(queryTokens) == 0 {
		return 0
	}

	pathTokens := tokenizePathForRanking(candidate.FilePath + " " + filepath.Base(candidate.FilePath))

	boosts := []struct {
		queryTokens     []string
		candidateTokens []string
		boost           float64
	}{
		{[]string{"config", "configure", "configured", "configuration", "setup", "initialize", "initialized", "client", "provider", "context", "api"}, []string{"config", "configure", "configuration", "setup", "init", "initialize", "client", "provider", "context", "api"}, hybridConfigIntentBoost},
		{[]string{"db", "database", "model", "models", "repository", "repo", "api", "client", "connection", "connect"}, []string{"db", "database", "model", "models", "repository", "repo", "api", "client", "connection", "connect", "service"}, hybridDbIntentBoost},
		{[]string{"ui", "layout", "theme", "navbar", "nav", "menu", "component", "page", "pages"}, []string{"ui", "layout", "theme", "navbar", "nav", "menu", "component", "page", "pages"}, hybridUiIntentBoost},
		{[]string{"seo", "meta", "metadata", "title", "description", "canonical", "og", "opengraph", "head"}, []string{"seo", "meta", "metadata", "title", "description", "canonical", "og", "opengraph", "head"}, hybridSeoIntentBoost},
		{[]string{"form", "forms", "contact", "submit", "validation", "input", "field", "recaptcha"}, []string{"form", "forms", "contact", "submit", "validation", "input", "field", "captcha", "recaptcha"}, hybridFormIntentBoost},
		{[]string{"state", "context", "provider", "reducer", "toggle", "hook", "theme", "menu"}, []string{"state", "context", "provider", "reducer", "toggle", "hook", "theme", "menu"}, hybridStateIntentBoost},
		{[]string{"auth", "jwt", "token", "login", "register", "user"}, []string{"auth", "jwt", "token", "login", "register", "user"}, hybridAuthIntentBoost},
		{[]string{"fetch", "load", "loaded", "sort", "sorted", "data", "database", "db", "work", "experience", "repo", "repository"}, []string{"fetch", "load", "sort", "data", "database", "db", "model", "repo", "repository"}, hybridDataIntentBoost},
	}

	best := 0.0
	for _, boost := range boosts {
		if hasAnyToken(queryTokens, boost.queryTokens...) && hasAnyToken(pathTokens, boost.candidateTokens...) {
			best = math.Max(best, boost.boost)
		}
	}
	return best
}

func seoSpecificBoost(queryTokens []string, candidate storage.CodeChunk) float64 {
	if !hasAnyToken(queryTokens, "seo", "meta", "metadata", "title", "description", "canonical", "og", "opengraph", "head") {
		return 0
	}

	pathTokens := tokenizePathForRanking(candidate.FilePath + " " + filepath.Base(candidate.FilePath))
	if hasAnyToken(pathTokens, "head", "document", "metadata", "seo") {
		return hybridSeoHeadBoost
	}
	if hasAnyToken(pathTokens, "page", "pages") && hasAnyToken(queryTokens, "head", "metadata", "seo", "title", "description") {
		return -hybridSeoPagePenalty
	}
	return 0
}

func configSpecificBoost(queryTokens []string, candidate storage.CodeChunk) float64 {
	if !hasAnyToken(queryTokens, "config", "configure", "configured", "configuration", "setup", "initialize", "initialized", "client", "provider", "api") {
		return 0
	}

	pathTokens := tokenizePathForRanking(candidate.FilePath + " " + filepath.Base(candidate.FilePath))
	if hasAnyToken(pathTokens, "config", "configure", "configuration", "setup", "init", "initialize") {
		return hybridConfigPathBoost
	}
	if hasAnyToken(pathTokens, "pages", "api") && !hasAnyToken(pathTokens, "config", "configuration") {
		return -hybridApiRoutePenalty
	}
	return 0
}

func databaseSpecificBoost(queryTokens []string, candidate storage.CodeChunk) float64 {
	if !hasAnyToken(queryTokens, "db", "database", "model", "models", "repository", "repo", "connection", "connect") {
		return 0
	}

	pathTokens := tokenizePathForRanking(candidate.FilePath + " " + filepath.Base(candidate.FilePath))
	if hasAnyToken(pathTokens, "db", "database", "model", "models", "repository", "repo", "connection", "connect") {
		return hybridDbPathBoost
	}
	if hasAnyToken(pathTokens, "config", "configuration") && !hasAnyToken(pathTokens, "db", "database", "model", "models") {
		return -hybridGenericConfigPenalty
	}
	return 0
}

func hasAnyToken(tokens []string, candidates ...string) bool {
	if len(tokens) == 0 || len(candidates) == 0 {
		return false
	}
	lookup := make(map[string]struct{}, len(tokens))
	for _, token := range tokens {
		lookup[token] = struct{}{}
	}
	for _, candidate := range candidates {
		if _, ok := lookup[candidate]; ok {
			return true
		}
	}
	return false
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
