package setup

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type ClaudeCodeAdapter struct{}

const (
	claudeGuidanceStart = "<!-- vectos-claude-guidance:start -->"
	claudeGuidanceEnd   = "<!-- vectos-claude-guidance:end -->"
)

func (ClaudeCodeAdapter) Name() string { return "claude" }

func (ClaudeCodeAdapter) Validate() error { return nil }

func (ClaudeCodeAdapter) Apply(ctx Context) error {
	configPath := filepath.Join(ctx.HomeDir, ".claude.json")
	config, err := readJSONConfig(configPath)
	if err != nil {
		return err
	}

	mcpServers, ok := config["mcpServers"].(map[string]interface{})
	if !ok || mcpServers == nil {
		mcpServers = map[string]interface{}{}
	}

	mcpServers["vectos"] = map[string]interface{}{
		"command": ctx.Executable,
		"args":    []string{"mcp"},
	}
	config["mcpServers"] = mcpServers

	if err := writeJSONConfig(configPath, config); err != nil {
		return err
	}

	claudePath := filepath.Join(ctx.HomeDir, ".claude", "CLAUDE.md")
	changed, err := ensureManagedGuidance(claudePath, managedClaudeGuidance(), claudeGuidanceStart, claudeGuidanceEnd, "Claude Code", "claude")
	if err != nil {
		return err
	}
	if changed {
		fmt.Printf("Updated global Claude Code guidance at %s to prefer Vectos tools.\n", claudePath)
	}

	return nil
}

func (ClaudeCodeAdapter) Remove(ctx Context) error {
	configPath := filepath.Join(ctx.HomeDir, ".claude.json")
	removedConfig, err := removeJSONMCPEntry(configPath, "mcpServers", "vectos")
	if err != nil {
		return err
	}

	claudePath := filepath.Join(ctx.HomeDir, ".claude", "CLAUDE.md")
	removedGuidance, err := removeManagedGuidance(claudePath, claudeGuidanceStart, claudeGuidanceEnd)
	if err != nil {
		return err
	}

	if removedConfig {
		fmt.Printf("Removed Vectos MCP entry from %s.\n", configPath)
	}
	if removedGuidance {
		fmt.Printf("Removed Vectos guidance block from %s.\n", claudePath)
	}
	if !removedConfig && !removedGuidance {
		fmt.Println("No Vectos-managed Claude Code setup was found to remove.")
	}

	return nil
}

func managedClaudeGuidance() string {
	return strings.Join([]string{
		claudeGuidanceStart,
		"## Prefer Vectos MCP",
		"",
		"When Vectos MCP tools are available for a project, prefer `vectos_search_code` before using `grep`, `find`, `glob`, or broad file reads.",
		"",
		"If the project is not yet indexed or `vectos_search_code` returns no useful results, run `vectos_index_project` and retry `vectos_search_code`.",
		"",
		"Use `grep`, `glob`, and direct file reads only as a fallback when Vectos has no useful results or when you need exact pattern matching.",
		claudeGuidanceEnd,
	}, "\n")
}

func readJSONConfig(path string) (map[string]interface{}, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]interface{}{}, nil
		}
		return nil, err
	}

	config := map[string]interface{}{}
	if len(strings.TrimSpace(string(content))) > 0 {
		if err := json.Unmarshal(content, &config); err != nil {
			return nil, fmt.Errorf("failed to parse existing config: %w", err)
		}
	}

	return config, nil
}

func writeJSONConfig(path string, config map[string]interface{}) error {
	encoded, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	encoded = append(encoded, '\n')
	return os.WriteFile(path, encoded, 0644)
}

func removeJSONMCPEntry(path string, containerKey string, serverKey string) (bool, error) {
	config, err := readJSONConfig(path)
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

	return true, writeJSONConfig(path, config)
}
