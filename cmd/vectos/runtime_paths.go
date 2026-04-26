package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"vectos/internal/storage"
	"vectos/internal/workspace"
)

func collectIndexablePaths(inputPaths []string) ([]string, []string, error) {
	var paths []string
	var skippedPaths []string
	seen := map[string]struct{}{}
	skippedSeen := map[string]struct{}{}
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
			if language, err := detectLanguage(absPath); err == nil {
				if !shouldIndexLanguage(language) {
					if _, ok := skippedSeen[absPath]; !ok {
						skippedPaths = append(skippedPaths, absPath)
						skippedSeen[absPath] = struct{}{}
					}
					continue
				}
				if _, ok := seen[absPath]; !ok {
					paths = append(paths, absPath)
					seen[absPath] = struct{}{}
				}
				continue
			}
			return nil, nil, fmt.Errorf("unsupported file type: %s", absPath)
		}

		err = filepath.Walk(absPath, func(current string, info os.FileInfo, walkErr error) error {
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
				if !shouldIndexLanguage(language) {
					if _, ok := skippedSeen[current]; !ok {
						skippedPaths = append(skippedPaths, current)
						skippedSeen[current] = struct{}{}
					}
					return nil
				}
				if _, ok := seen[current]; !ok {
					paths = append(paths, current)
					seen[current] = struct{}{}
				}
			}
			return nil
		})
		if err != nil {
			return nil, nil, err
		}
	}

	if len(paths) == 0 {
		return nil, nil, fmt.Errorf("no supported files found in selected scope")
	}

	return paths, skippedPaths, nil
}

func shouldSkipDir(name string) bool {
	switch name {
	case ".git", "node_modules", ".opencode", ".vectos", "coverage", "playwright-report", "test-results", "dist", ".next", "build":
		return true
	default:
		return false
	}
}

func shouldIndexLanguage(language string) bool {
	category := classifyCategory(language)
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
	allowedRoots := make([]string, 0, len(scope.Roots))
	for _, root := range scope.Roots {
		absRoot, err := filepath.Abs(root)
		if err != nil {
			return nil, nil, err
		}
		allowedRoots = append(allowedRoots, absRoot)
	}

	pathSet := make(map[string]struct{}, len(paths))
	for _, path := range paths {
		pathSet[path] = struct{}{}
	}

	skippedSet := make(map[string]struct{}, len(skippedPaths))
	for _, path := range skippedPaths {
		skippedSet[path] = struct{}{}
	}

	var filteredPaths []string
	var filteredSkipped []string
	seenPaths := map[string]struct{}{}
	seenSkipped := map[string]struct{}{}

	for _, changed := range changedPaths {
		resolved, err := resolveChangedPath(scope.PrimaryRoot, changed)
		if err != nil {
			return nil, nil, err
		}
		if !isWithinRoots(resolved, allowedRoots) {
			continue
		}
		if _, ok := pathSet[resolved]; ok {
			if _, seen := seenPaths[resolved]; !seen {
				filteredPaths = append(filteredPaths, resolved)
				seenPaths[resolved] = struct{}{}
			}
			continue
		}
		if _, ok := skippedSet[resolved]; ok || !fileExists(resolved) {
			if _, seen := seenSkipped[resolved]; !seen {
				filteredSkipped = append(filteredSkipped, resolved)
				seenSkipped[resolved] = struct{}{}
			}
		}
	}

	return filteredPaths, filteredSkipped, nil
}

func resolveChangedPath(baseRoot, changed string) (string, error) {
	if filepath.IsAbs(changed) {
		return filepath.Clean(changed), nil
	}
	return filepath.Abs(filepath.Join(baseRoot, changed))
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

func openStorageForScope(pm *storage.ProjectManager, scope *workspace.Scope) (*storage.SQLiteStorage, error) {
	if scope == nil || strings.TrimSpace(scope.Name) == "" {
		return storage.NewSQLiteStorage(pm)
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
		if strings.TrimSpace(projectName) == "" {
			return nil, nil
		}
		return nil, err
	}

	return &scope, nil
}

func resolveToolScope(path string, projectName string) (*workspace.Scope, error) {
	if strings.TrimSpace(path) == "" {
		if strings.TrimSpace(projectName) != "" {
			return &workspace.Scope{Name: projectName}, nil
		}
		return resolveRuntimeScope(projectName)
	}

	scope, err := workspace.ResolveScope(path, projectName)
	if err != nil {
		return nil, err
	}
	return &scope, nil
}

func detectLanguage(path string) (string, error) {
	baseName := filepath.Base(path)
	lowerBase := strings.ToLower(baseName)
	switch {
	case baseName == "Dockerfile" || strings.HasPrefix(baseName, "Dockerfile."):
		return "dockerfile", nil
	case baseName == "Makefile":
		return "makefile", nil
	case baseName == ".editorconfig":
		return "ini", nil
	case baseName == ".gitignore":
		return "gitignore", nil
	case baseName == ".prettierignore" || baseName == ".eslintignore":
		return "gitignore", nil
	case baseName == ".npmrc" || baseName == ".yarnrc" || baseName == ".nvmrc" || baseName == ".prettierrc" || baseName == ".tool-versions":
		return "config", nil
	case baseName == "gradlew" || baseName == "mvnw":
		return "shell", nil
	case strings.HasSuffix(baseName, ".gradle.kts"):
		return "gradle", nil
	case strings.HasSuffix(baseName, ".lock") || baseName == "bun.lockb":
		return "lockfile", nil
	case strings.HasPrefix(lowerBase, "docker-compose") && (strings.HasSuffix(lowerBase, ".yml") || strings.HasSuffix(lowerBase, ".yaml")):
		return "yaml.compose", nil
	case baseName == "BUILD":
		return "bazel.build", nil
	case baseName == "BUILD.bazel":
		return "bazel.build", nil
	case baseName == "WORKSPACE":
		return "bazel.workspace", nil
	case baseName == "MODULE.bazel":
		return "bazel.module", nil
	}

	switch strings.ToLower(filepath.Ext(path)) {
	case ".go":
		return "go", nil
	case ".js":
		return "javascript", nil
	case ".mjs", ".cjs":
		return "javascript", nil
	case ".jsx":
		return "jsx", nil
	case ".ts":
		return "typescript", nil
	case ".mts", ".cts":
		return "typescript", nil
	case ".tsx":
		return "tsx", nil
	case ".py":
		return "python", nil
	case ".java":
		return "java", nil
	case ".kt":
		return "kotlin", nil
	case ".kts":
		return "kotlin", nil
	case ".json":
		return "json", nil
	case ".sh":
		return "shell", nil
	case ".md":
		return "markdown", nil
	case ".mdx":
		return "markdown", nil
	case ".toml":
		return "toml", nil
	case ".ini":
		return "ini", nil
	case ".conf":
		return "config", nil
	case ".xml":
		return "xml", nil
	case ".properties":
		return "properties", nil
	case ".gradle":
		return "gradle", nil
	case ".sql":
		return "sql", nil
	case ".proto":
		return "proto", nil
	case ".graphql", ".gql":
		return "graphql", nil
	case ".css":
		return "css", nil
	case ".scss":
		return "scss", nil
	case ".sass":
		return "sass", nil
	case ".less":
		return "less", nil
	case ".yml":
		return "yaml", nil
	case ".yaml":
		return "yaml", nil
	case ".bzl":
		return "bazel.bzl", nil
	default:
		return "", fmt.Errorf("unsupported file type: %s", path)
	}
}

func classifyCategory(language string) string {
	switch {
	case language == "shell":
		return "scripts"
	case language == "markdown", language == "gitignore":
		return "docs"
	case language == "json", language == "toml", language == "ini", language == "xml", language == "properties", language == "makefile", language == "gradle", language == "lockfile", language == "config":
		return classifyMetadataCategory(language)
	case language == "dockerfile", strings.HasPrefix(language, "yaml"), strings.HasPrefix(language, "bazel"):
		return "infra_config"
	default:
		return "source"
	}
}

func classifyMetadataCategory(language string) string {
	switch language {
	case "json", "toml", "properties", "xml", "makefile", "gradle", "lockfile":
		return "dependency_metadata"
	default:
		return "infra_config"
	}
}
