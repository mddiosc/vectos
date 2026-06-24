package main

import (
	"fmt"
	"strings"

	"vectos/internal/storage"
)

const (
	searchPreviewShort  = 96
	searchPreviewMedium = 144
	searchPreviewLong   = 208
)

func formatSearchResults(query string, results []storage.CodeChunk, searchMode string, full bool) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("%s %s\n", dim("Search mode:"), searchMode))
	b.WriteString(bold(fmt.Sprintf("Found %d result(s):", len(results))) + "\n\n")
	previewLimit := adaptivePreviewLimit(query, results)

	for i, r := range results {
		b.WriteString(formatSearchResult(r, searchMode, full, previewLimitForResult(previewLimit, i)))
	}

	return b.String()
}

func formatSearchResult(r storage.CodeChunk, searchMode string, full bool, previewLimit int) string {
	var b strings.Builder
	location := cyan(bold(fmt.Sprintf("%s:%d-%d", r.FilePath, r.StartLine, r.EndLine)))
	b.WriteString(fmt.Sprintf("%s %s", location, dim(fmt.Sprintf("[%s/%s]", r.Category, r.Language))))
	if r.Score != 0 {
		b.WriteString(" " + colorScore(r.Score))
	}
	b.WriteString("\n")
	if reason := explainResultReason(searchMode, r); reason != "" {
		b.WriteString(dim(fmt.Sprintf("reason: %s", reason)) + "\n")
	}
	if full {
		b.WriteString(fmt.Sprintf("%s\n\n", strings.TrimSpace(r.Content)))
		return b.String()
	}
	b.WriteString(fmt.Sprintf("%s\n\n", compactPreviewLimit(r.Content, previewLimit)))
	return b.String()
}

func adaptivePreviewLimit(query string, results []storage.CodeChunk) int {
	queryTokens := tokenizeForRanking(query)
	if len(results) == 0 {
		return searchPreviewMedium
	}

	limit := searchPreviewMedium
	if len(queryTokens) <= 3 {
		limit = searchPreviewShort
	}

	if len(results) == 1 {
		return limit
	}

	first := results[0].Score
	second := results[1].Score
	gap := first - second
	if gap >= 0.08 {
		return searchPreviewShort
	}
	if gap >= 0.03 {
		return minInt(limit, searchPreviewMedium)
	}
	if len(queryTokens) >= 6 {
		return searchPreviewLong
	}
	return searchPreviewMedium
}

func previewLimitForResult(baseLimit int, index int) int {
	if index == 0 {
		return baseLimit
	}
	if baseLimit <= searchPreviewShort {
		return searchPreviewShort
	}
	if index == 1 {
		return minInt(baseLimit, searchPreviewMedium)
	}
	return minInt(baseLimit, searchPreviewShort)
}

func compactPreviewLimit(content string, limit int) string {
	trimmed := strings.TrimSpace(content)
	trimmed = strings.ReplaceAll(trimmed, "\n", " ")
	trimmed = strings.Join(strings.Fields(trimmed), " ")
	if limit > 0 && len(trimmed) > limit {
		return trimmed[:maxInt(0, limit-3)] + "..."
	}
	return trimmed
}

// colorScore tints the score by confidence: green strong, yellow moderate,
// dim weak. ponytail: fixed thresholds, tune if score distribution shifts.
func colorScore(score float64) string {
	label := fmt.Sprintf("score=%.4f", score)
	switch {
	case score >= 0.6:
		return green(label)
	case score >= 0.4:
		return yellow(label)
	default:
		return dim(label)
	}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
