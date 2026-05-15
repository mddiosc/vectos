package main

import (
	"fmt"
	"os"
	"strings"

	"vectos/internal/content"
	"vectos/internal/storage"
	"vectos/internal/workspace"
)

// parseChangedPaths splits a comma-separated string of changed file paths.
var parseChangedPaths = content.ParseChangedPaths

// collectIndexablePaths walks all given input paths and returns the list of
// indexable files and the list of skipped (non-indexable) files.
var collectIndexablePaths = content.CollectIndexablePaths

// filterChangedPaths filters the given paths and skippedPaths to only include
// entries matching the changed paths, relative to the workspace scope.
var filterChangedPaths = content.FilterChangedPaths

// resolveChangedPath resolves a changed file path against a workspace scope.
// Exposed as a function for test compatibility.
func resolveChangedPath(scope workspace.Scope, changed string) (string, error) {
	return content.ResolveChangedPath(scope, changed)
}

// detectLanguage determines the programming/markup language for a given file path.
var detectLanguage = content.DetectLanguage

// classifyCategory maps a language to its content category.
func classifyCategory(language string) string {
	return content.ClassifyCategory(language)
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
