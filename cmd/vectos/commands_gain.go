package main

import (
	"fmt"
	"log"

	"vectos/internal/storage"
)

func runGain(projectBaseDir string, projectName string, verbose bool) {
	scope, err := resolveRuntimeScope(projectName)
	if err != nil {
		log.Fatalf("error resolving project scope: %v", err)
	}

	pm, err := storage.NewProjectManager(projectBaseDir)
	if err != nil {
		log.Fatalf("error initializing project manager: %v", err)
	}

	summary, err := buildSearchGainSummary(pm, scope)
	if err != nil {
		log.Fatalf("error building gain summary: %v", err)
	}

	fmt.Print(formatSearchGainReport(summary, verbose))
}
