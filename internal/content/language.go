package content

import (
	"fmt"
	"path/filepath"
	"strings"
)

// skippedDirs lists directories that should be excluded from indexing.
var skippedDirs = map[string]struct{}{
	".git": {}, "node_modules": {}, ".opencode": {}, ".vectos": {},
	"coverage": {}, "playwright-report": {}, "test-results": {},
	"dist": {}, ".next": {}, "build": {},
}

// ShouldSkipDir reports whether a directory should be skipped during file walks.
func ShouldSkipDir(name string) bool {
	_, skip := skippedDirs[name]
	return skip
}

// fileNameMatchers maps special filenames (or filename patterns) to a language.
type fileNameMatcher struct {
	match func(baseName, lowerBase string) bool
	lang  string
}

var fileNameMatchers = []fileNameMatcher{
	{func(b, _ string) bool { return b == "Dockerfile" || strings.HasPrefix(b, "Dockerfile.") }, "dockerfile"},
	{func(b, _ string) bool { return b == "Makefile" }, "makefile"},
	{func(b, _ string) bool { return b == ".editorconfig" }, "ini"},
	{func(b, _ string) bool { return b == ".gitignore" || b == ".prettierignore" || b == ".eslintignore" }, "gitignore"},
	{func(b, _ string) bool {
		return b == ".npmrc" || b == ".yarnrc" || b == ".nvmrc" || b == ".prettierrc" || b == ".tool-versions"
	}, "config"},
	{func(b, _ string) bool { return b == "gradlew" || b == "mvnw" }, "shell"},
	{func(b, _ string) bool { return strings.HasSuffix(b, ".gradle.kts") }, "gradle"},
	{func(b, _ string) bool { return strings.HasSuffix(b, ".lock") || b == "bun.lockb" }, "lockfile"},
	{func(_, lb string) bool {
		return strings.HasPrefix(lb, "docker-compose") && (strings.HasSuffix(lb, ".yml") || strings.HasSuffix(lb, ".yaml"))
	}, "yaml.compose"},
	{func(b, _ string) bool { return b == "BUILD" || b == "BUILD.bazel" }, "bazel.build"},
	{func(b, _ string) bool { return b == "WORKSPACE" }, "bazel.workspace"},
	{func(b, _ string) bool { return b == "MODULE.bazel" }, "bazel.module"},
}

// extLanguages maps lowercase file extensions to a language name.
var extLanguages = map[string]string{
	".go":           "go",
	".js":           "javascript",
	".mjs":          "javascript",
	".cjs":          "javascript",
	".jsx":          "jsx",
	".ts":           "typescript",
	".mts":          "typescript",
	".cts":          "typescript",
	".tsx":          "tsx",
	".py":           "python",
	".java":         "java",
	".kt":           "kotlin",
	".kts":          "kotlin",
	".json":         "json",
	".sh":           "shell",
	".md":           "markdown",
	".mdx":          "markdown",
	".toml":         "toml",
	".ini":          "ini",
	".conf":         "config",
	".xml":          "xml",
	".properties":   "properties",
	".gradle":       "gradle",
	".sql":          "sql",
	".proto":        "proto",
	".graphql":      "graphql",
	".gql":          "graphql",
	".css":          "css",
	".scss":         "scss",
	".sass":         "sass",
	".less":         "less",
	".yml":          "yaml",
	".yaml":         "yaml",
	".rst":          "rst",
	".adoc":         "asciidoc",
	".asciidoc":     "asciidoc",
	".tex":          "latex",
	".latex":        "latex",
	".txt":          "text",
	".bzl":          "bazel.bzl",
}

// DetectLanguage determines the programming/markup language for a given file path.
func DetectLanguage(path string) (string, error) {
	baseName := filepath.Base(path)
	lowerBase := strings.ToLower(baseName)

	for _, m := range fileNameMatchers {
		if m.match(baseName, lowerBase) {
			return m.lang, nil
		}
	}

	if lang, ok := extLanguages[strings.ToLower(filepath.Ext(path))]; ok {
		return lang, nil
	}
	return "", fmt.Errorf("unsupported file type: %s", path)
}

// ShouldIndexLanguage reports whether a file with the given language should be
// indexed. When docsOnly is true, only documentation files are accepted.
func ShouldIndexLanguage(language string, docsOnly bool) bool {
	category := ClassifyCategory(language)
	if docsOnly {
		return category == "docs"
	}
	return category != "docs" && category != "dependency_metadata"
}
