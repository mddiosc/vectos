package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"vectos/internal/content"
	"vectos/internal/storage"
	"vectos/internal/workspace"
)

func collectIndexablePaths(inputPaths []string, docsOnly bool) ([]string, []string, error) {
	acc := newPathAccumulator()
	for _, path := range inputPaths {
		absPath, err := filepath.Abs(path)
		if err != nil {
			return nil, nil, err
		}
		info, err := os.Stat(absPath)
		if err != nil {
			return nil, nil, err
		}
		if !info.IsDir() {
			if err := acc.addFile(absPath, docsOnly); err != nil {
				return nil, nil, err
			}
			continue
		}
		if err := acc.walkDir(absPath, docsOnly); err != nil {
			return nil, nil, err
		}
	}
	if len(acc.paths) == 0 {
		return nil, nil, fmt.Errorf("no supported files found in selected scope")
	}
	return acc.paths, acc.skipped, nil
}

// pathAccumulator collects indexable and skipped paths, deduplicating both.
type pathAccumulator struct {
	paths      []string
	skipped    []string
	seenPaths  map[string]struct{}
	seenSkip   map[string]struct{}
}

func newPathAccumulator() *pathAccumulator {
	return &pathAccumulator{
		seenPaths: map[string]struct{}{},
		seenSkip:  map[string]struct{}{},
	}
}

func (a *pathAccumulator) addFile(absPath string, docsOnly bool) error {
	language, err := detectLanguage(absPath)
	if err != nil {
		return fmt.Errorf("unsupported file type: %s", absPath)
	}
	if !shouldIndexLanguage(language, docsOnly) {
		a.addSkipped(absPath)
		return nil
	}
	a.addIndexable(absPath)
	return nil
}

func (a *pathAccumulator) walkDir(absPath string, docsOnly bool) error {
	return filepath.Walk(absPath, func(current string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() {
			if shouldSkipDir(info.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if language, err := detectLanguage(current); err == nil {
			if !shouldIndexLanguage(language, docsOnly) {
				a.addSkipped(current)
			} else {
				a.addIndexable(current)
			}
		}
		return nil
	})
}

func (a *pathAccumulator) addIndexable(path string) {
	if _, ok := a.seenPaths[path]; !ok {
		a.paths = append(a.paths, path)
		a.seenPaths[path] = struct{}{}
	}
}

func (a *pathAccumulator) addSkipped(path string) {
	if _, ok := a.seenSkip[path]; !ok {
		a.skipped = append(a.skipped, path)
		a.seenSkip[path] = struct{}{}
	}
}

var skippedDirs = map[string]struct{}{
	".git": {}, "node_modules": {}, ".opencode": {}, ".vectos": {},
	"coverage": {}, "playwright-report": {}, "test-results": {},
	"dist": {}, ".next": {}, "build": {},
}

func shouldSkipDir(name string) bool {
	_, skip := skippedDirs[name]
	return skip
}

func shouldIndexLanguage(language string, docsOnly bool) bool {
	category := classifyCategory(language)
	if docsOnly {
		return category == "docs"
	}
	return category != "docs" && category != "dependency_metadata"
}

func collectExcludedDirs(root string) []string {
	var excluded []string
	_ = filepath.Walk(root, func(current string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() && shouldSkipDir(info.Name()) {
			excluded = append(excluded, current)
			return filepath.SkipDir
		}
		return nil
	})
	return excluded
}

func parseChangedPaths(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	changed := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		changed = append(changed, trimmed)
	}
	return changed
}

func filterChangedPaths(scope workspace.Scope, paths, skippedPaths, changedPaths []string) ([]string, []string, error) {
	allowedRoots, err := absRoots(scope.Roots)
	if err != nil {
		return nil, nil, err
	}
	pathSet := toSet(paths)
	skippedSet := toSet(skippedPaths)

	acc := newPathAccumulator()
	for _, changed := range changedPaths {
		resolved, err := resolveChangedPath(scope, changed)
		if err != nil {
			return nil, nil, err
		}
		if !isWithinRoots(resolved, allowedRoots) {
			continue
		}
		if _, ok := pathSet[resolved]; ok {
			acc.addIndexable(resolved)
			continue
		}
		if _, ok := skippedSet[resolved]; ok || !fileExists(resolved) {
			acc.addSkipped(resolved)
		}
	}
	return acc.paths, acc.skipped, nil
}

func absRoots(roots []string) ([]string, error) {
	abs := make([]string, 0, len(roots))
	for _, root := range roots {
		absRoot, err := filepath.Abs(root)
		if err != nil {
			return nil, err
		}
		abs = append(abs, absRoot)
	}
	return abs, nil
}

func toSet(items []string) map[string]struct{} {
	s := make(map[string]struct{}, len(items))
	for _, item := range items {
		s[item] = struct{}{}
	}
	return s
}

func resolveChangedPath(scope workspace.Scope, changed string) (string, error) {
	if filepath.IsAbs(changed) {
		return filepath.Clean(changed), nil
	}

	bases := []string{scope.PrimaryRoot}
	if scope.WorkspaceRoot != "" && scope.WorkspaceRoot != scope.PrimaryRoot {
		bases = append([]string{scope.WorkspaceRoot}, bases...)
	}

	for _, base := range bases {
		resolved, err := filepath.Abs(filepath.Join(base, changed))
		if err != nil {
			return "", err
		}
		if isWithinRoots(resolved, scope.Roots) || fileExists(resolved) {
			return resolved, nil
		}
	}

	return filepath.Abs(filepath.Join(scope.PrimaryRoot, changed))
}

func isWithinRoots(path string, roots []string) bool {
	for _, root := range roots {
		if path == root || strings.HasPrefix(path, root+string(os.PathSeparator)) {
			return true
		}
	}
	return false
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func openStorageForScope(pm *storage.ProjectManager, scope *workspace.Scope, docsOnly bool) (*storage.SQLiteStorage, error) {
	if scope == nil || strings.TrimSpace(scope.Name) == "" {
		wd, err := os.Getwd()
		if err != nil {
			return nil, err
		}
		resolved, err := workspace.ResolveScope(wd, "")
		if err != nil {
			return nil, err
		}
		scope = &resolved
	}

	if docsOnly {
		return storage.NewSQLiteStorageForDocsProjectName(pm, scope.Name)
	}
	return storage.NewSQLiteStorageForProjectName(pm, scope.Name)
}

func resolveRuntimeScope(projectName string) (*workspace.Scope, error) {
	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	scope, err := workspace.ResolveScope(wd, projectName)
	if err != nil {
		return nil, err
	}

	if strings.TrimSpace(projectName) != "" && !scope.IsWorkspace() {
		return nil, fmt.Errorf("project %q requires an Nx workspace; no nx.json found from %s", projectName, wd)
	}

	return &scope, nil
}

func resolveToolScope(path string, projectName string) (*workspace.Scope, error) {
	if strings.TrimSpace(path) == "" {
		return resolveRuntimeScope(projectName)
	}

	scope, err := workspace.ResolveScope(path, projectName)
	if err != nil {
		return nil, err
	}
	return &scope, nil
}

// fileNameMatchers maps special filenames (or filename patterns) to a language.
// Each entry is tried in order; the first match wins.
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
	".go":       "go",
	".js":       "javascript",
	".mjs":      "javascript",
	".cjs":      "javascript",
	".jsx":      "jsx",
	".ts":       "typescript",
	".mts":      "typescript",
	".cts":      "typescript",
	".tsx":      "tsx",
	".py":       "python",
	".java":     "java",
	".kt":       "kotlin",
	".kts":      "kotlin",
	".json":     "json",
	".sh":       "shell",
	".md":       "markdown",
	".mdx":      "markdown",
	".toml":     "toml",
	".ini":      "ini",
	".conf":     "config",
	".xml":      "xml",
	".properties": "properties",
	".gradle":   "gradle",
	".sql":      "sql",
	".proto":    "proto",
	".graphql":  "graphql",
	".gql":      "graphql",
	".css":      "css",
	".scss":     "scss",
	".sass":     "sass",
	".less":     "less",
	".yml":      "yaml",
	".yaml":     "yaml",
	".rst":      "rst",
	".adoc":     "asciidoc",
	".asciidoc": "asciidoc",
	".tex":      "latex",
	".latex":    "latex",
	".txt":      "text",
	".bzl":      "bazel.bzl",
}

func detectLanguage(path string) (string, error) {
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

func classifyCategory(language string) string {
	return content.ClassifyCategory(language)
}
