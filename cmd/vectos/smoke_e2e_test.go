package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"vectos/internal/storage"
)

func TestCLIRealWorldSmokeFlow(t *testing.T) {
	repoRoot := repoRootFromTestWD(t)
	binaryPath := buildVectosBinary(t, repoRoot)
	homeDir := t.TempDir()
	projectDir := filepath.Join(t.TempDir(), "smoke-project")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(projectDir): %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/embeddings" {
			http.NotFound(w, r)
			return
		}
		var req struct {
			Input []string `json:"input"`
			Model string   `json:"model"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		resp := struct {
			Data []struct {
				Embedding []float32 `json:"embedding"`
			} `json:"data"`
		}{
			Data: make([]struct {
				Embedding []float32 `json:"embedding"`
			}, len(req.Input)),
		}
		for i, input := range req.Input {
			resp.Data[i].Embedding = smokeVector(input)
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}))
	defer server.Close()

	writeSmokeConfig(t, homeDir, server.URL)

	filePath := filepath.Join(projectDir, "app.go")
	writeSmokeFile(t, filePath, "package smoke\n\nfunc checkout() string {\n\treturn \"checkout-token\"\n}\n")

	indexOutput := runVectosCommand(t, binaryPath, repoRoot, homeDir, "index", projectDir)
	assertContainsAll(t, indexOutput,
		"Indexing:",
		"Project:",
		"Found 1 supported files",
		"Done: 1 files",
	)

	searchOutput := runVectosCommand(t, binaryPath, projectDir, homeDir, "search", "checkout-token")
	assertContainsAll(t, searchOutput,
		"Searching: \"checkout-token\"",
		"Search mode: semantic_hybrid",
		"app.go",
		"checkout-token",
	)

	writeSmokeFile(t, filePath, "package smoke\n\nfunc refund() string {\n\treturn \"refund-token\"\n}\n")

	updateOutput := runVectosCommand(t, binaryPath, repoRoot, homeDir, "index", "--changed", "app.go", projectDir)
	assertContainsAll(t, updateOutput,
		"Found 1 changed supported files",
		"Done: 1 files",
	)

	refundOutput := runVectosCommand(t, binaryPath, projectDir, homeDir, "search", "refund-token")
	assertContainsAll(t, refundOutput,
		"Searching: \"refund-token\"",
		"Search mode: semantic_hybrid",
		"app.go",
		"refund-token",
	)

	statusOutput := runVectosCommand(t, binaryPath, projectDir, homeDir, "status")
	assertContainsAll(t, statusOutput,
		"Vectos status",
		"Embedding provider: remote",
		"Indexed files: 1",
	)

	pm, err := storage.NewProjectManager(filepath.Join(homeDir, ".vectos", "projects"))
	if err != nil {
		t.Fatalf("NewProjectManager: %v", err)
	}
	store, err := storage.NewSQLiteStorageForProjectName(pm, filepath.Base(projectDir))
	if err != nil {
		t.Fatalf("NewSQLiteStorageForProjectName: %v", err)
	}
	defer store.Close()

	oldResults, err := store.SearchText("checkout-token")
	if err != nil {
		t.Fatalf("SearchText checkout-token: %v", err)
	}
	if len(oldResults) != 0 {
		t.Fatalf("expected no stale checkout-token rows after update, got %d", len(oldResults))
	}

	newResults, err := store.SearchText("refund-token")
	if err != nil {
		t.Fatalf("SearchText refund-token: %v", err)
	}
	if len(newResults) == 0 {
		t.Fatal("expected updated refund-token content to be indexed")
	}

	stats, err := store.Stats()
	if err != nil {
		t.Fatalf("Stats: %v", err)
	}
	db, err := sql.Open("sqlite3", stats.DatabasePath)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	defer db.Close()

	var journalMode string
	if err := db.QueryRow("PRAGMA journal_mode;").Scan(&journalMode); err != nil {
		t.Fatalf("PRAGMA journal_mode: %v", err)
	}
	if journalMode != "wal" {
		t.Fatalf("expected WAL journal mode, got %q", journalMode)
	}

	if _, _, _, _, err := store.LoadVectorIndex(); err != nil {
		t.Fatalf("LoadVectorIndex: %v", err)
	}
}

func repoRootFromTestWD(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	return filepath.Clean(filepath.Join(wd, "..", ".."))
}

func buildVectosBinary(t *testing.T, repoRoot string) string {
	t.Helper()
	binaryPath := filepath.Join(t.TempDir(), "vectos-smoke")
	cmd := exec.Command("go", "build", "-o", binaryPath, "./cmd/vectos")
	cmd.Dir = repoRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go build ./cmd/vectos: %v\n%s", err, output)
	}
	return binaryPath
}

func writeSmokeConfig(t *testing.T, homeDir, baseURL string) {
	t.Helper()
	configDir := filepath.Join(homeDir, ".vectos")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(configDir): %v", err)
	}
	configPath := filepath.Join(configDir, "config.json")
	content := fmt.Sprintf(`{
  "embeddings": {
    "default_provider": "remote",
    "fallback_order": ["remote"],
    "embedded": {
      "enabled": false,
      "auto_download": false
    },
    "remote": {
      "enabled": true,
      "base_url": %q,
      "model": "smoke-test-model",
      "timeout_seconds": 5
    }
  }
}
`, baseURL)
	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(config.json): %v", err)
	}
}

func writeSmokeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%s): %v", path, err)
	}
}

func runVectosCommand(t *testing.T, binaryPath, workDir, homeDir string, args ...string) string {
	t.Helper()
	cmd := exec.Command(binaryPath, args...)
	cmd.Dir = workDir
	cmd.Env = append(os.Environ(), "HOME="+homeDir)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %s: %v\n%s", binaryPath, strings.Join(args, " "), err, output)
	}
	return string(output)
}

func assertContainsAll(t *testing.T, output string, parts ...string) {
	t.Helper()
	for _, part := range parts {
		if !strings.Contains(output, part) {
			t.Fatalf("expected output to contain %q\nfull output:\n%s", part, output)
		}
	}
}

func smokeVector(input string) []float32 {
	vector := make([]float32, 8)
	switch {
	case strings.Contains(strings.ToLower(input), "checkout-token"):
		vector[0] = 1
	case strings.Contains(strings.ToLower(input), "refund-token"):
		vector[1] = 1
	default:
		vector[2] = 1
	}
	return vector
}
