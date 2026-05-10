package setup

import (
	"os"
	"path/filepath"
	"strings"
)

func ensureManagedGuidance(path string, block string, startMarker string, endMarker string) (bool, error) {
	content, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return false, err
	}

	existing := string(content)
	updated, changed := upsertManagedSection(existing, block, startMarker, endMarker)
	if !changed {
		return false, nil
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


