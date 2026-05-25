package indexer

import (
	"fmt"
	"strings"

	ts "github.com/odvcencio/gotreesitter"
	"github.com/odvcencio/gotreesitter/grammars"
)

// astChunkBoundary represents a chunk boundary discovered from the AST.
type astChunkBoundary struct {
	startByte uint32
	endByte   uint32
}

// loadTreeSitterLanguage returns the tree-sitter Language for a given Vectos
// language string, or nil if no grammar is available.
func loadTreeSitterLanguage(language string) *ts.Language {
	switch language {
	case "tsx":
		return grammars.TsxLanguage()
	case "typescript":
		return grammars.TypescriptLanguage()
	case "javascript":
		return grammars.JavascriptLanguage()
	case "jsx":
		return grammars.JavascriptLanguage()
	case "go":
		return grammars.GoLanguage()
	case "python":
		return grammars.PythonLanguage()
	case "java":
		return grammars.JavaLanguage()
	case "shell":
		return grammars.BashLanguage()
	default:
		return nil
	}
}

// supportsTreeSitter reports whether a language has a tree-sitter grammar.
func supportsTreeSitter(language string) bool {
	return loadTreeSitterLanguage(language) != nil
}

// astChunkBoundaries returns chunk boundaries from the AST root node.
// Top-level named declarations become chunk boundaries. Import statements
// are grouped into prelude chunks (resetting between non-contiguous blocks).
func astChunkBoundaries(root *ts.Node, lang *ts.Language, maxBytes uint32) []astChunkBoundary {
	if maxBytes == 0 {
		maxBytes = 2500
	}
	childCount := root.NamedChildCount()
	if childCount == 0 {
		return nil
	}

	var boundaries []astChunkBoundary
	importStart := ^uint32(0) // sentinel: no pending import block
	inImports := false

	for i := 0; i < childCount; i++ {
		child := root.NamedChild(i)
		typ := child.Type(lang)
		start := child.StartByte()
		end := child.EndByte()

		isImport := typ == "import_statement" ||
			typ == "import_declaration" ||
			typ == "future_import_statement"
		isDeclaration := typ == "function_declaration" ||
			typ == "export_statement" ||
			typ == "class_declaration" ||
			typ == "method_declaration" ||
			typ == "method_definition" ||
			typ == "interface_declaration" ||
			typ == "type_alias_declaration" ||
			typ == "type_declaration" ||
			typ == "var_declaration" ||
			typ == "const_declaration" ||
			typ == "enum_declaration"

		if isImport {
			if !inImports {
				importStart = start
				inImports = true
			}
			continue
		}

		// Flush pending import block (resets importStart for next group).
		if inImports {
			boundaries = append(boundaries, astChunkBoundary{startByte: importStart, endByte: start})
			importStart = ^uint32(0)
			inImports = false
		}

		if isDeclaration {
			boundaries = append(boundaries, splitDeclarationNode(child, lang, maxBytes)...)
		} else {
			if end-start > maxBytes {
				boundaries = append(boundaries, splitOversizedNode(child, lang, maxBytes, start)...)
			} else {
				boundaries = append(boundaries, astChunkBoundary{startByte: start, endByte: end})
			}
		}
	}

	// Flush trailing import block.
	if inImports {
		boundaries = append(boundaries, astChunkBoundary{startByte: importStart, endByte: root.EndByte()})
	}

	return boundaries
}

// splitDeclarationNode splits a function/class/export declaration at its
// named children when the declaration exceeds maxBytes. The first chunk
// always includes the signature (identifier + params) plus the first body
// statement so subsequent chunks retain meaningful context.
func splitDeclarationNode(node *ts.Node, lang *ts.Language, maxBytes uint32) []astChunkBoundary {
	start := node.StartByte()
	end := node.EndByte()
	childCount := node.NamedChildCount()

	if end-start <= maxBytes || childCount <= 1 {
		return []astChunkBoundary{{startByte: start, endByte: end}}
	}

	var boundaries []astChunkBoundary
	// segStart initially covers the entire node; we'll re-anchor after
	// the first body child is consumed with the signature.
	segStart := start
	consumedSignature := false

	for i := 0; i < childCount; i++ {
		child := node.NamedChild(i)
		childEnd := child.EndByte()

		if !consumedSignature {
			// Include the signature (i==0) PLUS the first body statement (i==1).
			if i <= 1 {
				consumedSignature = true
				continue
			}
		}

		if childEnd-segStart > maxBytes && childEnd > segStart {
			if child.StartByte() > segStart {
				boundaries = append(boundaries, astChunkBoundary{startByte: segStart, endByte: child.StartByte()})
			}
			segStart = child.StartByte()
		}
	}

	if segStart < end {
		if len(boundaries) == 0 {
			return []astChunkBoundary{{startByte: start, endByte: end}}
		}
		boundaries = append(boundaries, astChunkBoundary{startByte: segStart, endByte: end})
	}

	return boundaries
}

// splitOversizedNode splits a generic oversized top-level node at its named
// children boundaries.
func splitOversizedNode(node *ts.Node, lang *ts.Language, maxBytes uint32, nodeStart uint32) []astChunkBoundary {
	childCount := node.NamedChildCount()
	if childCount <= 1 {
		return []astChunkBoundary{{startByte: nodeStart, endByte: node.EndByte()}}
	}

	var children []ts.Node
	for i := 0; i < childCount; i++ {
		children = append(children, *node.NamedChild(i))
	}

	var boundaries []astChunkBoundary
	segStart := nodeStart

	for _, child := range children {
		childEnd := child.EndByte()
		if childEnd-segStart > maxBytes && childEnd > segStart {
			if child.StartByte() > segStart {
				boundaries = append(boundaries, astChunkBoundary{startByte: segStart, endByte: child.StartByte()})
			}
			segStart = child.StartByte()
		}
	}

	if segStart < node.EndByte() {
		if len(boundaries) == 0 {
			return []astChunkBoundary{{startByte: nodeStart, endByte: node.EndByte()}}
		}
		boundaries = append(boundaries, astChunkBoundary{startByte: segStart, endByte: node.EndByte()})
	}

	return boundaries
}

// chunkASTFileImpl parses source with tree-sitter and extracts chunks at AST boundaries.
func (s *SimpleChunker) chunkASTFileImpl(filePath, language string, source []byte, embed bool) ([]ChunkResult, error) {
	lang := loadTreeSitterLanguage(language)
	if lang == nil {
		return nil, fmt.Errorf("no tree-sitter grammar for %s", language)
	}

	parser := ts.NewParser(lang)
	tree, err := parser.Parse(source)
	if err != nil {
		return nil, fmt.Errorf("tree-sitter parse error: %w", err)
	}
	defer tree.Release()

	root := tree.RootNode()
	maxBytes := uint32(s.config.MaxChars)
	if maxBytes == 0 {
		maxBytes = 2500
	}
	boundaries := astChunkBoundaries(root, lang, maxBytes)

	sourceStr := string(source)
	lines := strings.Split(sourceStr, "\n")
	lineByteOffsets := buildLineByteOffsets(sourceStr)

	var chunks []ChunkResult
	for _, b := range boundaries {
		startLine := byteOffsetToLine(lineByteOffsets, int(b.startByte)) + 1
		endLine := byteOffsetToLine(lineByteOffsets, int(b.endByte)) + 1
		chunkContent := sourceStr[b.startByte:b.endByte]
		chunkLines := extractLines(lines, startLine-1, endLine)

		chunk := s.buildChunkImpl(filePath, language, chunkLines, startLine, endLine, embed)
		chunk.Content = chunkContent
		chunks = append(chunks, chunk)
	}

	// AST chunk boundaries are semantically meaningful (function/class/export
	// declarations) — small chunks at these boundaries are correct and should
	// not be merged. The heuristic mergeTinyFragments only applies to line-chunked
	// content where boundaries are arbitrary.

	return chunks, nil
}

// buildLineByteOffsets returns the cumulative byte offset for each line.
func buildLineByteOffsets(source string) []int {
	offsets := make([]int, 0, strings.Count(source, "\n")+1)
	offsets = append(offsets, 0)
	for i, b := range source {
		if b == '\n' {
			offsets = append(offsets, i+1)
		}
	}
	return offsets
}

// byteOffsetToLine returns the line index (0-based) for a byte offset.
func byteOffsetToLine(offsets []int, byteOffset int) int {
	lo, hi := 0, len(offsets)-1
	for lo < hi {
		mid := (lo + hi + 1) / 2
		if offsets[mid] <= byteOffset {
			lo = mid
		} else {
			hi = mid - 1
		}
	}
	return lo
}

// extractLines returns a slice of lines between startLine (inclusive) and
// endLine (exclusive, or len(lines) if endLine > len(lines)).
func extractLines(lines []string, startLine, endLine int) []string {
	if startLine < 0 {
		startLine = 0
	}
	if endLine > len(lines) {
		endLine = len(lines)
	}
	if startLine >= endLine {
		return nil
	}
	return lines[startLine:endLine]
}
