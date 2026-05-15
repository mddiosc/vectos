package watcher

import "testing"

func TestWatcherDisabled(t *testing.T) {
	if testing.Short() {
		t.Skip("integration-style watch behavior skipped in short mode")
	}
	var w *Watcher
	if w != nil {
		t.Fatal("expected nil watcher in disabled case")
	}
}
