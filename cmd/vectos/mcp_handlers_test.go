package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"vectos/internal/workspace"
)

func TestListProjectsHandlerInNxWorkspace(t *testing.T) {
	workspaceRoot := t.TempDir()
	writeTestFileLocal(t, filepath.Join(workspaceRoot, "nx.json"), "{}")
	writeTestFileLocal(t, filepath.Join(workspaceRoot, "apps", "app-one", "project.json"), `{"name":"app-one","root":"apps/app-one"}`)
	writeTestFileLocal(t, filepath.Join(workspaceRoot, "libs", "lib-core", "project.json"), `{"name":"lib-core","root":"libs/lib-core"}`)
	writeTestFileLocal(t, filepath.Join(workspaceRoot, "libs", "lib-shared", "project.json"), `{"name":"lib-shared","root":"libs/lib-shared"}`)

	projects, err := workspace.DiscoverNxProjectNames(workspaceRoot)
	if err != nil {
		t.Fatalf("DiscoverNxProjectNames returned error: %v", err)
	}
	want := []string{"app-one", "lib-core", "lib-shared"}
	if !reflect.DeepEqual(projects, want) {
		t.Fatalf("unexpected projects: got %v want %v", projects, want)
	}
}

func writeTestFileLocal(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("failed to create dir for %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write %s: %v", path, err)
	}
}

func TestListProjectsHandlerOutsideNxWorkspace(t *testing.T) {
	workspaceRoot := t.TempDir()
	projects, err := workspace.DiscoverNxProjectNames(workspaceRoot)
	if err != nil {
		t.Fatalf("DiscoverNxProjectNames returned error: %v", err)
	}
	if projects != nil {
		t.Fatalf("expected nil projects outside Nx workspace, got %v", projects)
	}
}
