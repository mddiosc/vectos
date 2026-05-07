package main

import (
	"os"
	"path/filepath"
	"testing"

	"vectos/internal/workspace"
)

func TestResolveChangedPathUsesWorkspaceRootForNxRelativePaths(t *testing.T) {
	workspaceRoot := t.TempDir()
	scope := workspace.Scope{
		Name:          "app-one",
		WorkspaceRoot: workspaceRoot,
		PrimaryRoot:   filepath.Join(workspaceRoot, "apps", "app-one"),
		Roots: []string{
			filepath.Join(workspaceRoot, "apps", "app-one"),
			filepath.Join(workspaceRoot, "libs", "lib-ui"),
		},
		WorkspaceType: "nx",
	}

	got, err := resolveChangedPath(scope, filepath.Join("libs", "lib-ui", "src", "button.tsx"))
	if err != nil {
		t.Fatalf("resolveChangedPath returned error: %v", err)
	}

	want := filepath.Join(workspaceRoot, "libs", "lib-ui", "src", "button.tsx")
	if got != want {
		t.Fatalf("unexpected path: got %s want %s", got, want)
	}
}

func TestResolveChangedPathKeepsPrimaryRootRelativePaths(t *testing.T) {
	workspaceRoot := t.TempDir()
	scope := workspace.Scope{
		Name:          "app-one",
		WorkspaceRoot: workspaceRoot,
		PrimaryRoot:   filepath.Join(workspaceRoot, "apps", "app-one"),
		Roots:         []string{filepath.Join(workspaceRoot, "apps", "app-one")},
		WorkspaceType: "nx",
	}

	got, err := resolveChangedPath(scope, filepath.Join("src", "main.ts"))
	if err != nil {
		t.Fatalf("resolveChangedPath returned error: %v", err)
	}

	want := filepath.Join(scope.PrimaryRoot, "src", "main.ts")
	if got != want {
		t.Fatalf("unexpected path: got %s want %s", got, want)
	}
}

func TestResolveToolScopeWithProjectOnlyResolvesFullScope(t *testing.T) {
	workspaceRoot := t.TempDir()
	writeNxWorkspace(t, workspaceRoot)

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

	scope, err := resolveToolScope("", "app-one")
	if err != nil {
		t.Fatalf("resolveToolScope returned error: %v", err)
	}
	if scope.PrimaryRoot == "" {
		t.Fatalf("expected primary root to be resolved")
	}
	if !containsPath(scope.Roots, filepath.Join(workspaceRoot, "libs", "lib-core")) {
		t.Fatalf("unexpected roots: %#v", scope.Roots)
	}
}

func TestResolveRuntimeScopePropagatesAmbiguousWorkspaceError(t *testing.T) {
	workspaceRoot := t.TempDir()
	writeNxWorkspace(t, workspaceRoot)

	originalCwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd failed: %v", err)
	}
	if err := os.Chdir(workspaceRoot); err != nil {
		t.Fatalf("Chdir failed: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(originalCwd) })

	_, err = resolveRuntimeScope("")
	if err == nil {
		t.Fatalf("expected ambiguous workspace error")
	}
}

func writeNxWorkspace(t *testing.T, root string) {
	t.Helper()
	mustWrite := func(path, content string) {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("MkdirAll failed: %v", err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("WriteFile failed: %v", err)
		}
	}
	mustWrite(filepath.Join(root, "nx.json"), "{}")
	mustWrite(filepath.Join(root, "apps", "app-one", "project.json"), `{"name":"app-one","root":"apps/app-one"}`)
	mustWrite(filepath.Join(root, "libs", "lib-core", "project.json"), `{"name":"lib-core","root":"libs/lib-core"}`)
}

func containsPath(paths []string, want string) bool {
	// Resolve symlinks on both sides to handle macOS /var → /private/var aliasing.
	wantResolved, err := filepath.EvalSymlinks(want)
	if err != nil {
		wantResolved = want
	}
	for _, path := range paths {
		resolved, err := filepath.EvalSymlinks(path)
		if err != nil {
			resolved = path
		}
		if resolved == wantResolved {
			return true
		}
	}
	return false
}
