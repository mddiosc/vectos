package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"vectos/internal/config"
	"vectos/internal/embeddings"
	"vectos/internal/storage"
)

type searchRun struct {
	Results []storage.CodeChunk
	Mode    string
	Warning string
}

type retrievalBenchmarkFile struct {
	Name    string                    `json:"name,omitempty"`
	Queries []retrievalBenchmarkQuery `json:"queries"`
}

type retrievalBenchmarkQuery struct {
	Name           string                  `json:"name"`
	Query          string                  `json:"query"`
	ExpectedFiles  []string                `json:"expected_files,omitempty"`
	ExpectedChunks []retrievalExpectedChunk `json:"expected_chunks,omitempty"`
}

type retrievalExpectedChunk struct {
	File      string `json:"file"`
	StartLine int    `json:"start_line"`
	EndLine   int    `json:"end_line"`
}

type benchmarkQueryResult struct {
	QueryName string
	Query     string
	Mode      string
	Warning   string
	TopHits   map[int]bool
	Results   []storage.CodeChunk
}

func executeSearch(store *storage.SQLiteStorage, embedConfig config.EmbeddingConfig, query string, limit int) (searchRun, error) {
	run := searchRun{Mode: "text"}

	embedClient, providerInfo, err := embeddings.ResolveEmbedder(embedConfig)
	if err != nil {
		results, textErr := store.SearchText(query)
		if textErr != nil {
			return searchRun{}, textErr
		}
		run.Results = limitResults(results, limit)
		return run, nil
	}

	requiresReindex, err := store.RequiresReindex(providerInfo.Provider, providerInfo.Model, providerInfo.Dimensions)
	if err == nil && requiresReindex {
		results, textErr := store.SearchText(query)
		if textErr != nil {
			return searchRun{}, textErr
		}
		run.Warning = "index metadata does not match current embedding provider; semantic results may be stale until reindex"
		run.Mode = "text_stale_index"
		run.Results = limitResults(results, limit)
		return run, nil
	}

	queryVector, err := embedClient.GetEmbedding(query)
	if err == nil {
		results, semanticErr := store.SearchSemantic(queryVector, limit)
		if semanticErr != nil {
			return searchRun{}, semanticErr
		}
		if len(results) > 0 {
			run.Mode = "semantic"
			run.Results = results
			return run, nil
		}
	}

	results, textErr := store.SearchText(query)
	if textErr != nil {
		return searchRun{}, textErr
	}
	run.Results = limitResults(results, limit)
	return run, nil
}

func runBenchmark(projectBaseDir string, embedConfig config.EmbeddingConfig, fixturePath string, projectName string) {
	fixture, absFixturePath, err := loadBenchmarkFile(fixturePath)
	if err != nil {
		logFatalf("error loading benchmark file: %v", err)
	}

	scope, err := resolveRuntimeScope(projectName)
	if err != nil {
		logFatalf("error resolving project scope: %v", err)
	}

	pm, err := storage.NewProjectManager(projectBaseDir)
	if err != nil {
		logFatalf("error initializing project manager: %v", err)
	}

	store, err := openStorageForScope(pm, scope)
	if err != nil {
		logFatalf("error opening database: %v", err)
	}
	defer store.Close()

	stats, err := store.Stats()
	if err != nil {
		logFatalf("error reading project index stats: %v", err)
	}
	if stats.ChunkCount == 0 {
		logFatalf("project has no indexed chunks; run 'vectos index .' before benchmarking")
	}

	projectLabel := "current project"
	if scope != nil && strings.TrimSpace(scope.Name) != "" {
		projectLabel = scope.Name
	}

	fmt.Printf("Benchmark: %s\n", benchmarkName(fixture, absFixturePath))
	fmt.Printf("Fixture: %s\n", absFixturePath)
	fmt.Printf("Project: %s\n", projectLabel)
	fmt.Printf("Indexed files: %d\n", stats.FileCount)
	fmt.Printf("Indexed chunks: %d\n", stats.ChunkCount)
	fmt.Println()

	windows := []int{3, 5}
	results := make([]benchmarkQueryResult, 0, len(fixture.Queries))
	hitCounts := map[int]int{3: 0, 5: 0}

	for i, query := range fixture.Queries {
		searchResult, err := executeSearch(store, embedConfig, query.Query, 5)
		if err != nil {
			logFatalf("error executing benchmark query %q: %v", query.Name, err)
		}

		topHits := evaluateTopHits(query, searchResult.Results, windows)
		for _, window := range windows {
			if topHits[window] {
				hitCounts[window]++
			}
		}

		result := benchmarkQueryResult{
			QueryName: query.Name,
			Query:     query.Query,
			Mode:      searchResult.Mode,
			Warning:   searchResult.Warning,
			TopHits:   topHits,
			Results:   searchResult.Results,
		}
		results = append(results, result)

		fmt.Printf("[%d/%d] %s\n", i+1, len(fixture.Queries), query.Name)
		fmt.Printf("Query: %s\n", query.Query)
		fmt.Printf("Expected: %s\n", formatExpectedTargets(query))
		fmt.Printf("Mode: %s\n", searchResult.Mode)
		if searchResult.Warning != "" {
			fmt.Printf("Warning: %s\n", searchResult.Warning)
		}
		fmt.Printf("Top 3: %s\n", hitLabel(topHits[3]))
		fmt.Printf("Top 5: %s\n", hitLabel(topHits[5]))
		fmt.Println("Results:")
		if len(searchResult.Results) == 0 {
			fmt.Println("  none")
		} else {
			for idx, chunk := range searchResult.Results {
				fmt.Printf("  %d. %s:%d-%d", idx+1, chunk.FilePath, chunk.StartLine, chunk.EndLine)
				if chunk.Score != 0 {
					fmt.Printf(" score=%.4f", chunk.Score)
				}
				fmt.Println()
			}
		}
		fmt.Println()
	}

	totalQueries := len(results)
	fmt.Println("Aggregate results")
	for _, window := range windows {
		fmt.Printf("Top %d hit rate: %d/%d (%.1f%%)\n", window, hitCounts[window], totalQueries, percentage(hitCounts[window], totalQueries))
	}
}

func loadBenchmarkFile(path string) (retrievalBenchmarkFile, string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return retrievalBenchmarkFile{}, "", err
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		return retrievalBenchmarkFile{}, "", err
	}

	var fixture retrievalBenchmarkFile
	if err := json.Unmarshal(data, &fixture); err != nil {
		return retrievalBenchmarkFile{}, absPath, fmt.Errorf("invalid benchmark JSON: %w", err)
	}

	if err := validateBenchmarkFile(fixture); err != nil {
		return retrievalBenchmarkFile{}, absPath, err
	}

	return fixture, absPath, nil
}

func validateBenchmarkFile(fixture retrievalBenchmarkFile) error {
	if len(fixture.Queries) == 0 {
		return fmt.Errorf("benchmark file must define at least one query")
	}

	for i, query := range fixture.Queries {
		label := fmt.Sprintf("query %d", i+1)
		if strings.TrimSpace(query.Name) != "" {
			label = fmt.Sprintf("query %q", query.Name)
		}
		if strings.TrimSpace(query.Name) == "" {
			return fmt.Errorf("%s must include a name", label)
		}
		if strings.TrimSpace(query.Query) == "" {
			return fmt.Errorf("%s must include a query", label)
		}
		if len(query.ExpectedFiles) == 0 && len(query.ExpectedChunks) == 0 {
			return fmt.Errorf("%s must include at least one expected file or expected chunk", label)
		}
		for _, file := range query.ExpectedFiles {
			if strings.TrimSpace(file) == "" {
				return fmt.Errorf("%s contains an empty expected file entry", label)
			}
		}
		for _, chunk := range query.ExpectedChunks {
			if strings.TrimSpace(chunk.File) == "" {
				return fmt.Errorf("%s contains an expected chunk without a file", label)
			}
			if chunk.StartLine <= 0 || chunk.EndLine <= 0 || chunk.EndLine < chunk.StartLine {
				return fmt.Errorf("%s contains an invalid expected chunk line range for %s", label, chunk.File)
			}
		}
	}

	return nil
}

func evaluateTopHits(query retrievalBenchmarkQuery, results []storage.CodeChunk, windows []int) map[int]bool {
	hits := make(map[int]bool, len(windows))
	for _, window := range windows {
		limit := window
		if len(results) < limit {
			limit = len(results)
		}
		for _, result := range results[:limit] {
			if queryMatchesResult(query, result) {
				hits[window] = true
				break
			}
		}
	}
	return hits
}

func queryMatchesResult(query retrievalBenchmarkQuery, result storage.CodeChunk) bool {
	resultPath := normalizePathForMatch(result.FilePath)
	for _, expectedFile := range query.ExpectedFiles {
		if pathMatches(resultPath, expectedFile) {
			return true
		}
	}
	for _, expectedChunk := range query.ExpectedChunks {
		if !pathMatches(resultPath, expectedChunk.File) {
			continue
		}
		if result.StartLine <= expectedChunk.EndLine && result.EndLine >= expectedChunk.StartLine {
			return true
		}
	}
	return false
}

func pathMatches(actualPath string, expectedPath string) bool {
	expected := normalizePathForMatch(expectedPath)
	return actualPath == expected || strings.HasSuffix(actualPath, "/"+expected)
}

func normalizePathForMatch(path string) string {
	return strings.TrimPrefix(filepath.ToSlash(filepath.Clean(path)), "./")
}

func benchmarkName(fixture retrievalBenchmarkFile, absFixturePath string) string {
	if strings.TrimSpace(fixture.Name) != "" {
		return fixture.Name
	}
	return filepath.Base(absFixturePath)
}

func formatExpectedTargets(query retrievalBenchmarkQuery) string {
	targets := make([]string, 0, len(query.ExpectedFiles)+len(query.ExpectedChunks))
	for _, file := range query.ExpectedFiles {
		targets = append(targets, file)
	}
	for _, chunk := range query.ExpectedChunks {
		targets = append(targets, fmt.Sprintf("%s:%d-%d", chunk.File, chunk.StartLine, chunk.EndLine))
	}
	return strings.Join(targets, ", ")
}

func hitLabel(hit bool) string {
	if hit {
		return "HIT"
	}
	return "MISS"
}

func percentage(numerator, denominator int) float64 {
	if denominator == 0 {
		return 0
	}
	return (float64(numerator) / float64(denominator)) * 100
}

func limitResults(results []storage.CodeChunk, limit int) []storage.CodeChunk {
	if limit <= 0 || len(results) <= limit {
		return results
	}
	return results[:limit]
}

func logFatalf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
