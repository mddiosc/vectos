package content

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"vectos/internal/workspace"
)

// CollectIndexablePaths walks all given input paths and returns the list of
// indexable files and the list of skipped (non-indexable) files.
func CollectIndexablePaths(inputPaths []string, docsOnly bool) ([]string, []string, error) {
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
	paths     []string
	skipped   []string
	seenPaths map[string]struct{}
	seenSkip  map[string]struct{}
}

func newPathAccumulator() *pathAccumulator {
	return &pathAccumulator{
		seenPaths: map[string]struct{}{},
		seenSkip:  map[string]struct{}{},
	}
}

func (a *pathAccumulator) addFile(absPath string, docsOnly bool) error {
	language, err := DetectLanguage(absPath)
	if err != nil {
		return fmt.Errorf("unsupported file type: %s", absPath)
	}
	if !ShouldIndexLanguage(language, docsOnly) {
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
			if ShouldSkipDir(info.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if language, err := DetectLanguage(current); err == nil {
			if !ShouldIndexLanguage(language, docsOnly) {
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

// ParseChangedPaths splits a comma-separated string of changed file paths.
func ParseChangedPaths(raw string) []string {
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

// FilterChangedPaths filters the given paths and skippedPaths to only include
// entries matching the changed paths, relative to the workspace scope.
func FilterChangedPaths(scope workspace.Scope, paths, skippedPaths, changedPaths []string) ([]string, []string, error) {
	allowedRoots, err := absRoots(scope.Roots)
	if err != nil {
		return nil, nil, err
	}
	pathSet := toSet(paths)
	skippedSet := toSet(skippedPaths)

	acc := newPathAccumulator()
	for _, changed := range changedPaths {
		resolved, err := ResolveChangedPath(scope, changed)
		if err != nil {
			return nil, nil, err
		}
		if isWithinRoots(resolved, allowedRoots) {
			classifyChangedPath(resolved, pathSet, skippedSet, acc)
		}
	}
	return acc.paths, acc.skipped, nil
}

// classifyChangedPath adds the resolved path to the accumulator as either
// indexable or skipped, based on whether it appears in the existing path/skipped sets.
func classifyChangedPath(resolved string, pathSet, skippedSet map[string]struct{}, acc *pathAccumulator) {
	if _, ok := pathSet[resolved]; ok {
		acc.addIndexable(resolved)
		return
	}
	if _, ok := skippedSet[resolved]; ok || !fileExists(resolved) {
		acc.addSkipped(resolved)
	}
}

// CollectExcludedDirs finds all skipped directories within a root path.
func CollectExcludedDirs(root string) []string {
	var excluded []string
	_ = filepath.Walk(root, func(current string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() && ShouldSkipDir(info.Name()) {
			excluded = append(excluded, current)
			return filepath.SkipDir
		}
		return nil
	})
	return excluded
}

// absRoots converts all given paths to absolute paths.
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

// changedPathBases returns the ordered list of base directories to try when
// resolving a relative changed path. Priority: workspace root → primary root →
// remaining scope roots (dependency libs). Duplicates are skipped.
func changedPathBases(scope workspace.Scope) []string {
	seen := map[string]struct{}{}
	bases := make([]string, 0, 2+len(scope.Roots))

	add := func(b string) {
		if b == "" {
			return
		}
		if _, ok := seen[b]; ok {
			return
		}
		seen[b] = struct{}{}
		bases = append(bases, b)
	}

	add(scope.WorkspaceRoot)
	add(scope.PrimaryRoot)
	for _, root := range scope.Roots {
		add(root)
	}
	return bases
}

// ResolveChangedPath resolves a potentially relative changed file path against the
// workspace scope. It tries each scope root (workspace root, primary root, dependency
// roots) in priority order, preferring an existing file over a non-existent one.
func ResolveChangedPath(scope workspace.Scope, changed string) (string, error) {
	if filepath.IsAbs(changed) {
		return filepath.Clean(changed), nil
	}

	bases := changedPathBases(scope)

	// First pass: prefer a base where the file exists within scope.
	if resolved, ok, err := tryResolveInBases(bases, changed, scope.Roots, true); err != nil {
		return "", err
	} else if ok {
		return resolved, nil
	}

	// Second pass: accept a base where the path is within scope even if the
	// file does not yet exist (new file being indexed for the first time).
	if resolved, ok, err := tryResolveInBases(bases, changed, scope.Roots, false); err != nil {
		return "", err
	} else if ok {
		return resolved, nil
	}

	return filepath.Abs(filepath.Join(scope.PrimaryRoot, changed))
}

// tryResolveInBases attempts to resolve the changed path against each base
// directory. When requireExists is true, only bases where the file already
// exists are considered. Returns the resolved absolute path when found.
func tryResolveInBases(bases []string, changed string, roots []string, requireExists bool) (string, bool, error) {
	for _, base := range bases {
		resolved, err := filepath.Abs(filepath.Join(base, changed))
		if err != nil {
			return "", false, err
		}
		if requireExists && !fileExists(resolved) {
			continue
		}
		if isWithinRoots(resolved, roots) {
			return resolved, true, nil
		}
	}
	return "", false, nil
}

// IsWithinRoots checks if the given path is within any of the specified roots.
func IsWithinRoots(path string, roots []string) bool {
	for _, root := range roots {
		if path == root || strings.HasPrefix(path, root+string(os.PathSeparator)) {
			return true
		}
	}
	return false
}

func isWithinRoots(path string, roots []string) bool {
	return IsWithinRoots(path, roots)
}

// FileExists reports whether a file exists at the given path.
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func fileExists(path string) bool {
	return FileExists(path)
}
