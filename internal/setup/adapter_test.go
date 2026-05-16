package setup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOpenCodeAdapterInstallsSkill(t *testing.T) {
	dir := t.TempDir()
	ctx := Context{Executable: "mock-vectos", HomeDir: dir}

	if err := (OpenCodeAdapter{}).Apply(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	skillPath := filepath.Join(dir, ".agents", "skills", vectosSkillName, "SKILL.md")
	content, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatalf("expected skill file to exist: %v", err)
	}
	if string(content) != vectosSkillSource {
		t.Fatalf("unexpected skill content:\nwant: %q\ngot:  %q", vectosSkillSource, string(content))
	}
}

func TestOpenCodeAdapterRemovesSkill(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".config", "opencode", "opencode.json")
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}
	if err := os.WriteFile(configPath, []byte(`{"mcp":{"vectos":{"type":"local","enabled":true,"timeout":10000,"command":["mock-vectos","mcp"]}}}`), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	skillPath := filepath.Join(dir, ".agents", "skills", vectosSkillName, "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(skillPath), 0755); err != nil {
		t.Fatalf("mkdir skill dir: %v", err)
	}
	if err := os.WriteFile(skillPath, []byte(vectosSkillSource), 0644); err != nil {
		t.Fatalf("write skill: %v", err)
	}

	if err := (OpenCodeAdapter{}).Remove(Context{HomeDir: dir}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := os.Stat(skillPath); !os.IsNotExist(err) {
		t.Fatalf("expected skill file removed, stat err=%v", err)
	}
}

func TestOpenCodeAdapterInstallsSkillWithSkipGuidance(t *testing.T) {
	dir := t.TempDir()
	ctx := Context{Executable: "mock-vectos", HomeDir: dir, Options: Options{SkipGuidance: true}}

	if err := (OpenCodeAdapter{}).Apply(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	skillPath := filepath.Join(dir, ".agents", "skills", vectosSkillName, "SKILL.md")
	content, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatalf("expected skill file to exist: %v", err)
	}
	if !strings.Contains(string(content), vectosSkillSource) && string(content) != vectosSkillSource {
		t.Fatalf("unexpected skill content: %q", string(content))
	}
}

func TestCodexAdapterInstallsSkill(t *testing.T) {
	dir := t.TempDir()
	ctx := Context{Executable: "mock-vectos", HomeDir: dir}

	if err := (CodexAdapter{}).Apply(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	skillPath := filepath.Join(dir, ".codex", "skills", vectosSkillName, "SKILL.md")
	content, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatalf("expected skill file to exist: %v", err)
	}
	if string(content) != vectosSkillSource {
		t.Fatalf("unexpected skill content:\nwant: %q\ngot:  %q", vectosSkillSource, string(content))
	}
}

func TestCodexAdapterRemovesSkill(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".codex", "config.toml")
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}
	if err := os.WriteFile(configPath, []byte("[mcp_servers.vectos]\ncommand = \"mock-vectos\"\nargs = [\"mcp\"]\n"), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	skillPath := filepath.Join(dir, ".codex", "skills", vectosSkillName, "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(skillPath), 0755); err != nil {
		t.Fatalf("mkdir skill dir: %v", err)
	}
	if err := os.WriteFile(skillPath, []byte(vectosSkillSource), 0644); err != nil {
		t.Fatalf("write skill: %v", err)
	}

	if err := (CodexAdapter{}).Remove(Context{HomeDir: dir}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := os.Stat(skillPath); !os.IsNotExist(err) {
		t.Fatalf("expected skill file removed, stat err=%v", err)
	}
}
