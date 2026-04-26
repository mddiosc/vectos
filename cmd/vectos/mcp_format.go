package main

import (
	"fmt"
	"path/filepath"
	"strings"

	"vectos/internal/storage"
	"vectos/internal/workspace"
)

type mcpSearchPayload struct {
	Mode       string                 `json:"mode,omitempty"`
	Warning    string                 `json:"warning,omitempty"`
	Project    string                 `json:"project,omitempty"`
	Guidance   string                 `json:"guidance,omitempty"`
	NextAction string                 `json:"next_action,omitempty"`
	Results    []mcpSearchResultEntry `json:"results,omitempty"`
}

type mcpSearchResultEntry struct {
	Rank      int     `json:"rank"`
	FilePath  string  `json:"file_path"`
	FileName  string  `json:"file_name"`
	StartLine int     `json:"start_line"`
	EndLine   int     `json:"end_line"`
	Language  string  `json:"language,omitempty"`
	Category  string  `json:"category,omitempty"`
	Score     float64 `json:"score,omitempty"`
	Preview   string  `json:"preview,omitempty"`
	Reason    string  `json:"reason,omitempty"`
}

type mcpIndexPayload struct {
	Project      string   `json:"project"`
	Mode         string   `json:"mode"`
	IndexedFiles int      `json:"indexed_files"`
	IndexedChunks int     `json:"indexed_chunks"`
	SkippedPaths int      `json:"skipped_paths"`
	Roots        []string `json:"roots,omitempty"`
	Summary      string   `json:"summary"`
}

func buildMCPSearchPayload(scope *workspace.Scope, searchRun searchRun) mcpSearchPayload {
	payload := mcpSearchPayload{
		Mode:    searchRun.Mode,
		Warning: searchRun.Warning,
		Project: scopeName(scope),
		Results: make([]mcpSearchResultEntry, 0, len(searchRun.Results)),
	}

	for i, result := range searchRun.Results {
		payload.Results = append(payload.Results, mcpSearchResultEntry{
			Rank:      i + 1,
			FilePath:  result.FilePath,
			FileName:  filepath.Base(result.FilePath),
			StartLine: result.StartLine,
			EndLine:   result.EndLine,
			Language:  result.Language,
			Category:  result.Category,
			Score:     result.Score,
			Preview:   compactPreview(result.Content),
			Reason:    explainResultReason(searchRun.Mode, result),
		})
	}

	if searchRun.Warning != "" {
		payload.Guidance = "Refresh the project index before trusting semantic ranking."
		payload.NextAction = suggestedRefreshAction(scope)
	}

	return payload
}

func buildMCPMissingIndexPayload(scope *workspace.Scope) mcpSearchPayload {
	return mcpSearchPayload{
		Project:    scopeName(scope),
		Guidance:   "This project does not have a usable Vectos index yet.",
		NextAction: suggestedIndexAction(scope),
	}
}

func buildMCPIndexPayload(scope workspace.Scope, changedPaths []string, indexedFiles int, indexedChunks int, skippedPaths int) mcpIndexPayload {
	mode := "full"
	if len(changedPaths) > 0 {
		mode = "incremental"
	}
	label := "files"
	if mode == "incremental" {
		label = "changed files"
	}

	return mcpIndexPayload{
		Project:       scope.Name,
		Mode:          mode,
		IndexedFiles:  indexedFiles,
		IndexedChunks: indexedChunks,
		SkippedPaths:  skippedPaths,
		Roots:         scope.Roots,
		Summary:       fmt.Sprintf("Successfully indexed %d %s and %d chunks for %s", indexedFiles, label, indexedChunks, scope.Name),
	}
}

func compactPreview(content string) string {
	trimmed := strings.TrimSpace(content)
	trimmed = strings.ReplaceAll(trimmed, "\n", " ")
	trimmed = strings.Join(strings.Fields(trimmed), " ")
	if len(trimmed) > 160 {
		return trimmed[:157] + "..."
	}
	return trimmed
}

func explainResultReason(mode string, result storage.CodeChunk) string {
	parts := make([]string, 0, 3)
	if strings.Contains(mode, "semantic") {
		parts = append(parts, "strong semantic match")
	}
	if result.Category == "source" {
		parts = append(parts, "actionable source code")
	}
	if base := filepath.Base(result.FilePath); strings.Contains(strings.ToLower(result.Content), strings.TrimSuffix(strings.ToLower(base), filepath.Ext(base))) {
		parts = append(parts, "file content aligns with file name")
	}
	if len(parts) == 0 {
		parts = append(parts, "relevant ranked match")
	}
	return strings.Join(parts, "; ")
}

func suggestedIndexAction(scope *workspace.Scope) string {
	if scope != nil && len(scope.Roots) > 0 {
		return fmt.Sprintf("Run index_project for %s or use `vectos index %s`.", scopeName(scope), scope.PrimaryRoot)
	}
	return "Run index_project for this project or use `vectos index .`."
}

func suggestedRefreshAction(scope *workspace.Scope) string {
	if scope != nil && len(scope.Roots) > 0 {
		return fmt.Sprintf("Refresh the index for %s with index_project or `vectos index %s`.", scopeName(scope), scope.PrimaryRoot)
	}
	return "Refresh the index with index_project or `vectos index .`."
}

func scopeName(scope *workspace.Scope) string {
	if scope == nil || strings.TrimSpace(scope.Name) == "" {
		return "current project"
	}
	return scope.Name
}
