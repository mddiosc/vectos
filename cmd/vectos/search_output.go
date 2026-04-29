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
	b.WriteString(fmt.Sprintf("Search mode: %s\n", searchMode))
	b.WriteString(fmt.Sprintf("Found %d result(s):\n\n", len(results)))
	previewLimit := adaptivePreviewLimit(query, results)

	for i, r := range results {
		b.WriteString(formatSearchResult(r, searchMode, full, previewLimitForResult(previewLimit, i)))
	}

	return b.String()
}

func formatSearchResult(r storage.CodeChunk, searchMode string, full bool, previewLimit int) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("[%s:%d-%d] [%s/%s]", r.FilePath, r.StartLine, r.EndLine, r.Category, r.Language))
	if r.Score != 0 {
		b.WriteString(fmt.Sprintf(" score=%.4f", r.Score))
	}
	b.WriteString("\n")
	if reason := explainResultReason(searchMode, r); reason != "" {
		b.WriteString(fmt.Sprintf("reason: %s\n", reason))
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
