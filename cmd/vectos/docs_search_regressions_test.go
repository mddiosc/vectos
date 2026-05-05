package main

import (
	"os"
	"path/filepath"
	"testing"

	"vectos/internal/config"
	"vectos/internal/storage"
)

func TestExecuteSearchForCLIUsesCodeSearchByDefault(t *testing.T) {
	originalCodeSearch := cliCodeSearch
	originalDocsSearch := cliDocsSearch
	defer func() {
		cliCodeSearch = originalCodeSearch
		cliDocsSearch = originalDocsSearch
	}()

	codeCalled := false
	docsCalled := false
	cliCodeSearch = func(store *storage.SQLiteStorage, embedConfig config.EmbeddingConfig, query string, limit int) (searchRun, error) {
		codeCalled = true
		return searchRun{Mode: "code"}, nil
	}
	cliDocsSearch = func(store *storage.SQLiteStorage, embedConfig config.EmbeddingConfig, query string, limit int) (searchRun, error) {
		docsCalled = true
		return searchRun{Mode: "docs"}, nil
	}

	run, err := executeSearchForCLI(nil, config.EmbeddingConfig{}, "query", false)
	if err != nil {
		t.Fatalf("executeSearchForCLI returned error: %v", err)
	}
	if !codeCalled {
		t.Fatal("expected code search to be used for normal CLI search")
	}
	if docsCalled {
		t.Fatal("did not expect docs search for normal CLI search")
	}
	if run.Mode != "code" {
		t.Fatalf("unexpected mode: %q", run.Mode)
	}
}

func TestExecuteSearchForCLIUsesDocsSearchWhenRequested(t *testing.T) {
	originalCodeSearch := cliCodeSearch
	originalDocsSearch := cliDocsSearch
	defer func() {
		cliCodeSearch = originalCodeSearch
		cliDocsSearch = originalDocsSearch
	}()

	codeCalled := false
	docsCalled := false
	cliCodeSearch = func(store *storage.SQLiteStorage, embedConfig config.EmbeddingConfig, query string, limit int) (searchRun, error) {
		codeCalled = true
		return searchRun{Mode: "code"}, nil
	}
	cliDocsSearch = func(store *storage.SQLiteStorage, embedConfig config.EmbeddingConfig, query string, limit int) (searchRun, error) {
		docsCalled = true
		return searchRun{Mode: "docs"}, nil
	}

	run, err := executeSearchForCLI(nil, config.EmbeddingConfig{}, "query", true)
	if err != nil {
		t.Fatalf("executeSearchForCLI returned error: %v", err)
	}
	if codeCalled {
		t.Fatal("did not expect code search when docsOnly is true")
	}
	if !docsCalled {
		t.Fatal("expected docs search to be used for docs CLI search")
	}
	if run.Mode != "docs" {
		t.Fatalf("unexpected mode: %q", run.Mode)
	}
}

func TestDocsIndexHasChunksResolvesNilScope(t *testing.T) {
	workdir := t.TempDir()
	baseDir := filepath.Join(t.TempDir(), "vectos-projects")

	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	defer func() {
		if chdirErr := os.Chdir(originalWD); chdirErr != nil {
			t.Fatalf("restore cwd: %v", chdirErr)
		}
	}()
	if err := os.Chdir(workdir); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	pm, err := storage.NewProjectManager(baseDir)
	if err != nil {
		t.Fatalf("new project manager: %v", err)
	}

	hasDocs, err := docsIndexHasChunks(pm, nil)
	if err != nil {
		t.Fatalf("docsIndexHasChunks returned error with nil scope: %v", err)
	}
	if hasDocs {
		t.Fatal("expected empty docs index for fresh project")
	}

	store, err := openStorageForScope(pm, nil, true)
	if err != nil {
		t.Fatalf("openStorageForScope returned error: %v", err)
	}
	defer store.Close()

	if _, err := store.SaveChunk(storage.CodeChunk{
		FilePath:  filepath.Join(workdir, "README.md"),
		Content:   "# Project\n\nGetting started",
		StartLine: 1,
		EndLine:   3,
		Language:  "markdown",
		Category:  "docs",
	}); err != nil {
		t.Fatalf("save chunk: %v", err)
	}

	hasDocs, err = docsIndexHasChunks(pm, nil)
	if err != nil {
		t.Fatalf("docsIndexHasChunks returned error after seeding docs: %v", err)
	}
	if !hasDocs {
		t.Fatal("expected docs index to be detected after saving a docs chunk")
	}
}
