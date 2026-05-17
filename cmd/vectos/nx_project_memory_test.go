package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveToolScopeWithMemoryFallsBackToLastProjectAtWorkspaceRoot(t *testing.T) {
	workspaceRoot := t.TempDir()
	projectBaseDir := t.TempDir()
	writeNxWorkspace(t, workspaceRoot)

	if err := saveRememberedNxProject(projectBaseDir, workspaceRoot, "app-one"); err != nil {
		t.Fatalf("saveRememberedNxProject failed: %v", err)
	}

	originalCwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd failed: %v", err)
	}
	if err := os.Chdir(workspaceRoot); err != nil {
		t.Fatalf("Chdir failed: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(originalCwd) })

	binDir := filepath.Join(workspaceRoot, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	nxPath := filepath.Join(binDir, "nx")
	script := "#!/bin/sh\nif [ \"$1\" = graph ] && [ \"$2\" = --print ]; then\n  printf '%s' '{\"dependencies\":{\"app-one\":[{\"target\":\"lib-core\"}],\"lib-core\":[]}}'\n  exit 0\nfi\nexit 1\n"
	if err := os.WriteFile(nxPath, []byte(script), 0o755); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	originalPath := os.Getenv("PATH")
	if err := os.Setenv("PATH", binDir+string(os.PathListSeparator)+originalPath); err != nil {
		t.Fatalf("Setenv failed: %v", err)
	}
	t.Cleanup(func() { _ = os.Setenv("PATH", originalPath) })

	scope, err := resolveToolScopeWithMemory(projectBaseDir, "", "")
	if err != nil {
		t.Fatalf("resolveToolScopeWithMemory returned error: %v", err)
	}
	if scope.Name != "app-one" {
		t.Fatalf("unexpected project selected: got %s want app-one", scope.Name)
	}
	if len(scope.Warnings) == 0 || !strings.Contains(scope.Warnings[0], "reusing last project \"app-one\"") {
		t.Fatalf("expected warning about fallback reuse, got %v", scope.Warnings)
	}
	if !containsPath(scope.Roots, filepath.Join(workspaceRoot, "libs", "lib-core")) {
		t.Fatalf("unexpected roots: %#v", scope.Roots)
	}
}

func TestResolveToolScopeWithMemoryRemembersResolvedProject(t *testing.T) {
	workspaceRoot := t.TempDir()
	projectBaseDir := t.TempDir()
	writeNxWorkspace(t, workspaceRoot)

	binDir := filepath.Join(workspaceRoot, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	nxPath := filepath.Join(binDir, "nx")
	script := "#!/bin/sh\nif [ \"$1\" = graph ] && [ \"$2\" = --print ]; then\n  printf '%s' '{\"dependencies\":{\"app-one\":[{\"target\":\"lib-core\"}],\"lib-core\":[]}}'\n  exit 0\nfi\nexit 1\n"
	if err := os.WriteFile(nxPath, []byte(script), 0o755); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	originalPath := os.Getenv("PATH")
	if err := os.Setenv("PATH", binDir+string(os.PathListSeparator)+originalPath); err != nil {
		t.Fatalf("Setenv failed: %v", err)
	}
	t.Cleanup(func() { _ = os.Setenv("PATH", originalPath) })

	scope, err := resolveToolScopeWithMemory(projectBaseDir, filepath.Join(workspaceRoot, "apps", "app-one"), "")
	if err != nil {
		t.Fatalf("resolveToolScopeWithMemory returned error: %v", err)
	}
	if scope.Name != "app-one" {
		t.Fatalf("unexpected project selected: got %s want app-one", scope.Name)
	}

	remembered, err := loadRememberedNxProject(projectBaseDir, workspaceRoot)
	if err != nil {
		t.Fatalf("loadRememberedNxProject failed: %v", err)
	}
	if remembered != "app-one" {
		t.Fatalf("unexpected remembered project: got %s want app-one", remembered)
	}
}

func TestResolveToolScopeWithMemoryPreservesOriginalError(t *testing.T) {
	projectBaseDir := t.TempDir()
	missingPath := filepath.Join(t.TempDir(), "missing")

	_, err := resolveToolScopeWithMemory(projectBaseDir, missingPath, "")
	if err == nil {
		t.Fatal("expected error for missing path")
	}
	if !strings.Contains(err.Error(), missingPath) {
		t.Fatalf("expected original error to mention missing path, got: %v", err)
	}
}

func TestNxWorkspaceMemoryPathUsesCanonicalRoot(t *testing.T) {
	projectBaseDir := t.TempDir()
	parentDir := t.TempDir()
	realRoot := filepath.Join(parentDir, "real-workspace")
	if err := os.MkdirAll(realRoot, 0o755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	linkRoot := filepath.Join(parentDir, "workspace-link")
	if err := os.Symlink(realRoot, linkRoot); err != nil {
		t.Skipf("symlink not supported: %v", err)
	}

	realPath := nxWorkspaceMemoryPath(projectBaseDir, realRoot)
	linkPath := nxWorkspaceMemoryPath(projectBaseDir, linkRoot)
	if realPath != linkPath {
		t.Fatalf("expected canonical workspace paths to match:\nreal=%s\nlink=%s", realPath, linkPath)
	}
}

func TestShouldRetryScopeResolutionWithMemory(t *testing.T) {
	if !shouldRetryScopeResolutionWithMemory(fmt.Errorf("path is the Nx workspace root; please specify a project name")) {
		t.Fatal("expected Nx root ambiguity error to be retryable")
	}
	if shouldRetryScopeResolutionWithMemory(fmt.Errorf("path does not exist")) {
		t.Fatal("did not expect unrelated errors to be retryable")
	}
}
