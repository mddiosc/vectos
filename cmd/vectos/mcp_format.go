package main

import (
	"fmt"
	"path/filepath"
	"strings"

	"vectos/internal/storage"
	"vectos/internal/workspace"
)

const (
	highConfidenceThreshold = 0.90
)

type mcpSearchPayload struct {
	Mode       string                   `json:"mode,omitempty"`
	Warning    string                   `json:"warning,omitempty"`
	Project    string                   `json:"project,omitempty"`
	Guidance   string                   `json:"guidance,omitempty"`
	NextAction string                   `json:"next_action,omitempty"`
	Results    []mcpSearchFileResult   `json:"results,omitempty"`
}

type mcpSearchFileResult struct {
	Rank      int          `json:"rank"`
	FilePath  string       `json:"file_path"`
	FileName  string       `json:"file_name"`
	Language  string      `json:"language,omitempty"`
	Category  string      `json:"category,omitempty"`
	Relevance float64     `json:"relevance,omitempty"`
	LineRanges []storage.LineRange `json:"line_ranges"`
	Signatures []string    `json:"signatures"`
	Hint      string       `json:"hint,omitempty"`
}

type mcpIndexPayload struct {
	Project       string   `json:"project"`
	Mode          string   `json:"mode"`
	IndexedFiles  int      `json:"indexed_files"`
	IndexedChunks int      `json:"indexed_chunks"`
	SkippedPaths  int      `json:"skipped_paths"`
	Roots         []string `json:"roots,omitempty"`
	Summary       string   `json:"summary"`
}

func buildMCPSearchPayload(scope *workspace.Scope, query string, searchRun searchRun) mcpSearchPayload {
	payload := mcpSearchPayload{
		Mode:    searchRun.Mode,
		Warning: searchRun.Warning,
		Project: scopeName(scope),
	}

	if len(searchRun.FileResults) > 0 {
		payload.Results = make([]mcpSearchFileResult, 0, len(searchRun.FileResults))
		for i, fr := range searchRun.FileResults {
			hint := buildHintForFileResult(fr, query)
			payload.Results = append(payload.Results, mcpSearchFileResult{
				Rank:       i + 1,
				FilePath:   fr.FilePath,
				FileName:   fr.FileName,
				Language:   fr.Language,
				Category:   fr.Category,
				Relevance:  fr.Relevance,
				LineRanges: fr.LineRanges,
				Signatures: fr.Signatures,
				Hint:      hint,
			})
		}
	}

	if searchRun.Warning != "" {
		payload.Guidance = "Refresh the project index before trusting semantic ranking."
		payload.NextAction = suggestedRefreshAction(scope)
	}

	return payload
}

func buildHintForFileResult(fr storage.SearchFileResult, query string) string {
	if fr.Relevance >= highConfidenceThreshold {
		return ""
	}
	if fr.Purpose != "" {
		return fr.Purpose
	}
	if len(fr.Signatures) > 0 {
		return fr.Signatures[0]
	}
	switch fr.Category {
	case "source":
		return "source code"
	case "infra_config":
		return "configuration"
	case "docs":
		return "documentation"
	case "scripts":
		return "script"
	case "dependency_metadata":
		return "metadata"
	default:
		return "code"
	}
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
