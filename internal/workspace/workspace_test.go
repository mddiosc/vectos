package workspace

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
)

func TestResolveScopeExpandsNxDependencyRoots(t *testing.T) {
	resetNxGraphCacheForTest()
	workspaceRoot := t.TempDir()
	writeTestFile(t, filepath.Join(workspaceRoot, "nx.json"), "{}")
	writeTestFile(t, filepath.Join(workspaceRoot, "apps", "app-one", "project.json"), `{"name":"app-one","root":"apps/app-one"}`)
	writeTestFile(t, filepath.Join(workspaceRoot, "libs", "lib-core", "project.json"), `{"name":"lib-core","root":"libs/lib-core"}`)
	writeTestFile(t, filepath.Join(workspaceRoot, "libs", "lib-ui", "project.json"), `{"name":"lib-ui","root":"libs/lib-ui"}`)
	writeTestFile(t, filepath.Join(workspaceRoot, "libs", "lib-auth", "project.json"), `{"name":"lib-auth","root":"libs/lib-auth"}`)

	originalReader := nxGraphReader
	nxGraphReader = func(root string) (nxGraphData, error) {
		if root != workspaceRoot {
			return nxGraphData{}, fmt.Errorf("unexpected workspace root %s", root)
		}
		return nxGraphData{
			Dependencies: map[string][]nxGraphDependency{
				"app-one":  {{Target: "lib-core"}, {Target: "lib-ui"}},
				"lib-core": {{Target: "lib-auth"}},
				"lib-ui":   nil,
				"lib-auth": nil,
			},
		}, nil
	}
	t.Cleanup(func() {
		nxGraphReader = originalReader
	})

	scope, err := ResolveScope(filepath.Join(workspaceRoot, "apps", "app-one"), "app-one")
	if err != nil {
		t.Fatalf("ResolveScope returned error: %v", err)
	}

	wantRoots := []string{
		filepath.Join(workspaceRoot, "apps", "app-one"),
		filepath.Join(workspaceRoot, "libs", "lib-core"),
		filepath.Join(workspaceRoot, "libs", "lib-ui"),
		filepath.Join(workspaceRoot, "libs", "lib-auth"),
	}
	if !reflect.DeepEqual(scope.Roots, wantRoots) {
		t.Fatalf("unexpected roots: got %v want %v", scope.Roots, wantRoots)
	}
	if scope.PrimaryRoot != wantRoots[0] {
		t.Fatalf("unexpected primary root: got %s want %s", scope.PrimaryRoot, wantRoots[0])
	}
}

func TestResolveScopeFallsBackToPrimaryRootWhenNxGraphUnavailable(t *testing.T) {
	resetNxGraphCacheForTest()
	workspaceRoot := t.TempDir()
	writeTestFile(t, filepath.Join(workspaceRoot, "nx.json"), "{}")
	writeTestFile(t, filepath.Join(workspaceRoot, "apps", "app-two", "project.json"), `{"name":"app-two","root":"apps/app-two"}`)

	originalReader := nxGraphReader
	nxGraphReader = func(root string) (nxGraphData, error) {
		return nxGraphData{}, fmt.Errorf("graph unavailable")
	}
	t.Cleanup(func() {
		nxGraphReader = originalReader
	})

	scope, err := ResolveScope(filepath.Join(workspaceRoot, "apps", "app-two"), "app-two")
	if err != nil {
		t.Fatalf("ResolveScope returned error: %v", err)
	}

	wantRoots := []string{filepath.Join(workspaceRoot, "apps", "app-two")}
	if !reflect.DeepEqual(scope.Roots, wantRoots) {
		t.Fatalf("unexpected fallback roots: got %v want %v", scope.Roots, wantRoots)
	}
}

func TestResolveScopeExcludesE2EAndDocsProjects(t *testing.T) {
	resetNxGraphCacheForTest()
	workspaceRoot := t.TempDir()
	writeTestFile(t, filepath.Join(workspaceRoot, "nx.json"), "{}")
	writeTestFile(t, filepath.Join(workspaceRoot, "apps", "app-two", "project.json"), `{"name":"app-two","root":"apps/app-two"}`)
	writeTestFile(t, filepath.Join(workspaceRoot, "libs", "lib-ui", "project.json"), `{"name":"lib-ui","root":"libs/lib-ui"}`)
	writeTestFile(t, filepath.Join(workspaceRoot, "apps", "app-two-e2e", "project.json"), `{"name":"app-two-e2e","root":"apps/app-two-e2e"}`)
	writeTestFile(t, filepath.Join(workspaceRoot, "libs", "docs", "project.json"), `{"name":"docs-site","root":"libs/docs"}`)

	originalReader := nxGraphReader
	nxGraphReader = func(root string) (nxGraphData, error) {
		return nxGraphData{
			Dependencies: map[string][]nxGraphDependency{
				"app-two": {{Target: "lib-ui"}, {Target: "app-two-e2e"}, {Target: "docs-site"}},
			},
			Projects: []nxGraphProject{
				{Name: "app-two", Type: "app"},
				{Name: "lib-ui", Type: "lib"},
				{Name: "app-two-e2e", Type: "e2e"},
				{Name: "docs-site", Type: "lib", Data: struct {
					Root string `json:"root"`
				}{Root: "libs/docs"}},
			},
		}, nil
	}
	t.Cleanup(func() {
		nxGraphReader = originalReader
	})

	scope, err := ResolveScope(filepath.Join(workspaceRoot, "apps", "app-two"), "app-two")
	if err != nil {
		t.Fatalf("ResolveScope returned error: %v", err)
	}

	wantRoots := []string{
		filepath.Join(workspaceRoot, "apps", "app-two"),
		filepath.Join(workspaceRoot, "libs", "lib-ui"),
	}
	if !reflect.DeepEqual(scope.Roots, wantRoots) {
		t.Fatalf("unexpected roots with exclusions: got %v want %v", scope.Roots, wantRoots)
	}
}

func TestResolveScopeWorkspaceRootListsProjects(t *testing.T) {
	t.Run("multiple projects", func(t *testing.T) {
		resetNxGraphCacheForTest()
		workspaceRoot := t.TempDir()
		writeTestFile(t, filepath.Join(workspaceRoot, "nx.json"), "{}")
		writeTestFile(t, filepath.Join(workspaceRoot, "apps", "app-one", "project.json"), `{"name":"app-one","root":"apps/app-one"}`)
		writeTestFile(t, filepath.Join(workspaceRoot, "libs", "lib-core", "project.json"), `{"name":"lib-core","root":"libs/lib-core"}`)

		_, err := ResolveScope(workspaceRoot, "")
		if err == nil {
			t.Fatal("ResolveScope returned nil error")
		}
		msg := err.Error()
		if !strings.Contains(msg, "app-one") || !strings.Contains(msg, "lib-core") {
			t.Fatalf("error does not list projects: %v", err)
		}
		if !strings.Contains(msg, "workspace root") {
			t.Fatalf("error does not indicate workspace root: %v", err)
		}
	})

	t.Run("single project", func(t *testing.T) {
		resetNxGraphCacheForTest()
		workspaceRoot := t.TempDir()
		writeTestFile(t, filepath.Join(workspaceRoot, "nx.json"), "{}")
		writeTestFile(t, filepath.Join(workspaceRoot, "apps", "app-one", "project.json"), `{"name":"app-one","root":"apps/app-one"}`)

		originalReader := nxGraphReader
		nxGraphReader = func(root string) (nxGraphData, error) {
			return nxGraphData{}, fmt.Errorf("graph unavailable")
		}
		t.Cleanup(func() {
			nxGraphReader = originalReader
		})

		scope, err := ResolveScope(workspaceRoot, "")
		if err != nil {
			t.Fatalf("ResolveScope returned error: %v", err)
		}
		if scope.Name != "app-one" {
			t.Fatalf("unexpected project selected: got %s want app-one", scope.Name)
		}
	})
}

func TestLoadNxGraphCachesByWorkspaceRoot(t *testing.T) {
	resetNxGraphCacheForTest()
	workspaceRoot := t.TempDir()
	var calls int32

	originalReader := nxGraphReader
	nxGraphReader = func(root string) (nxGraphData, error) {
		atomic.AddInt32(&calls, 1)
		return nxGraphData{Dependencies: map[string][]nxGraphDependency{"app-one": nil}}, nil
	}
	t.Cleanup(func() {
		nxGraphReader = originalReader
	})

	first, err := loadNxGraph(workspaceRoot)
	if err != nil {
		t.Fatalf("first loadNxGraph returned error: %v", err)
	}
	second, err := loadNxGraph(workspaceRoot)
	if err != nil {
		t.Fatalf("second loadNxGraph returned error: %v", err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("cached graph mismatch: %v vs %v", first, second)
	}
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("expected one reader call, got %d", got)
	}
}

func resetNxGraphCacheForTest() {
	nxGraphCache.Lock()
	defer nxGraphCache.Unlock()
	nxGraphCache.entries = map[string]nxGraphData{}
}

func writeTestFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("failed to create dir for %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write %s: %v", path, err)
	}
}
