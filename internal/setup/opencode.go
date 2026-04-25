package setup

import (
	"bufio"
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
	agentsChanged, err := ensureOpenCodeGuidance(agentsPath)
	if err != nil {
		return err
	}

	if agentsChanged {
		fmt.Printf("Actualizada la guía global de OpenCode en %s para priorizar Vectos.\n", agentsPath)
	}

	return nil
}

func ensureOpenCodeGuidance(path string) (bool, error) {
	content, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return false, err
	}

	existing := string(content)
	section := managedOpenCodeGuidance()
	updated, changed := upsertManagedSection(existing, section)
	if !changed {
		return false, nil
	}

	if existing == "" && !isInteractiveTerminal() {
		fmt.Printf("Guía global de OpenCode omitida en modo no interactivo. Añádela después en %s si quieres priorizar Vectos por defecto.\n", path)
		return false, nil
	}

	if existing != "" && !strings.Contains(existing, opencodeGuidanceStart) {
		if !confirmInstallOpenCodeGuidance(path) {
			fmt.Printf("Guía global de OpenCode no modificada. Puedes volver a ejecutar 'vectos setup opencode' cuando quieras añadirla.\n")
			return false, nil
		}
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return false, err
	}

	return true, os.WriteFile(path, []byte(updated), 0644)
}

func managedOpenCodeGuidance() string {
	return strings.Join([]string{
		opencodeGuidanceStart,
		"## Preferir Vectos MCP",
		"",
		"Cuando un proyecto tenga disponibles las herramientas MCP de Vectos, prioriza `vectos_search_code` antes de usar `grep`, `find`, `glob` o lecturas amplias de archivos.",
		"",
		"Si el código aún no está indexado o `vectos_search_code` no devuelve nada útil, usa `vectos_index_project` y vuelve a intentar `vectos_search_code`.",
		"",
		"Usa `grep`, `glob` y lecturas directas de archivos solo como fallback cuando Vectos no tenga resultados útiles o cuando necesites coincidencia exacta.",
		opencodeGuidanceEnd,
	}, "\n")
}

func upsertManagedSection(existing string, section string) (string, bool) {
	start := strings.Index(existing, opencodeGuidanceStart)
	end := strings.Index(existing, opencodeGuidanceEnd)
	if start >= 0 && end >= start {
		end += len(opencodeGuidanceEnd)
		updated := existing[:start] + section + existing[end:]
		updated = strings.TrimSpace(updated) + "\n"
		return updated, updated != existing
	}

	trimmed := strings.TrimSpace(existing)
	if trimmed == "" {
		return section + "\n", true
	}

	updated := trimmed + "\n\n" + section + "\n"
	return updated, true
}

func confirmInstallOpenCodeGuidance(path string) bool {
	if !isInteractiveTerminal() {
		fmt.Printf("Detectada configuración global existente en %s. Reejecuta el setup en una terminal interactiva para decidir si quieres añadir la guía global de Vectos.\n", path)
		return false
	}

	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("¿Quieres añadir una guía global en %s para que OpenCode priorice Vectos antes de grep/find? [Y/n]: ", path)
	answer, err := reader.ReadString('\n')
	if err != nil {
		return false
	}

	answer = strings.ToLower(strings.TrimSpace(answer))
	return answer == "" || answer == "y" || answer == "yes" || answer == "s" || answer == "si"
}

func isInteractiveTerminal() bool {
	info, err := os.Stdin.Stat()
	if err != nil {
		return false
	}

	return (info.Mode() & os.ModeCharDevice) != 0
}
