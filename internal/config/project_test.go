package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadIndexConfig_MissingFiles(t *testing.T) {
	cfg := LoadIndexConfig("/nonexistent/global.json", "/nonexistent/project")
	if len(cfg.Docs.Exclude) != 0 {
		t.Errorf("expected empty docs exclusions, got %v", cfg.Docs.Exclude)
	}
	if len(cfg.Code.Exclude) != 0 {
		t.Errorf("expected empty code exclusions, got %v", cfg.Code.Exclude)
	}
}

func TestLoadIndexConfig_GlobalConfigOnly(t *testing.T) {
	dir := t.TempDir()
	globalCfg := `{"index":{"docs":{"exclude":[".agents/**"]},"code":{"exclude":["**/generated/**"]}}}`
	globalPath := filepath.Join(dir, "global.json")
	if err := os.WriteFile(globalPath, []byte(globalCfg), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := LoadIndexConfig(globalPath, filepath.Join(dir, "project"))
	if len(cfg.Docs.Exclude) != 1 || cfg.Docs.Exclude[0] != ".agents/**" {
		t.Errorf("expected [.agents/**], got %v", cfg.Docs.Exclude)
	}
	if len(cfg.Code.Exclude) != 1 || cfg.Code.Exclude[0] != "**/generated/**" {
		t.Errorf("expected [**/generated/**], got %v", cfg.Code.Exclude)
	}
}

func TestLoadIndexConfig_ProjectConfigAppendedToGlobal(t *testing.T) {
	dir := t.TempDir()
	globalCfg := `{"index":{"docs":{"exclude":[".agents/**"]},"code":{"exclude":[]}}}`
	globalPath := filepath.Join(dir, "global.json")
	if err := os.WriteFile(globalPath, []byte(globalCfg), 0644); err != nil {
		t.Fatal(err)
	}

	projectDir := filepath.Join(dir, "project")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatal(err)
	}
	projectCfg := `{"index":{"docs":{"exclude":["src/content/blog/**"]},"code":{"exclude":["**/__mocks__/**"]}}}`
	if err := os.WriteFile(filepath.Join(projectDir, "vectos.config.json"), []byte(projectCfg), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := LoadIndexConfig(globalPath, projectDir)
	if len(cfg.Docs.Exclude) != 2 {
		t.Fatalf("expected 2 docs exclusions, got %v", cfg.Docs.Exclude)
	}
	if cfg.Docs.Exclude[0] != ".agents/**" || cfg.Docs.Exclude[1] != "src/content/blog/**" {
		t.Errorf("global should come before project, got %v", cfg.Docs.Exclude)
	}
	if len(cfg.Code.Exclude) != 1 || cfg.Code.Exclude[0] != "**/__mocks__/**" {
		t.Errorf("expected [**/__mocks__/**], got %v", cfg.Code.Exclude)
	}
}

func TestExclusionPatterns_DocsOnlyMode(t *testing.T) {
	cfg := IndexConfig{
		Docs: IndexExclusionConfig{Exclude: []string{"blog/**"}},
		Code: IndexExclusionConfig{Exclude: []string{"**/__mocks__/**"}},
	}
	patterns := cfg.ExclusionPatterns(true)
	if len(patterns) != 1 || patterns[0] != "blog/**" {
		t.Errorf("expected [blog/**], got %v", patterns)
	}
}

func TestExclusionPatterns_CodeMode(t *testing.T) {
	cfg := IndexConfig{
		Docs: IndexExclusionConfig{Exclude: []string{"blog/**"}},
		Code: IndexExclusionConfig{Exclude: []string{"**/__mocks__/**"}},
	}
	patterns := cfg.ExclusionPatterns(false)
	if len(patterns) != 1 || patterns[0] != "**/__mocks__/**" {
		t.Errorf("expected [**/__mocks__/**], got %v", patterns)
	}
}

func TestExclusionPatterns_Deduplication(t *testing.T) {
	cfg := IndexConfig{
		Code: IndexExclusionConfig{Exclude: []string{"*.log", "*.log"}},
	}
	patterns := cfg.ExclusionPatterns(false)
	if len(patterns) != 1 {
		t.Errorf("expected deduped 1 pattern, got %v", patterns)
	}
}

func TestReadGitignorePatterns_MissingFile(t *testing.T) {
	patterns := ReadGitignorePatterns("/nonexistent/dir")
	if patterns != nil {
		t.Errorf("expected nil, got %v", patterns)
	}
}

func TestReadGitignorePatterns_BasicParsing(t *testing.T) {
	dir := t.TempDir()
	gitignoreContent := "# comment line\n*.log\ndist/\n!important.log\n/node_modules\n"
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte(gitignoreContent), 0644); err != nil {
		t.Fatal(err)
	}

	patterns := ReadGitignorePatterns(dir)
	expected := []string{"*.log", "dist/", "node_modules"}
	if len(patterns) != len(expected) {
		t.Fatalf("expected %v, got %v", expected, patterns)
	}
	for i, p := range expected {
		if patterns[i] != p {
			t.Errorf("pattern[%d] = %q, want %q", i, patterns[i], p)
		}
	}
}

func TestReadGitignorePatterns_LeadsSlashStripped(t *testing.T) {
	dir := t.TempDir()
	gitignoreContent := "/dist\n/build\n/coverage\n"
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte(gitignoreContent), 0644); err != nil {
		t.Fatal(err)
	}

	patterns := ReadGitignorePatterns(dir)
	for _, p := range patterns {
		if p[0] == '/' {
			t.Errorf("pattern %q should not start with /", p)
		}
	}
}
