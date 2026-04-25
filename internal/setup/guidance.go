package setup

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func ensureManagedGuidance(path string, block string, startMarker string, endMarker string, product string, commandName string) (bool, error) {
	content, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return false, err
	}

	existing := string(content)
	updated, changed := upsertManagedSection(existing, block, startMarker, endMarker)
	if !changed {
		return false, nil
	}

	if existing == "" && !isInteractiveTerminal() {
		fmt.Printf("%s global guidance skipped (non-interactive mode). Add it later at %s to prefer Vectos by default.\n", product, path)
		return false, nil
	}

	if existing != "" && !strings.Contains(existing, startMarker) {
		if !confirmInstallGuidance(path, product) {
			fmt.Printf("%s global guidance not modified. Re-run 'vectos setup %s' to add it later.\n", product, commandName)
			return false, nil
		}
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return false, err
	}

	return true, os.WriteFile(path, []byte(updated), 0644)
}

func removeManagedGuidance(path string, startMarker string, endMarker string) (bool, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	updated, changed := removeManagedSection(string(content), startMarker, endMarker)
	if !changed {
		return false, nil
	}

	return true, os.WriteFile(path, []byte(updated), 0644)
}

func upsertManagedSection(existing string, section string, startMarker string, endMarker string) (string, bool) {
	start := strings.Index(existing, startMarker)
	end := strings.Index(existing, endMarker)
	if start >= 0 && end >= start {
		end += len(endMarker)
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

func removeManagedSection(existing string, startMarker string, endMarker string) (string, bool) {
	start := strings.Index(existing, startMarker)
	end := strings.Index(existing, endMarker)
	if start < 0 || end < start {
		return existing, false
	}

	end += len(endMarker)
	updated := existing[:start] + existing[end:]
	updated = strings.TrimSpace(updated)
	if updated == "" {
		return "", true
	}

	return updated + "\n", true
}

func confirmInstallGuidance(path string, product string) bool {
	if !isInteractiveTerminal() {
		fmt.Printf("Existing global config found at %s. Re-run setup in an interactive terminal to decide whether to add Vectos guidance for %s.\n", path, product)
		return false
	}

	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("Add global guidance at %s so %s prefers Vectos before generic search tools? [Y/n]: ", path, product)
	answer, err := reader.ReadString('\n')
	if err != nil {
		return false
	}

	answer = strings.ToLower(strings.TrimSpace(answer))
	return answer == "" || answer == "y" || answer == "yes"
}

func isInteractiveTerminal() bool {
	info, err := os.Stdin.Stat()
	if err != nil {
		return false
	}

	return (info.Mode() & os.ModeCharDevice) != 0
}
