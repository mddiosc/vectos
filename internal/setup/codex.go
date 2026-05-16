package setup

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	toml "github.com/pelletier/go-toml/v2"
)

type CodexAdapter struct{}

const (
	codexGuidanceStart = "<!-- vectos-codex-guidance:start -->"
	codexGuidanceEnd   = "<!-- vectos-codex-guidance:end -->"
)

func (CodexAdapter) Name() string { return "codex" }

func (CodexAdapter) Validate() error { return nil }

func (CodexAdapter) Apply(ctx Context) error {
	configPath := filepath.Join(ctx.HomeDir, ".codex", "config.toml")
	config, err := readTOMLConfig(configPath)
	if err != nil {
		return err
	}

	mcpServers, ok := config["mcp_servers"].(map[string]interface{})
	if !ok || mcpServers == nil {
		mcpServers = map[string]interface{}{}
	}

	mcpServers["vectos"] = map[string]interface{}{
		"command": ctx.Executable,
		"args":    []string{"mcp"},
	}
	config["mcp_servers"] = mcpServers

	if err := writeTOMLConfig(configPath, config); err != nil {
		return err
	}

	if !ctx.Options.SkipGuidance {
		agentsPath := filepath.Join(ctx.HomeDir, ".codex", "AGENTS.md")
		changed, err := ensureManagedGuidance(agentsPath, managedCodexGuidance(), codexGuidanceStart, codexGuidanceEnd)
		if err != nil {
			return err
		}
		if changed {
			fmt.Printf("Updated global Codex guidance at %s to prefer Vectos tools.\n", agentsPath)
		}

		// Install the Vectos skill so agents can load detailed usage patterns
		skillsDir := filepath.Join(ctx.HomeDir, ".codex", "skills", opencodeSkillName)
		if err := os.MkdirAll(skillsDir, 0755); err != nil {
			return fmt.Errorf("failed to create skills directory: %w", err)
		}

		skillPath := filepath.Join(skillsDir, "SKILL.md")
		if err := os.WriteFile(skillPath, []byte(opencodeSkillSource), 0644); err != nil {
			return fmt.Errorf("failed to install Vectos skill: %w", err)
		}

		fmt.Printf("Installed Vectos skill at %s\n", skillPath)
	}

	return nil
}

func (CodexAdapter) Remove(ctx Context) error {
	configPath := filepath.Join(ctx.HomeDir, ".codex", "config.toml")
	removedConfig, err := removeTOMLMCPEntry(configPath, "mcp_servers", "vectos")
	if err != nil {
		return err
	}

	agentsPath := filepath.Join(ctx.HomeDir, ".codex", "AGENTS.md")
	removedGuidance, err := removeManagedGuidance(agentsPath, codexGuidanceStart, codexGuidanceEnd)
	if err != nil {
		return err
	}

	// Remove the Vectos skill
	skillPath := filepath.Join(ctx.HomeDir, ".codex", "skills", opencodeSkillName, "SKILL.md")
	removedSkill := false
	if _, err := os.Stat(skillPath); err == nil {
		if err := os.Remove(skillPath); err != nil {
			fmt.Printf("Warning: failed to remove Vectos skill at %s: %v\n", skillPath, err)
		} else {
			removedSkill = true
		}
	}

	if removedConfig {
		fmt.Printf("Removed Vectos MCP entry from %s.\n", configPath)
	}
	if removedGuidance {
		fmt.Printf("Removed Vectos guidance block from %s.\n", agentsPath)
	}
	if removedSkill {
		fmt.Printf("Removed Vectos skill at %s.\n", skillPath)
	}
	if !removedConfig && !removedGuidance && !removedSkill {
		fmt.Println("No Vectos-managed Codex setup was found to remove.")
	}

	return nil
}

func managedCodexGuidance() string {
	return managedGuidance(codexGuidanceStart, codexGuidanceEnd)
}

func readTOMLConfig(path string) (map[string]interface{}, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]interface{}{}, nil
		}
		return nil, err
	}

	config := map[string]interface{}{}
	if len(strings.TrimSpace(string(content))) > 0 {
		if err := toml.Unmarshal(content, &config); err != nil {
			return nil, fmt.Errorf("failed to parse existing config: %w", err)
		}
	}

	return config, nil
}

func writeTOMLConfig(path string, config map[string]interface{}) error {
	encoded, err := toml.Marshal(config)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, encoded, 0644)
}

func removeTOMLMCPEntry(path string, containerKey string, serverKey string) (bool, error) {
	config, err := readTOMLConfig(path)
	if err != nil {
		return false, err
	}

	servers, ok := config[containerKey].(map[string]interface{})
	if !ok || servers == nil {
		return false, nil
	}
	if _, exists := servers[serverKey]; !exists {
		return false, nil
	}

	delete(servers, serverKey)
	if len(servers) == 0 {
		delete(config, containerKey)
	} else {
		config[containerKey] = servers
	}

	return true, writeTOMLConfig(path, config)
}
