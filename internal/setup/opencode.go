package setup

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type OpenCodeAdapter struct{}

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
	return os.WriteFile(configPath, encoded, 0644)
}
