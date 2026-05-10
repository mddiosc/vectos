package main

import (
	"testing"
	"vectos/internal/workspace"
)

func TestResolveAndPrintScopeFormatsNxLibs(t *testing.T) {
	scope := workspace.Scope{
		Name:          "test-app",
		WorkspaceRoot: "/workspace",
		PrimaryRoot:   "/workspace/apps/test-app",
		Roots: []string{
			"/workspace/apps/test-app",
			"/workspace/libs/lib-core",
			"/workspace/libs/lib-ui",
		},
		WorkspaceType: "nx",
	}
	if !scope.IsWorkspace() {
		t.Fatal("expected workspace scope")
	}
	if len(scope.Roots) != 3 {
		t.Fatalf("expected 3 roots, got %d", len(scope.Roots))
	}
	if scope.Roots[0] != scope.PrimaryRoot {
		t.Fatal("expected first root to be primary root")
	}
}
