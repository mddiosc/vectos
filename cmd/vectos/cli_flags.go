package main

import (
	"flag"
	"os"
	"strconv"
)

type cliFlags struct {
	indexCmd         *flag.FlagSet
	searchCmd        *flag.FlagSet
	benchmarkCmd     *flag.FlagSet
	statusCmd        *flag.FlagSet
	mcpCmd           *flag.FlagSet
	setupCmd         *flag.FlagSet
	serveCmd         *flag.FlagSet
	indexProject     *string
	indexChanged     *string
	indexDocs        *bool
	searchProject    *string
	searchFull       *bool
	searchDocs       *bool
	benchmarkProject *string
	statusProject    *string
	statusDocs       *bool
	setupUninstall   *bool
	servePort        *int
}

func newCLIFlags() cliFlags {
	indexCmd := flag.NewFlagSet("index", flag.ExitOnError)
	searchCmd := flag.NewFlagSet("search", flag.ExitOnError)
	benchmarkCmd := flag.NewFlagSet("benchmark", flag.ExitOnError)
	statusCmd := flag.NewFlagSet("status", flag.ExitOnError)
	mcpCmd := flag.NewFlagSet("mcp", flag.ExitOnError)
	setupCmd := flag.NewFlagSet("setup", flag.ExitOnError)
	serveCmd := flag.NewFlagSet("serve", flag.ExitOnError)

	servePortDefault := 7438
	if envPort := os.Getenv("VECTOS_PORT"); envPort != "" {
		if p, err := strconv.Atoi(envPort); err == nil && p > 0 && p <= 65535 {
			servePortDefault = p
		}
	}

	return cliFlags{
		indexCmd:         indexCmd,
		searchCmd:        searchCmd,
		benchmarkCmd:     benchmarkCmd,
		statusCmd:        statusCmd,
		mcpCmd:           mcpCmd,
		setupCmd:         setupCmd,
		serveCmd:         serveCmd,
		indexProject:     indexCmd.String("project", "", "Nx project name to index with internal workspace dependencies when inside an Nx workspace"),
		indexChanged:     indexCmd.String("changed", "", "Comma-separated changed file paths to refresh incrementally"),
		indexDocs:        indexCmd.Bool("docs", false, "Index only documentation files into a separate docs database"),
		searchProject:    searchCmd.String("project", "", "Nx project name to search with internal workspace dependencies when inside an Nx workspace"),
		searchFull:       searchCmd.Bool("full", false, "Show full chunk content instead of compact snippets"),
		searchDocs:       searchCmd.Bool("docs", false, "Search documentation index instead of source code"),
		benchmarkProject: benchmarkCmd.String("project", "", "Nx project name to benchmark with internal workspace dependencies when inside an Nx workspace"),
		statusProject:    statusCmd.String("project", "", "Nx project name to inspect with internal workspace dependencies when inside an Nx workspace"),
		statusDocs:       statusCmd.Bool("docs", false, "Show documentation index status instead of source index"),
		setupUninstall:   setupCmd.Bool("uninstall", false, "Remove the Vectos MCP setup for the selected agent"),
		servePort:        serveCmd.Int("port", servePortDefault, "Port to listen on (default 7438, overridable via VECTOS_PORT env var)"),
	}
}

func normalizeIndexArgs(args []string) ([]string, bool) {
	if len(args) == 0 {
		return args, false
	}

	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			return nil, true
		}
	}

	if len(args) > 0 && !isFlagArg(args[0]) {
		return append(args[1:], args[0]), false
	}

	return args, false
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
		default:
			positionals = append(positionals, arg)
		}
	}

	return append(flags, positionals...), showHelp
}

func isFlagArg(arg string) bool {
	return len(arg) > 0 && arg[0] == '-'
}
