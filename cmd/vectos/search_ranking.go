package main

import (
	"path/filepath"
	"regexp"
	"strings"

	"vectos/internal/storage"
)

var tokenPattern = regexp.MustCompile(`[a-z0-9]+`)

var testSuffixes = []string{
	"_test.go", "_test.ts", "_test.tsx", "_test.js", "_test.jsx", "_test.py",
	".spec.ts", ".spec.tsx", ".spec.js", ".spec.jsx",
}

var testDirs = []string{
	"/test/", "/tests/", "/e2e/", "/__tests__/", "/cypress/",
}

// --- Structural penalty helpers (used by search_fusion.go) ---

func isTestFilePath(path string) bool {
	base := filepath.Base(path)
	norm := filepath.ToSlash(path)

	for _, s := range testSuffixes {
		if strings.HasSuffix(base, s) {
			return true
		}
	}
	if strings.Contains(base, ".test.") {
		return true
	}
	for _, d := range testDirs {
		if strings.Contains(norm, d) || strings.HasPrefix(norm, d[1:]) {
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

// isImportPrelude detects chunks that are predominantly import/package
// declarations with no substantive code. A chunk is considered a prelude when
// it has at least one import/package line AND zero function/type/class/export
// declarations.
func isImportPrelude(c storage.CodeChunk) bool {
	lines := strings.Split(c.Content, "\n")
	if len(lines) == 0 {
		return false
	}
	hasImport := false
	for _, l := range lines {
		t := strings.TrimSpace(l)
		if t == "" || strings.HasPrefix(t, "//") || strings.HasPrefix(t, "#") ||
			strings.HasPrefix(t, "/*") || strings.HasPrefix(t, "*") {
			continue
		}
		// Check for import/package lines.
		if strings.HasPrefix(t, "import ") || strings.HasPrefix(t, "import\t") ||
			strings.HasPrefix(t, "import(") ||
			strings.HasPrefix(t, "import {") ||
			strings.HasPrefix(t, "import type") ||
			strings.HasPrefix(t, "package ") ||
			strings.HasPrefix(t, "from ") ||
			strings.HasPrefix(t, "require(") {
			hasImport = true
			continue
		}
		// Rust/PHP use statements (must end with ; to avoid false positives
		// on Go variables named "use").
		if strings.HasPrefix(t, "use ") && strings.HasSuffix(strings.TrimRight(t, " "), ";") {
			hasImport = true
			continue
		}
		// If we find a code declaration, this is not a pure prelude.
		if strings.HasPrefix(t, "func ") ||
			strings.HasPrefix(t, "type ") ||
			strings.HasPrefix(t, "var ") ||
			strings.HasPrefix(t, "class ") ||
			strings.HasPrefix(t, "def ") ||
			strings.HasPrefix(t, "export ") ||
			strings.HasPrefix(t, "interface ") ||
			strings.HasPrefix(t, "enum ") ||
			strings.HasPrefix(t, "async function ") ||
			strings.HasPrefix(t, "function ") ||
			strings.HasPrefix(t, "public ") ||
			strings.HasPrefix(t, "private ") ||
			strings.HasPrefix(t, "protected ") ||
			strings.HasPrefix(t, "useEffect") ||
			strings.HasPrefix(t, "useState") ||
			strings.HasPrefix(t, "useCallback") ||
			strings.HasPrefix(t, "useMemo") ||
			strings.HasPrefix(t, "useRef") ||
			strings.HasPrefix(t, "useContext") ||
			strings.HasPrefix(t, "useReducer") ||
			isHookLike(t) ||
			strings.HasPrefix(t, "return <") ||
			strings.HasPrefix(t, "return(") {
			return false
		}
	}
	return hasImport
}

var hookPattern = regexp.MustCompile(`^use[A-Z]\w+`)

func isHookLike(line string) bool {
	return hookPattern.MatchString(line)
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
