package main

import (
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
