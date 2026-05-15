package main

import (
	"path/filepath"
	"regexp"
	"strings"

	"vectos/internal/storage"
)

var tokenPattern = regexp.MustCompile(`[a-z0-9]+`)
var camelBoundaryPattern = regexp.MustCompile(`([a-z0-9])([A-Z])`)

// --- Structural penalty helpers (used by search_fusion.go) ---

func isTestFilePath(path string) bool {
	base := filepath.Base(path)
	return strings.HasSuffix(base, "_test.go") || strings.Contains(path, ".test.") || strings.Contains(path, "/test/") || strings.Contains(path, "\\test\\")
}

func isBuildArtifactPath(path string) bool {
	for _, marker := range []string{"/dist/", "/coverage/", "/build/", "/.next/", "/playwright-report/", "/test-results/"} {
		if strings.Contains(path, marker) {
			return true
		}
	}
	return false
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

// --- Token utilities (used by search_fusion.go tests and benchmarks) ---

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

// --- Range overlap check (used by search_fusion.go dedup) ---

func rangesOverlapOrTouch(aStart, aEnd, bStart, bEnd, window int) bool {
	if aStart > bEnd+window {
		return false
	}
	if bStart > aEnd+window {
		return false
	}
	return true
}
