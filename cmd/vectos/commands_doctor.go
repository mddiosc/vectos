package main

import (
	"fmt"
	"os"
	"path/filepath"

	"vectos/internal/buildinfo"
	"vectos/internal/config"
	"vectos/internal/embeddings"
	"vectos/internal/storage"
)

type doctorSummary struct {
	passes   int
	warnings int
	fails    int
}

// osExit allows tests to intercept os.Exit calls.
var osExit = os.Exit

func runDoctorCommand(app appContext, args []string) {
	if hasHelpFlag(args) {
		printSubcommandHelp("doctor")
		osExit(0)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error resolving home directory: %v\n", err)
		osExit(1)
	}

	s := &doctorSummary{}

	fmt.Println("Vectos Doctor")
	fmt.Println("=============")

	runInstallChecks(home, s)
	runProviderChecks(app.embedConfig, s)
	runIndexChecks(app.projectBaseDir, app.embedConfig, s)

	printDoctorFooter(s)

	if s.fails > 0 {
		osExit(1)
	}
}

// ---------------------------------------------------------------------------
// Install / runtime checks
// ---------------------------------------------------------------------------

func runInstallChecks(home string, s *doctorSummary) {
	fmt.Printf("\nInstall / Runtime:\n")

	// version
	fmt.Printf("  Version: %s (commit: %s, built: %s)\n", buildinfo.Version, buildinfo.Commit, buildinfo.Date)
	s.passes++

	// config file
	configPath := filepath.Join(home, ".vectos", "config.json")
	if _, err := os.Stat(configPath); err == nil {
		checkPass("Config file: %s", configPath)
		s.passes++
	} else if os.IsNotExist(err) {
		checkWarn("Config file: %s (using defaults)", configPath)
		s.warnings++
	} else {
		checkWarn("Config file: %s: %v", configPath, err)
		s.warnings++
	}

	// projects directory
	projectsDir := fmt.Sprintf("%s/.vectos/projects", home)
	if info, err := os.Stat(projectsDir); err == nil && info.IsDir() {
		checkPass("Projects directory: %s", projectsDir)
		s.passes++
	} else {
		checkWarn("Projects directory: %s (not found)", projectsDir)
		checkHint("mkdir -p %s", projectsDir)
		s.warnings++
	}

	// model cache
	embedConfig, err := config.LoadEmbeddingConfig(home)
	if err == nil && embedConfig.Embedded.Enabled && embedConfig.Embedded.ModelDir != "" {
		if info, err := os.Stat(embedConfig.Embedded.ModelDir); err == nil && info.IsDir() {
			checkPass("Model cache: %s", embedConfig.Embedded.ModelDir)
			s.passes++
		} else {
			checkWarn("Model cache: %s (not downloaded yet)", embedConfig.Embedded.ModelDir)
			checkHint("Run 'vectos index .' to auto-download the embedding model")
			s.warnings++
		}
	}
}

// ---------------------------------------------------------------------------
// Provider checks
// ---------------------------------------------------------------------------

func runProviderChecks(embedConfig config.EmbeddingConfig, s *doctorSummary) {
	fmt.Printf("\nEmbedding Provider:\n")

	statuses := embeddings.InspectProviders(embedConfig)
	if len(statuses) == 0 {
		checkFail("No embedding providers configured")
		checkHint("Check embedding configuration in ~/.vectos/config.json")
		s.fails++
		return
	}

	readyCount := 0
	for _, p := range statuses {
		if p.Ready {
			checkPass("%s (%s): ready", p.Provider, p.Model)
			s.passes++
			readyCount++
		} else {
			msg := p.Message
			if msg == "" {
				msg = "not ready"
			}
			checkFail("%s (%s): %s", p.Provider, p.Model, msg)
			checkHint("Run 'vectos index .' to download the model or check provider config")
			s.fails++
		}
	}

	// Only count provider failure as critical if NO providers are ready
	if readyCount == 0 && s.fails > 0 {
		s.fails = 1 // one critical failure, not per-provider
	} else if readyCount > 0 && s.fails > 0 {
		// Some providers are working — demote to warnings
		s.warnings += s.fails
		s.fails = 0
	}
}

// ---------------------------------------------------------------------------
// Index checks
// ---------------------------------------------------------------------------

func runIndexChecks(projectBaseDir string, embedConfig config.EmbeddingConfig, s *doctorSummary) {
	fmt.Printf("\nIndex:\n")

	scope, err := resolveRuntimeScope("")
	if err != nil {
		checkWarn("Could not resolve project scope: %v", err)
		s.warnings++
		return
	}

	pm, err := storage.NewProjectManager(projectBaseDir)
	if err != nil {
		checkWarn("Could not initialise project manager: %v", err)
		s.warnings++
		return
	}

	store, err := openStorageForScope(pm, scope, false)
	if err != nil {
		checkWarn("No index found for current project: %v", err)
		checkHint("Run 'vectos index .' to create an index")
		s.warnings++
		return
	}
	defer store.Close()

	stats, err := store.Stats()
	if err != nil {
		checkWarn("Could not read index stats: %v", err)
		s.warnings++
		return
	}

	if stats.FileCount > 0 {
		checkPass("Files indexed: %d", stats.FileCount)
		s.passes++
	} else {
		checkWarn("No files indexed")
		checkHint("Run 'vectos index .' to index files")
		s.warnings++
	}

	if stats.ChunkCount > 0 {
		checkPass("Chunks: %d", stats.ChunkCount)
		s.passes++
	} else {
		checkWarn("No chunks in index")
		s.warnings++
	}

	if stats.EmbeddedCount > 0 {
		checkPass("Chunks with embeddings: %d", stats.EmbeddedCount)
		s.passes++
	} else if stats.ChunkCount > 0 {
		checkWarn("No chunks have embeddings yet")
		checkHint("Run 'vectos index .' to generate embeddings")
		s.warnings++
	}

	if _, providerInfo, err := embeddings.ResolveEmbedder(embedConfig); err == nil {
		needsReindex, err := store.RequiresReindex(providerInfo.Provider, providerInfo.Model, providerInfo.Dimensions, currentIndexFingerprint(indexChunkerConfig(embedConfig.Embedded.BatchSize)))
		if err == nil && needsReindex {
			checkFail("Index/provider mismatch: reindex required")
			checkHint("Run 'vectos index .' to rebuild the index with the current provider")
			s.fails++
		} else if err == nil {
			checkPass("Index metadata matches embedding provider")
			s.passes++
		}
	}
}

// ---------------------------------------------------------------------------
// Output helpers
// ---------------------------------------------------------------------------

func printDoctorFooter(s *doctorSummary) {
	fmt.Println()
	fmt.Println("Result:")
	if s.fails > 0 {
		fmt.Printf("  %d check(s) failed (exit code 1)\n", s.fails)
	} else {
		fmt.Println("  All checks passed")
	}
	fmt.Printf("  %d passed, %d warning(s), %d failure(s)\n", s.passes, s.warnings, s.fails)
}

func checkPass(format string, args ...interface{}) {
	fmt.Printf("  [✔] %s\n", fmt.Sprintf(format, args...))
}

func checkWarn(format string, args ...interface{}) {
	fmt.Printf("  [!] %s\n", fmt.Sprintf(format, args...))
}

func checkFail(format string, args ...interface{}) {
	fmt.Printf("  [✘] %s\n", fmt.Sprintf(format, args...))
}

func checkHint(format string, args ...interface{}) {
	fmt.Printf("       Hint: %s\n", fmt.Sprintf(format, args...))
}
