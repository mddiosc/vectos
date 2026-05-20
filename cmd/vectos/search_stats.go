package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"vectos/internal/storage"
	"vectos/internal/workspace"
)

const searchStatsFileName = "search_stats.jsonl"
const currentSearchStatsVersion = 1

const (
	searchCallCLICode  = "cli_search_code"
	searchCallCLIDocs  = "cli_search_docs"
	searchCallMCPCode  = "mcp_search_code"
	searchCallMCPDocs  = "mcp_search_docs"
	approxCharsPerToken = 4
)

var searchStatsLocks sync.Map

type searchStatsRecord struct {
	Version      int       `json:"v"`
	Timestamp    time.Time `json:"ts"`
	Call         string    `json:"call"`
	Mode         string    `json:"mode,omitempty"`
	Results      int       `json:"results"`
	SnippetChars int64     `json:"snippet_chars"`
	FileChars    int64     `json:"file_chars"`
	Project      string    `json:"project,omitempty"`
}

type searchGainBucket struct {
	Label        string
	Calls        int
	SnippetChars int64
	FileChars    int64
}

func (b searchGainBucket) SavedChars() int64 {
	saved := b.FileChars - b.SnippetChars
	if saved < 0 {
		return 0
	}
	return saved
}

func (b searchGainBucket) SavedTokensApprox() int64 {
	return b.SavedChars() / approxCharsPerToken
}

func (b searchGainBucket) SavingPercent() float64 {
	if b.FileChars <= 0 {
		return 0
	}
	return float64(b.SavedChars()) * 100 / float64(b.FileChars)
}

type searchGainSummary struct {
	Today       searchGainBucket
	Last7Days   searchGainBucket
	AllTime     searchGainBucket
	ByCall      map[string]searchGainBucket
	StatsPath   string
	ProjectName string
}

func recordSearchStats(pm *storage.ProjectManager, scope *workspace.Scope, callType, _ string, run searchRun) error {
	if pm == nil || scope == nil || strings.TrimSpace(scope.Name) == "" {
		return nil
	}

	projectDir, err := pm.EnsureProjectDirForName(scope.Name)
	if err != nil {
		return err
	}

	record := searchStatsRecord{
		Version:      currentSearchStatsVersion,
		Timestamp:    time.Now().UTC(),
		Call:         callType,
		Mode:         run.Mode,
		Results:      len(run.Results),
		SnippetChars: measureSnippetChars(run.Results),
		FileChars:    measureFileChars(run.Results, scope.PrimaryRoot),
		Project:      scope.Name,
	}

	return appendSearchStatsRecord(filepath.Join(projectDir, searchStatsFileName), record)
}

func measureSnippetChars(results []storage.CodeChunk) int64 {
	var total int64
	for _, result := range results {
		total += int64(len(result.Content))
	}
	return total
}

func measureFileChars(results []storage.CodeChunk, primaryRoot string) int64 {
	seen := make(map[string]struct{}, len(results))
	var total int64
	for _, result := range results {
		resolved := result.FilePath
		if !filepath.IsAbs(resolved) && primaryRoot != "" {
			resolved = filepath.Join(primaryRoot, resolved)
		}
		if _, ok := seen[resolved]; ok || strings.TrimSpace(resolved) == "" {
			continue
		}
		seen[resolved] = struct{}{}
		info, err := os.Stat(resolved)
		if err != nil || info.IsDir() {
			continue
		}
		total += info.Size()
	}
	return total
}

func appendSearchStatsRecord(path string, record searchStatsRecord) error {
	line, err := json.Marshal(record)
	if err != nil {
		return err
	}

	mu := searchStatsLock(path)
	mu.Lock()
	defer mu.Unlock()

	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := f.Write(append(line, '\n')); err != nil {
		return err
	}
	return nil
}

func searchStatsLock(path string) *sync.Mutex {
	mu, _ := searchStatsLocks.LoadOrStore(path, &sync.Mutex{})
	return mu.(*sync.Mutex)
}

func buildSearchGainSummary(pm *storage.ProjectManager, scope *workspace.Scope) (searchGainSummary, error) {
	if pm == nil || scope == nil || strings.TrimSpace(scope.Name) == "" {
		return searchGainSummary{}, fmt.Errorf("project scope is required")
	}
	projectDir, err := pm.EnsureProjectDirForName(scope.Name)
	if err != nil {
		return searchGainSummary{}, err
	}
	statsPath := filepath.Join(projectDir, searchStatsFileName)
	summary := searchGainSummary{
		Today:       searchGainBucket{Label: "Today"},
		Last7Days:   searchGainBucket{Label: "Last 7 days"},
		AllTime:     searchGainBucket{Label: "All time"},
		ByCall:      make(map[string]searchGainBucket),
		StatsPath:   statsPath,
		ProjectName: scope.Name,
	}

	f, err := os.Open(statsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return summary, nil
		}
		return searchGainSummary{}, err
	}
	defer f.Close()

	now := time.Now().UTC()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	sevenDaysAgo := now.AddDate(0, 0, -7)

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		var record searchStatsRecord
		if err := json.Unmarshal(scanner.Bytes(), &record); err != nil {
			continue
		}
		accumulateGainBucket(&summary.AllTime, record)
		if record.Timestamp.After(sevenDaysAgo) {
			accumulateGainBucket(&summary.Last7Days, record)
		}
		if !record.Timestamp.Before(todayStart) {
			accumulateGainBucket(&summary.Today, record)
		}
		bucket := summary.ByCall[record.Call]
		bucket.Label = record.Call
		accumulateGainBucket(&bucket, record)
		summary.ByCall[record.Call] = bucket
	}
	if err := scanner.Err(); err != nil {
		return summary, nil
	}
	return summary, nil
}

func accumulateGainBucket(bucket *searchGainBucket, record searchStatsRecord) {
	bucket.Calls++
	bucket.SnippetChars += record.SnippetChars
	bucket.FileChars += record.FileChars
}

func formatSearchGainReport(summary searchGainSummary, verbose bool) string {
	var b strings.Builder
	b.WriteString("Vectos gain\n")
	b.WriteString(fmt.Sprintf("Project: %s\n\n", summary.ProjectName))
	writeGainBucket(&b, summary.Today)
	writeGainBucket(&b, summary.Last7Days)
	writeGainBucket(&b, summary.AllTime)

	if verbose && len(summary.ByCall) > 0 {
		b.WriteString("\nUsage breakdown\n")
		keys := make([]string, 0, len(summary.ByCall))
		for key := range summary.ByCall {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			bucket := summary.ByCall[key]
			b.WriteString(fmt.Sprintf("- %s: %d calls, %s saved (~%s tokens, %.1f%%)\n",
				key,
				bucket.Calls,
				formatCompactInt(bucket.SavedChars()),
				formatCompactInt(bucket.SavedTokensApprox()),
				bucket.SavingPercent(),
			))
		}
	}

	if summary.AllTime.Calls == 0 {
		b.WriteString("\nNo search stats recorded yet.\n")
	}

	return b.String()
}

func writeGainBucket(b *strings.Builder, bucket searchGainBucket) {
	b.WriteString(bucket.Label + "\n")
	b.WriteString(fmt.Sprintf("  Searches: %d\n", bucket.Calls))
	b.WriteString(fmt.Sprintf("  Returned chars: %s\n", formatCompactInt(bucket.SnippetChars)))
	b.WriteString(fmt.Sprintf("  Full-file chars: %s\n", formatCompactInt(bucket.FileChars)))
	b.WriteString(fmt.Sprintf("  Saved chars: %s\n", formatCompactInt(bucket.SavedChars())))
	b.WriteString(fmt.Sprintf("  Saved tokens (~): %s\n", formatCompactInt(bucket.SavedTokensApprox())))
	b.WriteString(fmt.Sprintf("  Saving: %.1f%%\n\n", bucket.SavingPercent()))
}

func formatCompactInt(value int64) string {
	abs := value
	if abs < 0 {
		abs = -abs
	}
	switch {
	case abs >= 1_000_000:
		return fmt.Sprintf("%.1fM", float64(value)/1_000_000)
	case abs >= 1_000:
		return fmt.Sprintf("%.1fk", float64(value)/1_000)
	default:
		return fmt.Sprintf("%d", value)
	}
}
