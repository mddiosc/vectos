package main

import (
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"vectos/internal/workspace"
)

type nxWorkspaceMemory struct {
	LastProject string `json:"last_project"`
}

func resolveToolScopeWithMemory(projectBaseDir string, path string, projectName string) (*workspace.Scope, error) {
	scope, err := resolveToolScope(path, projectName)
	if err == nil {
		rememberResolvedNxProject(projectBaseDir, *scope)
		return scope, nil
	}

	if strings.TrimSpace(projectName) != "" {
		return nil, err
	}

	startPath, startErr := scopeStartPath(path)
	if startErr != nil {
		return nil, err
	}

	workspaceRoot, rootErr := workspace.DetectNxWorkspaceRoot(startPath)
	if rootErr != nil || strings.TrimSpace(workspaceRoot) == "" {
		return nil, err
	}

	lastProject, loadErr := loadRememberedNxProject(projectBaseDir, workspaceRoot)
	if loadErr != nil || strings.TrimSpace(lastProject) == "" {
		return nil, err
	}

	resolved, fallbackErr := workspace.ResolveScope(startPath, lastProject)
	if fallbackErr != nil {
		return nil, err
	}
	resolved.Warnings = append(resolved.Warnings, fmt.Sprintf("No Nx project specified from workspace root; reusing last project %q", lastProject))
	rememberResolvedNxProject(projectBaseDir, resolved)
	return &resolved, nil
}

func scopeStartPath(path string) (string, error) {
	if strings.TrimSpace(path) != "" {
		return path, nil
	}
	return os.Getwd()
}

func rememberResolvedNxProject(projectBaseDir string, scope workspace.Scope) {
	if !scope.IsWorkspace() || strings.TrimSpace(scope.WorkspaceRoot) == "" || strings.TrimSpace(scope.Name) == "" {
		return
	}
	_ = saveRememberedNxProject(projectBaseDir, scope.WorkspaceRoot, scope.Name)
}

func saveRememberedNxProject(projectBaseDir string, workspaceRoot string, projectName string) error {
	if strings.TrimSpace(projectBaseDir) == "" || strings.TrimSpace(workspaceRoot) == "" || strings.TrimSpace(projectName) == "" {
		return nil
	}

	path := nxWorkspaceMemoryPath(projectBaseDir, workspaceRoot)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	payload, err := json.MarshalIndent(nxWorkspaceMemory{LastProject: projectName}, "", "  ")
	if err != nil {
		return err
	}
	payload = append(payload, '\n')
	return os.WriteFile(path, payload, 0o644)
}

func loadRememberedNxProject(projectBaseDir string, workspaceRoot string) (string, error) {
	content, err := os.ReadFile(nxWorkspaceMemoryPath(projectBaseDir, workspaceRoot))
	if err != nil {
		return "", err
	}

	var memory nxWorkspaceMemory
	if err := json.Unmarshal(content, &memory); err != nil {
		return "", err
	}
	return strings.TrimSpace(memory.LastProject), nil
}

func nxWorkspaceMemoryPath(projectBaseDir string, workspaceRoot string) string {
	normalizedRoot := workspaceRoot
	if resolved, err := filepath.EvalSymlinks(workspaceRoot); err == nil && strings.TrimSpace(resolved) != "" {
		normalizedRoot = resolved
	}
	sum := sha1.Sum([]byte(normalizedRoot))
	return filepath.Join(projectBaseDir, ".nx-workspaces", fmt.Sprintf("%x.json", sum))
}
