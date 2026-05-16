package setup

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type OpenCodeAdapter struct{}

const (
	opencodeGuidanceStart = "<!-- vectos-opencode-guidance:start -->"
	opencodeGuidanceEnd   = "<!-- vectos-opencode-guidance:end -->"
	opencodePluginFile    = "vectos.ts"
	opencodeConfigDir     = ".config" // XDG-style config directory
	opencodeSkillName     = "vectos"
)

//go:embed plugins/vectos.ts
var opencodePluginSource string

//go:embed skills/vectos/SKILL.md
var opencodeSkillSource string

func (OpenCodeAdapter) Name() string {
	return "opencode"
}

func (OpenCodeAdapter) Validate() error {
	return nil
}

func (OpenCodeAdapter) Apply(ctx Context) error {
	configPath := filepath.Join(ctx.HomeDir, opencodeConfigDir, "opencode", "opencode.json")
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return err
	}

	config := map[string]interface{}{}
	if content, err := os.ReadFile(configPath); err == nil && len(strings.TrimSpace(string(content))) > 0 {
		if err := json.Unmarshal(content, &config); err != nil {
			return fmt.Errorf("failed to parse existing config: %w", err)
		}
	}

	if _, ok := config["$schema"]; !ok {
		config["$schema"] = "https://opencode.ai/config.json"
	}

	mcpConfig, ok := config["mcp"].(map[string]interface{})
	if !ok || mcpConfig == nil {
		mcpConfig = map[string]interface{}{}
	}

	mcpConfig["vectos"] = map[string]interface{}{
		"type":    "local",
		"enabled": true,
		"timeout": 10000,
		"command": []string{ctx.Executable, "mcp"},
	}
	config["mcp"] = mcpConfig

	encoded, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	encoded = append(encoded, '\n')
	if err := os.WriteFile(configPath, encoded, 0644); err != nil {
		return err
	}

	if !ctx.Options.SkipGuidance {
		agentsPath := filepath.Join(ctx.HomeDir, opencodeConfigDir, "opencode", "AGENTS.md")
		agentsChanged, err := ensureManagedGuidance(agentsPath, managedOpenCodeGuidance(), opencodeGuidanceStart, opencodeGuidanceEnd)
		if err != nil {
			return err
		}
		if agentsChanged {
			fmt.Printf("Updated global OpenCode guidance at %s to prefer Vectos tools.\n", agentsPath)
		}

		// Install the Vectos skill so agents can load detailed usage patterns
		skillsDir := filepath.Join(ctx.HomeDir, ".agents", "skills", opencodeSkillName)
		if err := os.MkdirAll(skillsDir, 0755); err != nil {
			return fmt.Errorf("failed to create skills directory: %w", err)
		}

		skillPath := filepath.Join(skillsDir, "SKILL.md")
		if err := os.WriteFile(skillPath, []byte(opencodeSkillSource), 0644); err != nil {
			return fmt.Errorf("failed to install Vectos skill: %w", err)
		}

		fmt.Printf("Installed Vectos skill at %s\n", skillPath)
	}

	// Install the OpenCode plugin for auto-reindex on file changes
	pluginsDir := filepath.Join(ctx.HomeDir, opencodeConfigDir, "opencode", "plugins")
	if err := os.MkdirAll(pluginsDir, 0755); err != nil {
		return fmt.Errorf("failed to create plugins directory: %w", err)
	}

	pluginPath := filepath.Join(pluginsDir, opencodePluginFile)
	if err := os.WriteFile(pluginPath, []byte(opencodePluginSource), 0644); err != nil {
		return fmt.Errorf("failed to install OpenCode plugin: %w", err)
	}

	fmt.Printf("Installed OpenCode plugin at %s\n", pluginPath)

	return nil
}

func (OpenCodeAdapter) Remove(ctx Context) error {
	configPath := filepath.Join(ctx.HomeDir, opencodeConfigDir, "opencode", "opencode.json")
	removedConfig, err := removeOpenCodeMCPEntry(configPath)
	if err != nil {
		return err
	}

	agentsPath := filepath.Join(ctx.HomeDir, opencodeConfigDir, "opencode", "AGENTS.md")
	removedGuidance, err := removeManagedGuidance(agentsPath, opencodeGuidanceStart, opencodeGuidanceEnd)
	if err != nil {
		return err
	}

	// Remove the OpenCode plugin
	pluginPath := filepath.Join(ctx.HomeDir, opencodeConfigDir, "opencode", "plugins", opencodePluginFile)
	removedPlugin := false
	if _, err := os.Stat(pluginPath); err == nil {
		if err := os.Remove(pluginPath); err != nil {
			fmt.Printf("Warning: failed to remove OpenCode plugin at %s: %v\n", pluginPath, err)
		} else {
			removedPlugin = true
		}
	}

	// Remove the Vectos skill
	skillPath := filepath.Join(ctx.HomeDir, ".agents", "skills", opencodeSkillName, "SKILL.md")
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
	if removedPlugin {
		fmt.Printf("Removed OpenCode plugin at %s.\n", pluginPath)
	}
	if removedSkill {
		fmt.Printf("Removed Vectos skill at %s.\n", skillPath)
	}
	if !removedConfig && !removedGuidance && !removedPlugin && !removedSkill {
		fmt.Println("No Vectos-managed OpenCode setup was found to remove.")
	}

	return nil
}

func managedOpenCodeGuidance() string {
	return managedGuidance(opencodeGuidanceStart, opencodeGuidanceEnd)
}

func removeOpenCodeMCPEntry(path string) (bool, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	config := map[string]interface{}{}
	if len(strings.TrimSpace(string(content))) > 0 {
		if err := json.Unmarshal(content, &config); err != nil {
			return false, fmt.Errorf("failed to parse existing config: %w", err)
		}
	}

	mcpConfig, ok := config["mcp"].(map[string]interface{})
	if !ok || mcpConfig == nil {
		return false, nil
	}

	if _, exists := mcpConfig["vectos"]; !exists {
		return false, nil
	}

	delete(mcpConfig, "vectos")
	if len(mcpConfig) == 0 {
		delete(config, "mcp")
	} else {
		config["mcp"] = mcpConfig
	}

	encoded, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return false, err
	}

	encoded = append(encoded, '\n')
	return true, os.WriteFile(path, encoded, 0644)
}
