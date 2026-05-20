package main

import (
	"flag"
	"os"
	"strconv"
	"time"
)

type cliFlags struct {
	indexCmd            *flag.FlagSet
	searchCmd           *flag.FlagSet
	gainCmd             *flag.FlagSet
	benchmarkCmd        *flag.FlagSet
	statusCmd           *flag.FlagSet
	mcpCmd              *flag.FlagSet
	setupCmd            *flag.FlagSet
	serveCmd            *flag.FlagSet
	doctorCmd           *flag.FlagSet
	indexProject        *string
	indexChanged        *string
	indexDocs           *bool
	indexDimensions     *int
	searchProject       *string
	searchFull          *bool
	searchDocs          *bool
	gainProject         *string
	gainVerbose         *bool
	benchmarkProject    *string
	statusProject       *string
	statusDocs          *bool
	setupUninstall      *bool
	setupYes            *bool
	setupNoGuidance     *bool
	servePort           *int
	serveProjectBaseDir *string
	watchEnabled        *bool
	watchDebounce       *time.Duration
	watchIgnore         *string
}

func newCLIFlags() cliFlags {
	indexCmd := flag.NewFlagSet("index", flag.ExitOnError)
	searchCmd := flag.NewFlagSet("search", flag.ExitOnError)
	gainCmd := flag.NewFlagSet("gain", flag.ExitOnError)
	benchmarkCmd := flag.NewFlagSet("benchmark", flag.ExitOnError)
	statusCmd := flag.NewFlagSet("status", flag.ExitOnError)
	mcpCmd := flag.NewFlagSet("mcp", flag.ExitOnError)
	setupCmd := flag.NewFlagSet("setup", flag.ExitOnError)
	serveCmd := flag.NewFlagSet("serve", flag.ExitOnError)
	doctorCmd := flag.NewFlagSet("doctor", flag.ExitOnError)

	servePortDefault := 7438
	if envPort := os.Getenv("VECTOS_PORT"); envPort != "" {
		if p, err := strconv.Atoi(envPort); err == nil && p > 0 && p <= 65535 {
			servePortDefault = p
		}
	}

	return cliFlags{
		indexCmd:            indexCmd,
		searchCmd:           searchCmd,
		gainCmd:             gainCmd,
		benchmarkCmd:        benchmarkCmd,
		statusCmd:           statusCmd,
		mcpCmd:              mcpCmd,
		setupCmd:            setupCmd,
		serveCmd:            serveCmd,
		doctorCmd:           doctorCmd,
		indexProject:        indexCmd.String("project", "", "Nx project name to index with internal workspace dependencies when inside an Nx workspace"),
		indexChanged:        indexCmd.String("changed", "", "Comma-separated changed file paths to refresh incrementally"),
		indexDocs:           indexCmd.Bool("docs", false, "Index only documentation files into a separate docs database"),
		indexDimensions:     indexCmd.Int("dimensions", 0, "Embedding dimensions (32, 64, 128, 256, 512, 768, 1024). Default: 512. Lower = faster + smaller, higher = better quality"),
		searchProject:       searchCmd.String("project", "", "Nx project name to search with internal workspace dependencies when inside an Nx workspace"),
		searchFull:          searchCmd.Bool("full", false, "Show full chunk content instead of compact snippets"),
		searchDocs:          searchCmd.Bool("docs", false, "Search documentation index instead of source code"),
		gainProject:         gainCmd.String("project", "", "Nx project name to inspect with internal workspace dependencies when inside an Nx workspace"),
		gainVerbose:         gainCmd.Bool("verbose", false, "Show usage breakdown by search call type"),
		benchmarkProject:    benchmarkCmd.String("project", "", "Nx project name to benchmark with internal workspace dependencies when inside an Nx workspace"),
		statusProject:       statusCmd.String("project", "", "Nx project name to inspect with internal workspace dependencies when inside an Nx workspace"),
		statusDocs:          statusCmd.Bool("docs", false, "Show documentation index status instead of source index"),
		setupUninstall:      setupCmd.Bool("uninstall", false, "Remove the Vectos MCP setup for the selected agent"),
		setupYes:            setupCmd.Bool("yes", false, "Answer yes to all prompts (non-interactive mode)"),
		setupNoGuidance:     setupCmd.Bool("no-guidance", false, "Skip global guidance updates"),
		servePort:           serveCmd.Int("port", servePortDefault, "Port to listen on (default 7438, overridable via VECTOS_PORT env var)"),
		serveProjectBaseDir: serveCmd.String("project-base-dir", "", "Directory for project index databases (default ~/.vectos/projects)"),
		watchEnabled:        serveCmd.Bool("watch", true, "enable automatic reindex on file changes"),
		watchDebounce:       serveCmd.Duration("watch-debounce", 500*time.Millisecond, "debounce delay for file change events"),
		watchIgnore:         serveCmd.String("watch-ignore", ".git,node_modules,*.lock", "comma-separated glob patterns to ignore"),
	}
}

// normalizePositionalArgs moves all leading positional (non-flag) arguments to
// after the flags block so that Go's flag.Parse can see flags that follow
// positionals.  Returns the re-ordered slice and whether the caller should
// print help and exit.
func normalizePositionalArgs(args []string) (reordered []string, showHelp bool) {
	if len(args) == 0 {
		return args, false
	}

	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			return nil, true
		}
	}

	if !isFlagArg(args[0]) {
		// Find where the contiguous leading positional args end.
		split := 1
		for split < len(args) && !isFlagArg(args[split]) {
			split++
		}
		// No flags at all — nothing to reorder.
		if split == len(args) {
			return args, false
		}
		// Move all leading positionals after the flags block.
		reordered := make([]string, 0, len(args))
		reordered = append(reordered, args[split:]...)
		reordered = append(reordered, args[:split]...)
		return reordered, false
	}

	return args, false
}

func normalizeIndexArgs(args []string) ([]string, bool) {
	return normalizePositionalArgs(args)
}

func normalizeSearchArgs(args []string) ([]string, bool) {
	return normalizePositionalArgs(args)
}

func normalizeBenchmarkArgs(args []string) ([]string, bool) {
	return normalizePositionalArgs(args)
}

func normalizeSetupArgs(args []string) ([]string, bool) {
	flags := make([]string, 0, len(args))
	positionals := make([]string, 0, len(args))
	showHelp := false

	for _, arg := range args {
		switch arg {
		case "--help", "-h":
			showHelp = true
		case "--uninstall":
			flags = append(flags, arg)
		case "--yes", "-y":
			flags = append(flags, "--yes")
		case "--no-guidance":
			flags = append(flags, arg)
		default:
			positionals = append(positionals, arg)
		}
	}

	return append(flags, positionals...), showHelp
}

func isFlagArg(arg string) bool {
	return len(arg) > 0 && arg[0] == '-'
}
