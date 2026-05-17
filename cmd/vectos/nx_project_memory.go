package main

import (
	"crypto/sha256"
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

	if strings.TrimSpace(projectName) != "" || !shouldRetryScopeResolutionWithMemory(err) {
		return nil, err
	}

	startPath := strings.TrimSpace(path)
	if startPath == "" {
		wd, wdErr := os.Getwd()
		if wdErr != nil {
			return nil, err
		}
		startPath = wd
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
		return nil, fmt.Errorf("%w (fallback to remembered Nx project %q also failed: %v)", err, lastProject, fallbackErr)
	}
	resolved.Warnings = append(resolved.Warnings, fmt.Sprintf("No Nx project specified from workspace root; reusing last project %q", lastProject))
	rememberResolvedNxProject(projectBaseDir, resolved)
	return &resolved, nil
}

func shouldRetryScopeResolutionWithMemory(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "Nx workspace root") && strings.Contains(err.Error(), "please specify a project name")
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
	normalizedRoot := canonicalWorkspaceRoot(workspaceRoot)
	sum := sha256.Sum256([]byte(normalizedRoot))
	return filepath.Join(projectBaseDir, ".nx-workspaces", fmt.Sprintf("%x.json", sum))
}

func canonicalWorkspaceRoot(workspaceRoot string) string {
	normalizedRoot := filepath.Clean(strings.TrimSpace(workspaceRoot))
	if absRoot, err := filepath.Abs(normalizedRoot); err == nil && strings.TrimSpace(absRoot) != "" {
		normalizedRoot = absRoot
	}
	if resolved, err := filepath.EvalSymlinks(normalizedRoot); err == nil && strings.TrimSpace(resolved) != "" {
		normalizedRoot = resolved
	}
	return filepath.Clean(normalizedRoot)
}
