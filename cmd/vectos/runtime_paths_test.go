package main

import (
	"path/filepath"
	"testing"

	"vectos/internal/workspace"
)

func TestResolveChangedPathUsesWorkspaceRootForNxRelativePaths(t *testing.T) {
	workspaceRoot := t.TempDir()
	scope := workspace.Scope{
		Name:          "agents",
		WorkspaceRoot: workspaceRoot,
		PrimaryRoot:   filepath.Join(workspaceRoot, "apps", "dx", "agents"),
		Roots: []string{
			filepath.Join(workspaceRoot, "apps", "dx", "agents"),
			filepath.Join(workspaceRoot, "libs", "ui"),
		},
		WorkspaceType: "nx",
	}

	got, err := resolveChangedPath(scope, filepath.Join("libs", "ui", "src", "button.tsx"))
	if err != nil {
		t.Fatalf("resolveChangedPath returned error: %v", err)
	}

	want := filepath.Join(workspaceRoot, "libs", "ui", "src", "button.tsx")
	if got != want {
		t.Fatalf("unexpected path: got %s want %s", got, want)
	}
}

func TestResolveChangedPathKeepsPrimaryRootRelativePaths(t *testing.T) {
	workspaceRoot := t.TempDir()
	scope := workspace.Scope{
		Name:          "agents",
		WorkspaceRoot: workspaceRoot,
		PrimaryRoot:   filepath.Join(workspaceRoot, "apps", "dx", "agents"),
		Roots:         []string{filepath.Join(workspaceRoot, "apps", "dx", "agents")},
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
