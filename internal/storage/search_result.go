package storage

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

const (
	// maxPreviewBytes is the hard cap for preview text per file result.
	// Keeps MCP payloads compact (~180 bytes × 5 results ≈ 900 bytes of previews).
	maxPreviewBytes = 180
)

type LineRange struct {
	Start int `json:"start"`
	End   int `json:"end"`
}

type SearchFileResult struct {
	FilePath   string      `json:"file_path"`
	FileName   string      `json:"file_name"`
	Language   string      `json:"language,omitempty"`
	Category   string      `json:"category,omitempty"`
	Relevance  float64     `json:"relevance"`
	LineRanges []LineRange `json:"line_ranges"`
	Signatures []string    `json:"signatures"`
	Purpose    string      `json:"purpose,omitempty"`
	Hint       string      `json:"hint,omitempty"`
	Preview    string      `json:"preview,omitempty"`
}

type fileEntry struct {
	result    SearchFileResult
	sigSet    map[string]struct{}
	seenLines map[string]struct{}
	purpose   string
}

func CollapseFileResults(chunks []CodeChunk, lineWindow int) []SearchFileResult {
	if len(chunks) == 0 {
		return nil
	}

	sort.Slice(chunks, func(i, j int) bool {
		if chunks[i].Score != chunks[j].Score {
			return chunks[i].Score > chunks[j].Score
		}
		return chunks[i].FilePath < chunks[j].FilePath
	})

	byFile := make(map[string]*fileEntry)
	for _, chunk := range chunks {
		addChunkToFileEntry(byFile, chunk)
	}

	deduped := make([]SearchFileResult, 0, len(byFile))
	for _, fe := range byFile {
		if fe.purpose != "" {
			fe.result.Purpose = fe.purpose
		}
		if len(fe.result.LineRanges) > 1 {
			fe.result.LineRanges = mergeLineRanges(fe.result.LineRanges, lineWindow)
		}
		deduped = append(deduped, fe.result)
	}

	sort.Slice(deduped, func(i, j int) bool {
		return deduped[i].Relevance > deduped[j].Relevance
	})

	return deduped
}

func addChunkToFileEntry(byFile map[string]*fileEntry, chunk CodeChunk) {
	fe, ok := byFile[chunk.FilePath]
	if !ok {
		fe = &fileEntry{
			result: SearchFileResult{
				FilePath:  chunk.FilePath,
				FileName:  filepath.Base(chunk.FilePath),
				Language:  chunk.Language,
				Category:  chunk.Category,
				Relevance: chunk.Score,
				Preview:   ExtractChunkPreview(chunk.Content, maxPreviewBytes),
			},
			sigSet:    make(map[string]struct{}),
			seenLines: make(map[string]struct{}),
		}
		byFile[chunk.FilePath] = fe
	}

	if chunk.Score > fe.result.Relevance {
		fe.result.Relevance = chunk.Score
		// Update preview from the higher-scoring chunk.
		if p := ExtractChunkPreview(chunk.Content, maxPreviewBytes); p != "" {
			fe.result.Preview = p
		}
	}

	if chunk.Purpose != "" && fe.purpose == "" {
		fe.purpose = chunk.Purpose
	}

	key := fmt.Sprintf("%d-%d", chunk.StartLine, chunk.EndLine)
	if _, seen := fe.seenLines[key]; !seen {
		fe.seenLines[key] = struct{}{}
		fe.result.LineRanges = append(fe.result.LineRanges, LineRange{
			Start: chunk.StartLine,
			End:   chunk.EndLine,
		})
	}

	if chunk.Signature != "" {
		if _, exists := fe.sigSet[chunk.Signature]; !exists {
			fe.sigSet[chunk.Signature] = struct{}{}
			fe.result.Signatures = append(fe.result.Signatures, chunk.Signature)
		}
	}
}

func mergeLineRanges(ranges []LineRange, window int) []LineRange {
	if len(ranges) <= 1 {
		return ranges
	}

	sort.Slice(ranges, func(i, j int) bool {
		return ranges[i].Start < ranges[j].Start
	})

	merged := []LineRange{ranges[0]}

	for i := 1; i < len(ranges); i++ {
		last := &merged[len(merged)-1]
		if ranges[i].Start <= last.End+window {
			if ranges[i].End > last.End {
				last.End = ranges[i].End
			}
		} else {
			merged = append(merged, ranges[i])
		}
	}

	return merged
}

// ExtractChunkPreview produces a compact, single-line preview from chunk content.
// It collapses whitespace, trims to maxBytes, and appends "..." when truncated.
// Returns "" for empty or whitespace-only content.
func ExtractChunkPreview(content string, maxBytes int) string {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return ""
	}

	// Collapse all whitespace (newlines, tabs, runs of spaces) into single spaces.
	trimmed = strings.Join(strings.Fields(trimmed), " ")

	if maxBytes > 0 && len(trimmed) > maxBytes {
		// Truncate at a word boundary if possible, otherwise hard-cut.
		cut := maxBytes - 3 // room for "..."
		if cut <= 0 {
			return "..."
		}
		// Try to break at the last space before the cut point.
		if idx := strings.LastIndex(trimmed[:cut], " "); idx > cut/2 {
			cut = idx
		}
		return trimmed[:cut] + "..."
	}
	return trimmed
}
