package main

import (
	"fmt"
	"log"
	"os"

	"vectos/internal/config"
	setupinternal "vectos/internal/setup"
)

func main() {
	home, _ := os.UserHomeDir()
	projectBaseDir := fmt.Sprintf("%s/.vectos/projects", home)
	embedConfig, err := config.LoadEmbeddingConfig(home)
	if err != nil {
		log.Fatalf("error loading embedding config: %v", err)
	}

	runCLI(appContext{
		projectBaseDir: projectBaseDir,
		embedConfig:    embedConfig,
		flags:          newCLIFlags(),
	}, os.Args[1:])
}

func runSetup(agent string, opts setupinternal.Options) {
	action := "setting up"
	if opts.Uninstall {
		action = "removing"
	}

	if err := setupinternal.Run(agent, opts); err != nil {
		log.Fatalf("error %s %s: %v", action, agent, err)
	}
	if opts.Uninstall {
		fmt.Printf("Vectos setup removed for %s.\n", agent)
		return
	}

	fmt.Printf("Vectos configured for %s.\n", agent)
}
