package workspace

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sync/atomic"
	"testing"
)

func TestResolveScopeExpandsNxDependencyRoots(t *testing.T) {
	resetNxGraphCacheForTest()
	workspaceRoot := t.TempDir()
	writeTestFile(t, filepath.Join(workspaceRoot, "nx.json"), "{}")
	writeTestFile(t, filepath.Join(workspaceRoot, "apps", "dx", "agents", "project.json"), `{"name":"agents","root":"apps/dx/agents"}`)
	writeTestFile(t, filepath.Join(workspaceRoot, "libs", "dx", "agents", "project.json"), `{"name":"dx-agents","root":"libs/dx/agents"}`)
	writeTestFile(t, filepath.Join(workspaceRoot, "libs", "ui", "project.json"), `{"name":"ui","root":"libs/ui"}`)
	writeTestFile(t, filepath.Join(workspaceRoot, "libs", "auth", "project.json"), `{"name":"auth","root":"libs/auth"}`)

	originalReader := nxGraphReader
	nxGraphReader = func(root string) (nxGraphData, error) {
		if root != workspaceRoot {
			return nxGraphData{}, fmt.Errorf("unexpected workspace root %s", root)
		}
		return nxGraphData{
			Dependencies: map[string][]nxGraphDependency{
				"agents":    {{Target: "dx-agents"}, {Target: "ui"}},
				"dx-agents": {{Target: "auth"}},
				"ui":        nil,
				"auth":      nil,
			},
		}, nil
	}
	t.Cleanup(func() {
		nxGraphReader = originalReader
	})

	scope, err := ResolveScope(filepath.Join(workspaceRoot, "apps", "dx", "agents"), "agents")
	if err != nil {
		t.Fatalf("ResolveScope returned error: %v", err)
	}

	wantRoots := []string{
		filepath.Join(workspaceRoot, "apps", "dx", "agents"),
		filepath.Join(workspaceRoot, "libs", "dx", "agents"),
		filepath.Join(workspaceRoot, "libs", "ui"),
		filepath.Join(workspaceRoot, "libs", "auth"),
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
	writeTestFile(t, filepath.Join(workspaceRoot, "apps", "web", "project.json"), `{"name":"web","root":"apps/web"}`)

	originalReader := nxGraphReader
	nxGraphReader = func(root string) (nxGraphData, error) {
		return nxGraphData{}, fmt.Errorf("graph unavailable")
	}
	t.Cleanup(func() {
		nxGraphReader = originalReader
	})

	scope, err := ResolveScope(filepath.Join(workspaceRoot, "apps", "web"), "web")
	if err != nil {
		t.Fatalf("ResolveScope returned error: %v", err)
	}

	wantRoots := []string{filepath.Join(workspaceRoot, "apps", "web")}
	if !reflect.DeepEqual(scope.Roots, wantRoots) {
		t.Fatalf("unexpected fallback roots: got %v want %v", scope.Roots, wantRoots)
	}
}

func TestResolveScopeExcludesE2EAndDocsProjects(t *testing.T) {
	resetNxGraphCacheForTest()
	workspaceRoot := t.TempDir()
	writeTestFile(t, filepath.Join(workspaceRoot, "nx.json"), "{}")
	writeTestFile(t, filepath.Join(workspaceRoot, "apps", "web", "project.json"), `{"name":"web","root":"apps/web"}`)
	writeTestFile(t, filepath.Join(workspaceRoot, "libs", "ui", "project.json"), `{"name":"ui","root":"libs/ui"}`)
	writeTestFile(t, filepath.Join(workspaceRoot, "apps", "web-e2e", "project.json"), `{"name":"web-e2e","root":"apps/web-e2e"}`)
	writeTestFile(t, filepath.Join(workspaceRoot, "libs", "docs", "project.json"), `{"name":"docs-site","root":"libs/docs"}`)

	originalReader := nxGraphReader
	nxGraphReader = func(root string) (nxGraphData, error) {
		return nxGraphData{
			Dependencies: map[string][]nxGraphDependency{
				"web": {{Target: "ui"}, {Target: "web-e2e"}, {Target: "docs-site"}},
			},
			Projects: []nxGraphProject{
				{Name: "web", Type: "app"},
				{Name: "ui", Type: "lib"},
				{Name: "web-e2e", Type: "e2e"},
				{Name: "docs-site", Type: "lib", Data: struct {
					Root string `json:"root"`
				}{Root: "libs/docs"}},
			},
		}, nil
	}
	t.Cleanup(func() {
		nxGraphReader = originalReader
	})

	scope, err := ResolveScope(filepath.Join(workspaceRoot, "apps", "web"), "web")
	if err != nil {
		t.Fatalf("ResolveScope returned error: %v", err)
	}

	wantRoots := []string{
		filepath.Join(workspaceRoot, "apps", "web"),
		filepath.Join(workspaceRoot, "libs", "ui"),
	}
	if !reflect.DeepEqual(scope.Roots, wantRoots) {
		t.Fatalf("unexpected roots with exclusions: got %v want %v", scope.Roots, wantRoots)
	}
}

func TestLoadNxGraphCachesByWorkspaceRoot(t *testing.T) {
	resetNxGraphCacheForTest()
	workspaceRoot := t.TempDir()
	var calls int32

	originalReader := nxGraphReader
	nxGraphReader = func(root string) (nxGraphData, error) {
		atomic.AddInt32(&calls, 1)
		return nxGraphData{Dependencies: map[string][]nxGraphDependency{"web": nil}}, nil
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
