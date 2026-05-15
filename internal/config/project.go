package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// IndexExclusionConfig holds per-mode exclusion patterns.
type IndexExclusionConfig struct {
	Exclude []string `json:"exclude,omitempty"`
}

// IndexConfig holds index-level configuration for docs and code.
type IndexConfig struct {
	Docs IndexExclusionConfig `json:"docs"`
	Code IndexExclusionConfig `json:"code"`
}

// projectConfigDisk is the on-disk format of vectos.config.json.
type projectConfigDisk struct {
	Index struct {
		Docs struct {
			Exclude []string `json:"exclude,omitempty"`
		} `json:"docs"`
		Code struct {
			Exclude []string `json:"exclude,omitempty"`
		} `json:"code"`
	} `json:"index"`
}

// LoadIndexConfig merges global defaults with project-level exclusion patterns.
// Global patterns come from ~/.vectos/config.json (index section).
// Project patterns come from vectos.config.json in the project root.
// gitignore patterns are merged in separately via ReadGitignorePatterns.
// Returns empty config if neither file exists.
func LoadIndexConfig(globalConfigPath, projectDir string) IndexConfig {
	cfg := IndexConfig{}

	// Layer 1: Global defaults (~/.vectos/config.json index section)
	if content, err := os.ReadFile(globalConfigPath); err == nil {
		var globalCfg struct {
			Index struct {
				Docs struct {
					Exclude []string `json:"exclude,omitempty"`
				} `json:"docs"`
				Code struct {
					Exclude []string `json:"exclude,omitempty"`
				} `json:"code"`
			} `json:"index"`
		}
		if json.Unmarshal(content, &globalCfg) == nil {
			cfg.Docs.Exclude = append(cfg.Docs.Exclude, globalCfg.Index.Docs.Exclude...)
			cfg.Code.Exclude = append(cfg.Code.Exclude, globalCfg.Index.Code.Exclude...)
		}
	}

	// Layer 2: Project config (vectos.config.json in project root)
	projectConfigPath := filepath.Join(projectDir, "vectos.config.json")
	if content, err := os.ReadFile(projectConfigPath); err == nil {
		var pc projectConfigDisk
		if json.Unmarshal(content, &pc) == nil {
			cfg.Docs.Exclude = append(cfg.Docs.Exclude, pc.Index.Docs.Exclude...)
			cfg.Code.Exclude = append(cfg.Code.Exclude, pc.Index.Code.Exclude...)
		}
	}

	return cfg
}

// exclusionPatternsForMode returns the merged exclusion patterns for the given mode
// (docs or code). Patterns are deduplicated.
func (c IndexConfig) ExclusionPatterns(docsOnly bool) []string {
	var patterns []string
	if docsOnly {
		patterns = append(patterns, c.Docs.Exclude...)
	} else {
		patterns = append(patterns, c.Code.Exclude...)
	}
	return dedupeStrings(patterns)
}

// ReadGitignorePatterns reads .gitignore from the project root and returns
// usable file-matching patterns. Negation patterns (!) are skipped.
// Directory-only patterns (trailing /) are preserved.
func ReadGitignorePatterns(projectDir string) []string {
	gitignorePath := filepath.Join(projectDir, ".gitignore")
	content, err := os.ReadFile(gitignorePath)
	if err != nil {
		return nil
	}

	lines := strings.Split(string(content), "\n")
	var patterns []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "!") {
			continue
		}
		if strings.HasPrefix(line, "/") {
			line = line[1:]
		}
		patterns = append(patterns, line)
	}
	return patterns
}

func dedupeStrings(in []string) []string {
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, s := range in {
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}
