package config

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// GlobalConfigPath returns the absolute path to the user-level vectos
// configuration file (~/.vectos/config.json). The file is optional; callers
// should treat an empty return path as "no global layer available" and skip
// loading the global config rather than falling back to a project-local path.
//
// Returns an error when the user's home directory cannot be determined.
func GlobalConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not determine home directory: %w", err)
	}
	return filepath.Join(homeDir, ".vectos", "config.json"), nil
}

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
//
// An empty globalConfigPath disables the global layer entirely (recommended
// when GlobalConfigPath() failed — do not substitute a project-local fallback,
// that defeats the point of having a user-level layer).
// Returns empty config if neither file exists.
func LoadIndexConfig(globalConfigPath, projectDir string) IndexConfig {
	cfg := IndexConfig{}

	// Layer 1: Global defaults (~/.vectos/config.json index section).
	// Skipped when globalConfigPath is empty.
	if globalConfigPath != "" {
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
			if err := parseJSONC(content, &globalCfg); err != nil {
				log.Printf("warning: invalid JSON in %s: %v", globalConfigPath, err)
			} else {
				cfg.Docs.Exclude = append(cfg.Docs.Exclude, globalCfg.Index.Docs.Exclude...)
				cfg.Code.Exclude = append(cfg.Code.Exclude, globalCfg.Index.Code.Exclude...)
			}
		}
	}

	// Layer 2: Project config (vectos.config.json in project root)
	projectConfigPath := filepath.Join(projectDir, "vectos.config.json")
	if content, err := os.ReadFile(projectConfigPath); err == nil {
		var pc projectConfigDisk
		if err := parseJSONC(content, &pc); err != nil {
			log.Printf("warning: invalid JSON in %s: %v", projectConfigPath, err)
		} else {
			cfg.Docs.Exclude = append(cfg.Docs.Exclude, pc.Index.Docs.Exclude...)
			cfg.Code.Exclude = append(cfg.Code.Exclude, pc.Index.Code.Exclude...)
		}
	}

	return cfg
}

// ExclusionPatterns returns the merged exclusion patterns for the given mode
// (docs or code). Patterns are deduplicated. Note: gitignore patterns are
// merged in by callers AFTER this returns, so deduplication of the combined
// set should be performed via MergeExclusionPatterns at the call site.
func (c IndexConfig) ExclusionPatterns(docsOnly bool) []string {
	var patterns []string
	if docsOnly {
		patterns = append(patterns, c.Docs.Exclude...)
	} else {
		patterns = append(patterns, c.Code.Exclude...)
	}
	return dedupeStrings(patterns)
}

// MergeExclusionPatterns concatenates exclusion sources in priority order and
// deduplicates the final set. Use this at call sites that combine config
// patterns with .gitignore patterns (or any other source); deduplicating after
// the merge is the only place that catches overlap between sources.
func MergeExclusionPatterns(sources ...[]string) []string {
	total := 0
	for _, s := range sources {
		total += len(s)
	}
	if total == 0 {
		return nil
	}
	combined := make([]string, 0, total)
	for _, s := range sources {
		combined = append(combined, s...)
	}
	return dedupeStrings(combined)
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
		line = strings.TrimPrefix(line, "/")
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
