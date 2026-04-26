package main

import (
	"fmt"
	"os"

	"vectos/internal/buildinfo"
	"vectos/internal/config"
)

type appContext struct {
	projectBaseDir string
	embedConfig    config.EmbeddingConfig
	flags          cliFlags
}

func runCLI(app appContext, args []string) {
	if len(args) >= 1 && isHelpFlag(args[0]) {
		printHelp()
		os.Exit(0)
	}

	if len(args) == 0 {
		printHelp()
		os.Exit(0)
	}

	command := args[0]
	commandArgs := args[1:]

	switch command {
	case "help":
		runHelp(commandArgs)
	case "index":
		runIndexCommand(app, commandArgs)
	case "search":
		runSearchCommand(app, commandArgs)
	case "benchmark":
		runBenchmarkCommand(app, commandArgs)
	case "status":
		runStatusCommand(app, commandArgs)
	case "mcp":
		runMCPCommand(app, commandArgs)
	case "setup":
		runSetupCommand(app, commandArgs)
	case "version":
		runVersionCommand(commandArgs)
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", command)
		fmt.Fprintln(os.Stderr, "Run 'vectos help' for a list of available commands.")
		os.Exit(1)
	}
}

func runHelp(args []string) {
	if len(args) >= 1 {
		printSubcommandHelp(args[0])
		return
	}
	printHelp()
}

func runIndexCommand(app appContext, args []string) {
	indexArgs, showHelp := normalizeIndexArgs(args)
	if showHelp {
		printSubcommandHelp("index")
		os.Exit(0)
	}
	if err := app.flags.indexCmd.Parse(indexArgs); err != nil {
		fatalErr(err)
	}
	if app.flags.indexCmd.NArg() < 1 {
		printSubcommandHelp("index")
		os.Exit(1)
	}
	runIndex(app.projectBaseDir, app.embedConfig, app.flags.indexCmd.Arg(0), *app.flags.indexProject, parseChangedPaths(*app.flags.indexChanged))
}

func runSearchCommand(app appContext, args []string) {
	if hasHelpFlag(args) {
		printSubcommandHelp("search")
		os.Exit(0)
	}
	if err := app.flags.searchCmd.Parse(args); err != nil {
		fatalErr(err)
	}
	if app.flags.searchCmd.NArg() < 1 {
		printSubcommandHelp("search")
		os.Exit(1)
	}
	runSearch(app.projectBaseDir, app.embedConfig, app.flags.searchCmd.Arg(0), *app.flags.searchProject)
}

func runBenchmarkCommand(app appContext, args []string) {
	if hasHelpFlag(args) {
		printSubcommandHelp("benchmark")
		os.Exit(0)
	}
	if err := app.flags.benchmarkCmd.Parse(args); err != nil {
		fatalErr(err)
	}
	if app.flags.benchmarkCmd.NArg() < 1 {
		printSubcommandHelp("benchmark")
		os.Exit(1)
	}
	runBenchmark(app.projectBaseDir, app.embedConfig, app.flags.benchmarkCmd.Arg(0), *app.flags.benchmarkProject)
}

func runStatusCommand(app appContext, args []string) {
	if hasHelpFlag(args) {
		printSubcommandHelp("status")
		os.Exit(0)
	}
	if err := app.flags.statusCmd.Parse(args); err != nil {
		fatalErr(err)
	}
	runStatus(app.projectBaseDir, *app.flags.statusProject)
}

func runMCPCommand(app appContext, args []string) {
	if hasHelpFlag(args) {
		printSubcommandHelp("mcp")
		os.Exit(0)
	}
	if err := app.flags.mcpCmd.Parse(args); err != nil {
		fatalErr(err)
	}
	runMCP(app.projectBaseDir, app.embedConfig)
}

func runSetupCommand(app appContext, args []string) {
	setupArgs, showHelp := normalizeSetupArgs(args)
	if showHelp {
		printSubcommandHelp("setup")
		os.Exit(0)
	}
	if err := app.flags.setupCmd.Parse(setupArgs); err != nil {
		fatalErr(err)
	}
	if app.flags.setupCmd.NArg() < 1 {
		printSubcommandHelp("setup")
		os.Exit(1)
	}
	runSetup(app.flags.setupCmd.Arg(0), *app.flags.setupUninstall)
}

func runVersionCommand(args []string) {
	if hasHelpFlag(args) {
		printSubcommandHelp("version")
		os.Exit(0)
	}
	fmt.Printf("vectos %s\n", buildinfo.Version)
	fmt.Printf("commit: %s\n", buildinfo.Commit)
	fmt.Printf("built:  %s\n", buildinfo.Date)
}

func isHelpFlag(arg string) bool {
	return arg == "--help" || arg == "-h"
}

func hasHelpFlag(args []string) bool {
	for _, arg := range args {
		if isHelpFlag(arg) {
			return true
		}
	}
	return false
}

func fatalErr(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
