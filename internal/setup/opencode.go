package setup

import (
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
)

func (OpenCodeAdapter) Name() string {
	return "opencode"
}

func (OpenCodeAdapter) Validate() error {
	return nil
}

func (OpenCodeAdapter) Apply(ctx Context) error {
	configPath := filepath.Join(ctx.HomeDir, ".config", "opencode", "opencode.json")
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

	agentsPath := filepath.Join(ctx.HomeDir, ".config", "opencode", "AGENTS.md")
	agentsChanged, err := ensureManagedGuidance(agentsPath, managedOpenCodeGuidance(), opencodeGuidanceStart, opencodeGuidanceEnd, "OpenCode", "opencode")
	if err != nil {
		return err
	}

	if agentsChanged {
		fmt.Printf("Updated global OpenCode guidance at %s to prefer Vectos tools.\n", agentsPath)
	}

	return nil
}

func (OpenCodeAdapter) Remove(ctx Context) error {
	configPath := filepath.Join(ctx.HomeDir, ".config", "opencode", "opencode.json")
	removedConfig, err := removeOpenCodeMCPEntry(configPath)
	if err != nil {
		return err
	}

	agentsPath := filepath.Join(ctx.HomeDir, ".config", "opencode", "AGENTS.md")
	removedGuidance, err := removeManagedGuidance(agentsPath, opencodeGuidanceStart, opencodeGuidanceEnd)
	if err != nil {
		return err
	}

	if removedConfig {
		fmt.Printf("Removed Vectos MCP entry from %s.\n", configPath)
	}
	if removedGuidance {
		fmt.Printf("Removed Vectos guidance block from %s.\n", agentsPath)
	}
	if !removedConfig && !removedGuidance {
		fmt.Println("No Vectos-managed OpenCode setup was found to remove.")
	}

	return nil
}

func managedOpenCodeGuidance() string {
	return strings.Join([]string{
		opencodeGuidanceStart,
		"## Prefer Vectos MCP",
		"",
		"When Vectos MCP tools are available for a project, prefer `vectos_search_code` before using `grep`, `find`, `glob`, or broad file reads.",
		"",
		"If the project is not yet indexed or `vectos_search_code` returns no useful results, run `vectos_index_project` and retry `vectos_search_code`.",
		"",
		"Use `grep`, `glob`, and direct file reads only as a fallback when Vectos has no useful results or when you need exact pattern matching.",
		opencodeGuidanceEnd,
	}, "\n")
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
