package watcher

import (
	"reflect"
	"sync"
	"testing"
	"time"
)

func TestWatcherShouldIgnore(t *testing.T) {
	w := &Watcher{rootPath: "/tmp/test", ignores: []string{".git", "node_modules", "*.lock", "*.test"}}
	tests := []struct {
		path string
		want bool
	}{
		{"/tmp/test/.git", true},
		{"/tmp/test/node_modules", true},
		{"/tmp/test/src/main.go", false},
		{"/tmp/test/package-lock.json", false},
		{"/tmp/test/yarn.lock", true},
		{"/tmp/test/foo.test", true},
		{"/tmp/test/.gitignore", false},
	}
	for _, tt := range tests {
		if got := w.shouldIgnore(tt.path); got != tt.want {
			t.Errorf("shouldIgnore(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}

func TestWatcherDebounce(t *testing.T) {
	var mu sync.Mutex
	var receivedPaths []string
	onChange := func(paths []string) {
		mu.Lock()
		receivedPaths = append(receivedPaths, paths...)
		mu.Unlock()
	}
	w := &Watcher{debounce: 100 * time.Millisecond, onChange: onChange, pending: make(map[string]struct{})}
	w.debounceEvent("a.go")
	w.debounceEvent("b.go")
	w.debounceEvent("c.go")
	time.Sleep(250 * time.Millisecond)
	mu.Lock()
	defer mu.Unlock()
	if len(receivedPaths) != 3 {
		t.Fatalf("received %d paths, want 3", len(receivedPaths))
	}
	if !reflect.DeepEqual(map[string]struct{}{"a.go": {}, "b.go": {}, "c.go": {}}, toSet(receivedPaths)) {
		t.Fatalf("received paths = %v, want all three paths", receivedPaths)
	}
}

func toSet(paths []string) map[string]struct{} {
	out := make(map[string]struct{}, len(paths))
	for _, p := range paths {
		out[p] = struct{}{}
	}
	return out
}
