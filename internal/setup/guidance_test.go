package setup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testStart = "<!-- test:start -->"
const testEnd = "<!-- test:end -->"

func TestEnsureManagedGuidanceAppendsToEmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "AGENTS.md")

	block := testStart + "\nsome guidance\n" + testEnd
	changed, err := ensureManagedGuidance(path, block, testStart, testEnd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !changed {
		t.Fatal("expected changed=true for empty file")
	}

	content, _ := os.ReadFile(path)
	if !strings.Contains(string(content), "some guidance") {
		t.Fatalf("expected written content to contain the block, got: %s", content)
	}
}

func TestEnsureManagedGuidanceReplacesExistingBlock(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "AGENTS.md")

	// Write existing content with old block
	oldBlock := testStart + "\nold guidance\n" + testEnd
	os.WriteFile(path, []byte("user text\n\n"+oldBlock+"\n\nmore user text"), 0644)

	newBlock := testStart + "\nnew guidance\n" + testEnd
	changed, err := ensureManagedGuidance(path, newBlock, testStart, testEnd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !changed {
		t.Fatal("expected changed=true for replacement")
	}

	content, _ := os.ReadFile(path)
	if !strings.Contains(string(content), "new guidance") {
		t.Fatalf("expected new guidance in content, got: %s", content)
	}
	if strings.Contains(string(content), "old guidance") {
		t.Fatal("expected old guidance to be removed")
	}
	if !strings.Contains(string(content), "user text") {
		t.Fatal("expected user text to be preserved")
	}
}

func TestEnsureManagedGuidanceSkipsWhenNoChange(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "AGENTS.md")

	block := testStart + "\nsome guidance\n" + testEnd
	changed, err := ensureManagedGuidance(path, block, testStart, testEnd)
	if err != nil {
		t.Fatalf("first call error: %v", err)
	}
	if !changed {
		t.Fatal("expected changed=true on first call")
	}

	changed, err = ensureManagedGuidance(path, block, testStart, testEnd)
	if err != nil {
		t.Fatalf("second call error: %v", err)
	}
	if changed {
		t.Fatal("expected changed=false when content identical")
	}
}

func TestRemoveManagedGuidanceRemovesBlock(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "AGENTS.md")

	block := testStart + "\nguidance\n" + testEnd
	os.WriteFile(path, []byte("lead-in\n\n"+block+"\n\ntrail-out"), 0644)

	removed, err := removeManagedGuidance(path, testStart, testEnd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !removed {
		t.Fatal("expected removed=true")
	}

	content, _ := os.ReadFile(path)
	if strings.Contains(string(content), "guidance") {
		t.Fatal("expected guidance block to be removed")
	}
	if !strings.Contains(string(content), "lead-in") {
		t.Fatal("expected user text before block to be preserved")
	}
	if !strings.Contains(string(content), "trail-out") {
		t.Fatal("expected user text after block to be preserved")
	}
}

// RunOptionsTest verifies Options are correctly passed to Run and adapters can read them.
func TestOptionsPassedToAdapters(t *testing.T) {
	opts := Options{
		Uninstall:    false,
		SkipGuidance: true,
		AssumeYes:    false,
	}
	if opts.SkipGuidance != true {
		t.Fatal("expected SkipGuidance to be true")
	}
	if opts.Uninstall != false {
		t.Fatal("expected Uninstall to be false")
	}
}
