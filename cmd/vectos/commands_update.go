package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"vectos/internal/buildinfo"
)

const (
	updateRepo       = "mddiosc/vectos"
	updateInstallURL = "https://github.com/mddiosc/vectos/releases/latest/download/install.sh"
)

type githubRelease struct {
	TagName string `json:"tag_name"`
	Name    string `json:"name"`
	Body    string `json:"body"`
	HTMLURL string `json:"html_url"`
}

func runUpdateCommand(_ appContext, args []string) {
	if hasHelpFlag(args) {
		printSubcommandHelp("update")
		os.Exit(0)
	}

	yes := false
	for _, a := range args {
		if a == "--yes" || a == "-y" {
			yes = true
		}
	}

	rel, err := fetchLatestRelease()
	if err != nil {
		fatalErr(fmt.Errorf("checking for updates: %w", err))
	}

	current := buildinfo.Version
	latest := rel.TagName

	fmt.Printf("Current version: %s\n", current)
	fmt.Printf("Latest version:  %s\n", latest)
	fmt.Println()

	if current == latest {
		fmt.Println("✓ You are already on the latest version.")
		return
	}
	// ponytail: string compare, not semver. Equal=up-to-date is all we need;
	// upgrade to a semver compare if downgrade detection ever matters.

	if notes := strings.TrimSpace(rel.Body); notes != "" {
		fmt.Printf("Release notes for %s:\n\n%s\n\n", latest, notes)
	}
	fmt.Printf("Full changelog: %s\n\n", rel.HTMLURL)

	if !yes && !confirm(fmt.Sprintf("Update from %s to %s now?", current, latest)) {
		fmt.Println("Update cancelled.")
		return
	}

	if err := runInstaller(); err != nil {
		fatalErr(fmt.Errorf("update failed: %w", err))
	}
	fmt.Printf("\n✓ Updated to %s. Run 'vectos version' to verify.\n", latest)
}

func fetchLatestRelease() (*githubRelease, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", updateRepo)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned %s", resp.Status)
	}

	return decodeRelease(resp.Body)
}

func decodeRelease(r io.Reader) (*githubRelease, error) {
	var rel githubRelease
	if err := json.NewDecoder(r).Decode(&rel); err != nil {
		return nil, err
	}
	if rel.TagName == "" {
		return nil, fmt.Errorf("no release tag found")
	}
	return &rel, nil
}

// runInstaller delegates the actual download/checksum/replace to the published
// install.sh, which already handles OS/arch detection, checksum verification,
// and PATH setup. ponytail: reuse the shell installer instead of reimplementing
// asset download + checksum + atomic replace in Go.
func runInstaller() error {
	cmd := exec.Command("sh", "-c",
		fmt.Sprintf("curl -fsSL %s | sh", updateInstallURL))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

func confirm(prompt string) bool {
	fmt.Printf("%s [y/N]: ", prompt)
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return false
	}
	answer := strings.ToLower(strings.TrimSpace(line))
	return answer == "y" || answer == "yes"
}
