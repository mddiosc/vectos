package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"vectos/internal/storage"
	"vectos/internal/workspace"
)

func TestRecordSearchStatsAndBuildSummary(t *testing.T) {
	baseDir := t.TempDir()
	projectRoot := t.TempDir()
	pm, err := storage.NewProjectManager(baseDir)
	if err != nil {
		t.Fatalf("new project manager: %v", err)
	}

	fileA := filepath.Join(projectRoot, "alpha.go")
	fileB := filepath.Join(projectRoot, "beta.go")
	if err := os.WriteFile(fileA, []byte(strings.Repeat("a", 400)), 0644); err != nil {
		t.Fatalf("write alpha: %v", err)
	}
	if err := os.WriteFile(fileB, []byte(strings.Repeat("b", 300)), 0644); err != nil {
		t.Fatalf("write beta: %v", err)
	}

	scope := &workspace.Scope{Name: "vectos-test", PrimaryRoot: projectRoot}
	run := searchRun{Mode: "semantic_hybrid", Results: []storage.CodeChunk{
		{FilePath: fileA, Content: strings.Repeat("x", 100)},
		{FilePath: fileA, Content: strings.Repeat("y", 50)},
		{FilePath: fileB, Content: strings.Repeat("z", 75)},
	}}

	if err := recordSearchStats(pm, scope, searchCallCLICode, "query", run); err != nil {
		t.Fatalf("recordSearchStats: %v", err)
	}

	summary, err := buildSearchGainSummary(pm, scope)
	if err != nil {
		t.Fatalf("buildSearchGainSummary: %v", err)
	}

	if summary.AllTime.Calls != 1 {
		t.Fatalf("expected 1 call, got %d", summary.AllTime.Calls)
	}
	if summary.AllTime.SnippetChars != 225 {
		t.Fatalf("expected 225 snippet chars, got %d", summary.AllTime.SnippetChars)
	}
	if summary.AllTime.FileChars != 700 {
		t.Fatalf("expected 700 file chars, got %d", summary.AllTime.FileChars)
	}
	if summary.AllTime.SavedChars() != 475 {
		t.Fatalf("expected 475 saved chars, got %d", summary.AllTime.SavedChars())
	}
	if summary.ByCall[searchCallCLICode].Calls != 1 {
		t.Fatalf("expected breakdown for %s", searchCallCLICode)
	}
	statsBytes, err := os.ReadFile(summary.StatsPath)
	if err != nil {
		t.Fatalf("read stats file: %v", err)
	}
	if strings.Contains(string(statsBytes), "query") {
		t.Fatalf("expected persisted stats to omit query, got %q", string(statsBytes))
	}
}

func TestBuildSearchGainSummaryBucketsByTime(t *testing.T) {
	baseDir := t.TempDir()
	pm, err := storage.NewProjectManager(baseDir)
	if err != nil {
		t.Fatalf("new project manager: %v", err)
	}
	scope := &workspace.Scope{Name: "vectos-test"}
	projectDir, err := pm.EnsureProjectDirForName(scope.Name)
	if err != nil {
		t.Fatalf("ensure project dir: %v", err)
	}
	statsPath := filepath.Join(projectDir, searchStatsFileName)
	now := time.Now().UTC()
	records := []string{
		`{"ts":"` + now.Format(time.RFC3339) + `","call":"cli_search_code","results":1,"snippet_chars":100,"file_chars":500,"project":"vectos-test"}`,
		`{"ts":"` + now.AddDate(0, 0, -3).Format(time.RFC3339) + `","call":"mcp_search_code","results":1,"snippet_chars":50,"file_chars":250,"project":"vectos-test"}`,
		`{"ts":"` + now.AddDate(0, 0, -10).Format(time.RFC3339) + `","call":"mcp_search_docs","results":1,"snippet_chars":40,"file_chars":200,"project":"vectos-test"}`,
	}
	if err := os.WriteFile(statsPath, []byte(strings.Join(records, "\n")+"\n"), 0644); err != nil {
		t.Fatalf("write stats file: %v", err)
	}

	summary, err := buildSearchGainSummary(pm, scope)
	if err != nil {
		t.Fatalf("buildSearchGainSummary: %v", err)
	}
	if summary.Today.Calls != 1 {
		t.Fatalf("expected 1 today call, got %d", summary.Today.Calls)
	}
	if summary.Last7Days.Calls != 2 {
		t.Fatalf("expected 2 last7 calls, got %d", summary.Last7Days.Calls)
	}
	if summary.AllTime.Calls != 3 {
		t.Fatalf("expected 3 all-time calls, got %d", summary.AllTime.Calls)
	}
}

func TestFormatSearchGainReportVerbose(t *testing.T) {
	summary := searchGainSummary{
		Today:       searchGainBucket{Label: "Today", Calls: 1, SnippetChars: 100, FileChars: 500},
		Last7Days:   searchGainBucket{Label: "Last 7 days", Calls: 2, SnippetChars: 150, FileChars: 800},
		AllTime:     searchGainBucket{Label: "All time", Calls: 3, SnippetChars: 200, FileChars: 1200},
		ProjectName: "vectos",
		ByCall: map[string]searchGainBucket{
			searchCallCLICode: {Label: searchCallCLICode, Calls: 2, SnippetChars: 150, FileChars: 900},
		},
	}

	report := formatSearchGainReport(summary, true)
	if !strings.Contains(report, "Vectos gain") {
		t.Fatalf("expected report header, got %q", report)
	}
	if !strings.Contains(report, "Returned bytes") || !strings.Contains(report, "Saved bytes") {
		t.Fatalf("expected byte-based labels in report, got %q", report)
	}
	if !strings.Contains(report, "Usage breakdown") {
		t.Fatalf("expected verbose breakdown, got %q", report)
	}
	if !strings.Contains(report, searchCallCLICode) {
		t.Fatalf("expected call type in breakdown, got %q", report)
	}
}

func TestMeasureFileCharsResolvesRelativePathsAndDedupes(t *testing.T) {
	root := t.TempDir()
	rel := filepath.Join("src", "alpha.go")
	abs := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(abs), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(abs, []byte(strings.Repeat("a", 250)), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	got := measureFileChars([]storage.CodeChunk{
		{FilePath: rel, Content: "one"},
		{FilePath: rel, Content: "two"},
	}, root)
	if got != 250 {
		t.Fatalf("expected deduped 250 chars, got %d", got)
	}
}

func TestBuildSearchGainSummarySkipsMalformedLines(t *testing.T) {
	baseDir := t.TempDir()
	pm, err := storage.NewProjectManager(baseDir)
	if err != nil {
		t.Fatalf("new project manager: %v", err)
	}
	scope := &workspace.Scope{Name: "vectos-test"}
	projectDir, err := pm.EnsureProjectDirForName(scope.Name)
	if err != nil {
		t.Fatalf("ensure project dir: %v", err)
	}
	statsPath := filepath.Join(projectDir, searchStatsFileName)
	now := time.Now().UTC().Format(time.RFC3339)
	content := strings.Join([]string{
		`{"v":1,"ts":"` + now + `","call":"` + searchCallCLICode + `","results":1,"snippet_chars":100,"file_chars":500,"project":"vectos-test"}`,
		`{not-json}`,
		`{"v":1,"ts":"` + now + `","call":"` + searchCallMCPCode + `","results":0,"snippet_chars":0,"file_chars":0,"project":"vectos-test"}`,
	}, "\n") + "\n"
	if err := os.WriteFile(statsPath, []byte(content), 0644); err != nil {
		t.Fatalf("write stats file: %v", err)
	}

	summary, err := buildSearchGainSummary(pm, scope)
	if err != nil {
		t.Fatalf("buildSearchGainSummary: %v", err)
	}
	if summary.AllTime.Calls != 2 {
		t.Fatalf("expected 2 valid calls, got %d", summary.AllTime.Calls)
	}
}

func TestRecordSearchStatsAllowsZeroResults(t *testing.T) {
	baseDir := t.TempDir()
	pm, err := storage.NewProjectManager(baseDir)
	if err != nil {
		t.Fatalf("new project manager: %v", err)
	}
	scope := &workspace.Scope{Name: "vectos-test", PrimaryRoot: t.TempDir()}
	if err := recordSearchStats(pm, scope, searchCallMCPDocs, "ignored", searchRun{Mode: "text", Results: nil}); err != nil {
		t.Fatalf("recordSearchStats: %v", err)
	}
	summary, err := buildSearchGainSummary(pm, scope)
	if err != nil {
		t.Fatalf("buildSearchGainSummary: %v", err)
	}
	if summary.AllTime.Calls != 1 {
		t.Fatalf("expected zero-result call to be recorded, got %d", summary.AllTime.Calls)
	}
	if summary.ByCall[searchCallMCPDocs].Calls != 1 {
		t.Fatalf("expected MCP docs breakdown call to be recorded")
	}
}
