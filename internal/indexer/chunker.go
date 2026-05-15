package indexer

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"vectos/internal/content"
	"vectos/internal/embeddings"
)

var goFuncPattern = regexp.MustCompile(`^func\s+`)
var jsFuncPattern = regexp.MustCompile(`^(export\s+)?(async\s+)?function\s+|^(export\s+)?(const|let|var)\s+\w+\s*=\s*(async\s*)?\(`)
var jsNamedArrowPattern = regexp.MustCompile(`^(export\s+)?(const|let|var)\s+[A-Za-z_$][\w$]*\s*=\s*(async\s*)?(\([^)]*\)|[A-Za-z_$][\w$]*)\s*=>`)
var jsComponentPattern = regexp.MustCompile(`^(export\s+default\s+function\s+[A-Z][\w$]*|export\s+function\s+[A-Z][\w$]*|function\s+[A-Z][\w$]*|export\s+default\s+(const|let|var)\s+[A-Z][\w$]*\s*=|export\s+(const|let|var)\s+[A-Z][\w$]*\s*=|(const|let|var)\s+[A-Z][\w$]*\s*=)`)
var jsHookPattern = regexp.MustCompile(`^(export\s+)?function\s+use[A-Z][\w$]*|^(export\s+)?(const|let|var)\s+use[A-Z][\w$]*\s*=`)
var jsClassPattern = regexp.MustCompile(`^(export\s+default\s+)?class\s+[A-Za-z_$][\w$]*|^export\s+class\s+[A-Za-z_$][\w$]*`)
var jsTestPattern = regexp.MustCompile(`^(describe|it|test)\s*\(`)
var pyBlockPattern = regexp.MustCompile(`^(def|class)\s+`)
var javaBlockPattern = regexp.MustCompile(`^(public|protected|private|static|final|abstract|class|interface|enum|record)\s+`)
var shellBlockPattern = regexp.MustCompile(`^(function\s+\w+|\w+\s*\(\)\s*\{|if\s|for\s|while\s|case\s)`)
var markdownBlockPattern = regexp.MustCompile(`^(#{1,4}\s|[-*]\s|\d+\.\s|~~~)`)
var tsInterfacePattern = regexp.MustCompile(`^(export\s+)?interface\s+[A-Z][\w$]*`)
var tsTypeAliasPattern = regexp.MustCompile(`^(export\s+)?type\s+[A-Z][\w$]*(<[^>]+>)?\s*=`)
var tsEnumPattern = regexp.MustCompile(`^(export\s+)?(const\s+)?enum\s+[A-Z][\w$]*`)
var tsAsyncPattern = regexp.MustCompile(`(async\s+function\s+[A-Za-z_$]|async\s*\([^)]*\)\s*=>|=\s*async\s*\()`)

// ChunkConfig define los parámetros para la segmentación del código.
type ChunkConfig struct {
	MaxLines int // Máximo de líneas por trozo
	MinLines int // Mínimo de líneas por trozo
}

// ChunkResult contains the content of a chunk and its position.
type ChunkResult struct {
	Content      string
	StartLine    int
	EndLine      int
	Vector       []float32
	Signature    string
	Purpose      string
	SemanticText string // the text sent to the embedder; set by buildChunkImpl
}

// SimpleChunker es una implementación básica de segmentación de archivos.
type SimpleChunker struct {
	config      ChunkConfig
	embedClient embeddings.Embedder
}

// NewSimpleChunker crea una nueva instancia del indexador.
func NewSimpleChunker(config ChunkConfig, embedClient embeddings.Embedder) *SimpleChunker {
	return &SimpleChunker{
		config:      config,
		embedClient: embedClient,
	}
}

// ChunkFile lee un archivo y lo divide en trozos, generando sus embeddings.
func (s *SimpleChunker) ChunkFile(filePath string, language string) ([]ChunkResult, error) {
	return s.chunkFileImpl(filePath, language, true)
}

// ChunkFileRaw lee un archivo y lo divide en trozos sin generar embeddings.
// Los vectores quedan nil y deben ser rellenados posteriormente con
// BatchEmbedChunks.
func (s *SimpleChunker) ChunkFileRaw(filePath string, language string) ([]ChunkResult, error) {
	return s.chunkFileImpl(filePath, language, false)
}

func (s *SimpleChunker) chunkFileImpl(filePath string, language string, embed bool) ([]ChunkResult, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	normalized := strings.ReplaceAll(string(content), "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r", "\n")
	lines := strings.Split(normalized, "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	if language == "go" {
		return s.chunkGoFileImpl(filePath, language, lines, embed)
	}

	if language == "dockerfile" || strings.HasPrefix(language, "yaml") || strings.HasPrefix(language, "bazel") || isLineChunkedLanguage(language) {
		return s.chunkByLinesImpl(filePath, language, lines, embed), nil
	}

	if supportsStructuredChunking(language) {
		return s.chunkStructuredFileImpl(filePath, language, lines, embed), nil
	}

	return s.chunkByLinesImpl(filePath, language, lines, embed), nil
}

func supportsStructuredChunking(language string) bool {
	switch language {
	case "javascript", "typescript", "tsx", "jsx", "python", "java", "shell", "markdown":
		return true
	default:
		return false
	}
}

func isLineChunkedLanguage(language string) bool {
	switch language {
	case "json", "toml", "ini", "xml", "properties", "makefile", "gitignore", "gradle", "lockfile", "config":
		return true
	default:
		return false
	}
}

func (s *SimpleChunker) chunkByLinesImpl(filePath, language string, lines []string, embed bool) []ChunkResult {
	var chunks []ChunkResult
	var currentLines []string
	startLine := 1

	for i, line := range lines {
		currentLines = append(currentLines, line)
		if len(currentLines) < s.config.MaxLines {
			continue
		}

		chunks = append(chunks, s.buildChunkImpl(filePath, language, currentLines, startLine, i+1, embed))
		currentLines = nil
		startLine = i + 2
	}

	if len(currentLines) > 0 {
		chunks = append(chunks, s.buildChunkImpl(filePath, language, currentLines, startLine, len(lines), embed))
	}

	return chunks
}

func (s *SimpleChunker) chunkGoFileImpl(filePath, language string, lines []string, embed bool) ([]ChunkResult, error) {
	var chunks []ChunkResult
	var prelude []string
	preludeEnd := 0

	for i := 0; i < len(lines); {
		trimmed := strings.TrimSpace(lines[i])
		if !goFuncPattern.MatchString(trimmed) {
			prelude = append(prelude, lines[i])
			preludeEnd = i + 1
			i++
			continue
		}

		start := i + 1
		endIndex := findGoBlockEnd(lines, i)

		chunks = append(chunks, s.buildChunkImpl(filePath, language, lines[i:endIndex+1], start, endIndex+1, embed))
		i = endIndex + 1
	}

	if len(chunks) == 0 {
		return s.chunkByLinesImpl(filePath, language, lines, embed), nil
	}

	if len(prelude) > 0 {
		chunks = append([]ChunkResult{s.buildChunkImpl(filePath, language, prelude, 1, preludeEnd, embed)}, chunks...)
	}

	return chunks, nil
}

func findGoBlockEnd(lines []string, startIdx int) int {
	braceDepth := 0
	seenOpeningBrace := false
	for i := startIdx; i < len(lines); i++ {
		line := lines[i]
		braceDepth += strings.Count(line, "{")
		if strings.Contains(line, "{") {
			seenOpeningBrace = true
		}
		braceDepth -= strings.Count(line, "}")
		if seenOpeningBrace && braceDepth <= 0 {
			return i
		}
	}
	return len(lines) - 1
}

func (s *SimpleChunker) chunkStructuredFileImpl(filePath, language string, lines []string, embed bool) []ChunkResult {
	var chunks []ChunkResult
	var prelude []string
	preludeEnd := 0

	if isBraceStructuredLanguage(language) {
		return s.chunkBraceStructuredFileImpl(filePath, language, lines, embed)
	}

	for i := 0; i < len(lines); {
		trimmed := strings.TrimSpace(lines[i])
		if !isStructuredBoundary(language, trimmed) {
			prelude = append(prelude, lines[i])
			preludeEnd = i + 1
			i++
			continue
		}

		start := i + 1
		end := len(lines) - 1
		for j := i + 1; j < len(lines); j++ {
			if isStructuredBoundary(language, strings.TrimSpace(lines[j])) {
				end = j - 1
				break
			}
		}

		chunks = append(chunks, s.buildChunkImpl(filePath, language, lines[i:end+1], start, end+1, embed))
		i = end + 1
	}

	if len(chunks) == 0 {
		return s.chunkByLinesImpl(filePath, language, lines, embed)
	}

	if len(prelude) > 0 {
		chunks = append([]ChunkResult{s.buildChunkImpl(filePath, language, prelude, 1, preludeEnd, embed)}, chunks...)
	}

	return chunks
}

func (s *SimpleChunker) chunkBraceStructuredFileImpl(filePath, language string, lines []string, embed bool) []ChunkResult {
	var chunks []ChunkResult
	var prelude []string
	preludeEnd := 0

	for i := 0; i < len(lines); {
		trimmed := strings.TrimSpace(lines[i])
		if !isStructuredBoundary(language, trimmed) {
			prelude = append(prelude, lines[i])
			preludeEnd = i + 1
			i++
			continue
		}

		start := i + 1
		end := findStructuredBlockEnd(language, lines, i)
		if end < i {
			end = i
		}

		chunks = append(chunks, s.buildChunkImpl(filePath, language, lines[i:end+1], start, end+1, embed))
		i = end + 1
	}

	if len(chunks) == 0 {
		return s.chunkByLinesImpl(filePath, language, lines, embed)
	}

	if len(prelude) > 0 {
		chunks = append([]ChunkResult{s.buildChunkImpl(filePath, language, prelude, 1, preludeEnd, embed)}, chunks...)
	}

	return chunks
}

func isBraceStructuredLanguage(language string) bool {
	switch language {
	case "javascript", "typescript", "tsx", "jsx":
		return true
	default:
		return false
	}
}

func findStructuredBlockEnd(language string, lines []string, startIdx int) int {
	braceDepth := 0
	seenBrace := false
	startedExpr := false
	parenDepth := 0

	for i := startIdx; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)

		braceDepth += strings.Count(line, "{")
		if strings.Contains(line, "{") {
			seenBrace = true
		}
		braceDepth -= strings.Count(line, "}")

		parenDepth += strings.Count(line, "(")
		parenDepth -= strings.Count(line, ")")

		if strings.Contains(line, "=>") || strings.Contains(line, "=") || strings.Contains(line, "function") || strings.Contains(line, "class ") {
			startedExpr = true
		}

		if seenBrace && braceDepth <= 0 {
			return i
		}

		if startedExpr && !seenBrace && parenDepth <= 0 && isLikelyExpressionTerminator(language, trimmed) {
			return i
		}

		if i > startIdx && isStructuredBoundary(language, trimmed) && !seenBrace {
			return i - 1
		}
	}

	return len(lines) - 1
}

func isLikelyExpressionTerminator(language, trimmedLine string) bool {
	if !isBraceStructuredLanguage(language) {
		return false
	}
	if trimmedLine == "" {
		return true
	}
	return strings.HasSuffix(trimmedLine, ";") || strings.HasSuffix(trimmedLine, ")") || strings.HasSuffix(trimmedLine, "/>") || strings.HasSuffix(trimmedLine, ">")
}

func isStructuredBoundary(language, trimmedLine string) bool {
	switch language {
	case "javascript", "typescript", "tsx", "jsx":
		return jsFuncPattern.MatchString(trimmedLine) || jsNamedArrowPattern.MatchString(trimmedLine) || jsComponentPattern.MatchString(trimmedLine) || jsHookPattern.MatchString(trimmedLine) || jsClassPattern.MatchString(trimmedLine) || jsTestPattern.MatchString(trimmedLine) || tsInterfacePattern.MatchString(trimmedLine) || tsEnumPattern.MatchString(trimmedLine)
	case "python":
		return pyBlockPattern.MatchString(trimmedLine)
	case "java":
		return javaBlockPattern.MatchString(trimmedLine)
	case "shell":
		return shellBlockPattern.MatchString(trimmedLine)
	case "markdown":
		return markdownBlockPattern.MatchString(trimmedLine)
	default:
		return false
	}
}

func (s *SimpleChunker) buildChunkImpl(filePath, language string, chunkLines []string, startLine, endLine int, embed bool) ChunkResult {
	chunkContent := strings.Join(chunkLines, "\n")
	signature := extractSignature(language, chunkContent)
	purpose := inferPurpose(language, chunkContent)
	semanticContent := buildSemanticContent(filePath, language, chunkContent)

	var vector []float32
	if embed && s.embedClient != nil {
		var err error
		vector, err = s.embedClient.GetEmbedding(semanticContent)
		if err != nil {
			fmt.Printf("⚠️ Warning: failed to generate embedding for chunk in %s: %v\n", filePath, err)
		}
	}

	return ChunkResult{
		Content:       chunkContent,
		StartLine:     startLine,
		EndLine:       endLine,
		Vector:        vector,
		Signature:     signature,
		Purpose:       purpose,
		SemanticText:  semanticContent,
	}
}

// BatchEmbedChunks fills in the Vector field of every chunk in the slice by
// calling the embedder's GetEmbeddings in batches of at most batchSize.
// Chunks that already have a non-nil Vector are skipped.
func (s *SimpleChunker) BatchEmbedChunks(chunks []ChunkResult, batchSize int) error {
	if s.embedClient == nil {
		return nil
	}
	if batchSize <= 0 {
		batchSize = 32
	}

	// Collect indices of chunks that still need embeddings.
	var pending []int
	var semanticTexts []string
	for i := range chunks {
		if chunks[i].Vector == nil && chunks[i].SemanticText != "" {
			pending = append(pending, i)
			semanticTexts = append(semanticTexts, chunks[i].SemanticText)
		}
	}

	for start := 0; start < len(semanticTexts); start += batchSize {
		end := start + batchSize
		if end > len(semanticTexts) {
			end = len(semanticTexts)
		}
		batch := semanticTexts[start:end]
		vecs, err := s.embedClient.GetEmbeddings(batch)
		if err != nil {
			return fmt.Errorf("batch embedding failed at offset %d: %w", start, err)
		}
		for j, vec := range vecs {
			idx := pending[start+j]
			chunks[idx].Vector = vec
		}
	}

	return nil
}

func buildSemanticContent(filePath, language, chunkContent string) string {
	var sections []string
	sections = append(sections, "File: "+filepath.Base(filePath))

	if signature := extractSignature(language, chunkContent); signature != "" {
		sections = append(sections, "Signature: "+signature)
	}

	if purpose := inferPurpose(language, chunkContent); purpose != "" {
		sections = append(sections, "Purpose: "+purpose)
	}

	sections = append(sections, "Code:\n"+chunkContent)
	return strings.Join(sections, "\n")
}

func extractSignature(language, chunkContent string) string {
	if language != "go" {
		for _, line := range strings.Split(chunkContent, "\n") {
			trimmed := strings.TrimSpace(line)
			if isStructuredBoundary(language, trimmed) {
				return trimmed
			}
		}
		return ""
	}

	for _, line := range strings.Split(chunkContent, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "func ") {
			return trimmed
		}
	}

	return ""
}

func inferPurpose(language, chunkContent string) string {
	return inferNonGoPurpose(language, chunkContent)
}

func inferNonGoPurpose(language, chunkContent string) string {
	lower := strings.ToLower(chunkContent)
	var tags []string

	if isReactComponentChunk(language, chunkContent) {
		tags = append(tags, "react component")
	}
	if isHookChunk(language, chunkContent) {
		tags = append(tags, "custom hook")
	}
	if isTestChunk(language, chunkContent) {
		tags = append(tags, "test block")
	}
	if isExportedChunk(language, chunkContent) {
		tags = append(tags, "exported api")
	}
	if isInterfaceChunk(language, chunkContent) || isTypeAliasChunk(language, chunkContent) {
		tags = append(tags, "type definition")
	}
	if isEnumChunk(language, chunkContent) {
		tags = append(tags, "enumeration")
	}
	if isAsyncChunk(language, chunkContent) {
		tags = append(tags, "async function")
	}

	category := content.ClassifyCategory(language)

	if result, done := inferDocsCategory(lower, category, tags); done {
		return result
	}
	if result, done := inferScriptsCategory(lower, category, tags); done {
		return result
	}
	if result, done := inferDependencyCategory(lower, category, tags); done {
		return result
	}
	// Note: replicates original double call to ClassifyCategory
	if result, done := inferInfraCategory(lower, language, tags); done {
		return result
	}

	if strings.Contains(lower, "fetch") || strings.Contains(lower, "axios") {
		tags = append(tags, "network or api access")
	}
	if strings.Contains(lower, "def ") || strings.Contains(lower, "function ") || strings.Contains(lower, "=>") {
		tags = append(tags, "function or callable block")
	}
	if strings.Contains(lower, "class ") {
		tags = append(tags, "class definition")
	}
	if strings.Contains(lower, "return") {
		tags = append(tags, "returns computed values")
	}
	if len(tags) == 0 {
		return "code block"
	}
	return strings.Join(tags, "; ")
}

func inferDocsCategory(lower, category string, tags []string) (string, bool) {
	if category != "docs" {
		return "", false
	}
	if strings.Contains(lower, "install") || strings.Contains(lower, "usage") {
		tags = append(tags, "documentation or usage instructions")
	}
	if len(tags) == 0 {
		return "documentation content", true
	}
	return strings.Join(tags, "; "), true
}

func inferScriptsCategory(lower, category string, tags []string) (string, bool) {
	if category != "scripts" {
		return "", false
	}
	if strings.Contains(lower, "#!/bin/") {
		tags = append(tags, "shell script entrypoint")
	}
	if strings.Contains(lower, "export ") || strings.Contains(lower, "set -") {
		tags = append(tags, "environment or execution setup")
	}
	if len(tags) == 0 {
		return "script or automation block", true
	}
	return strings.Join(tags, "; "), true
}

func inferDependencyCategory(lower, category string, tags []string) (string, bool) {
	if category != "dependency_metadata" {
		return "", false
	}
	if strings.Contains(lower, "dependencies") || strings.Contains(lower, "require") {
		tags = append(tags, "dependency or project metadata")
	}
	if len(tags) == 0 {
		return "project or dependency metadata", true
	}
	return strings.Join(tags, "; "), true
}

func inferInfraCategory(lower, language string, tags []string) (string, bool) {
	if content.ClassifyCategory(language) != "infra_config" {
		return "", false
	}
	if strings.Contains(lower, "image:") || strings.Contains(lower, "docker") {
		tags = append(tags, "container or image configuration")
	}
	if strings.Contains(lower, "service") || strings.Contains(lower, "services:") {
		tags = append(tags, "service configuration")
	}
	if strings.Contains(lower, "rule") || strings.Contains(lower, "load(") {
		tags = append(tags, "build or workspace configuration")
	}
	if len(tags) == 0 {
		return "infrastructure or configuration block", true
	}
	return strings.Join(tags, "; "), true
}

func isReactComponentChunk(language, chunkContent string) bool {
	if language != "javascript" && language != "typescript" && language != "tsx" && language != "jsx" {
		return false
	}
	for _, line := range strings.Split(chunkContent, "\n") {
		if jsComponentPattern.MatchString(strings.TrimSpace(line)) {
			return true
		}
	}
	return false
}

func isHookChunk(language, chunkContent string) bool {
	if language != "javascript" && language != "typescript" && language != "tsx" && language != "jsx" {
		return false
	}
	for _, line := range strings.Split(chunkContent, "\n") {
		if jsHookPattern.MatchString(strings.TrimSpace(line)) {
			return true
		}
	}
	return false
}

func isTestChunk(language, chunkContent string) bool {
	if language != "javascript" && language != "typescript" && language != "tsx" && language != "jsx" {
		return false
	}
	for _, line := range strings.Split(chunkContent, "\n") {
		if jsTestPattern.MatchString(strings.TrimSpace(line)) {
			return true
		}
	}
	return false
}

func isExportedChunk(language, chunkContent string) bool {
	if language != "javascript" && language != "typescript" && language != "tsx" && language != "jsx" {
		return false
	}
	for _, line := range strings.Split(chunkContent, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "export ") {
			return true
		}
	}
	return false
}

func isInterfaceChunk(language, chunkContent string) bool {
	if language != "javascript" && language != "typescript" && language != "tsx" && language != "jsx" {
		return false
	}
	for _, line := range strings.Split(chunkContent, "\n") {
		if tsInterfacePattern.MatchString(strings.TrimSpace(line)) {
			return true
		}
	}
	return false
}

func isTypeAliasChunk(language, chunkContent string) bool {
	if language != "javascript" && language != "typescript" && language != "tsx" && language != "jsx" {
		return false
	}
	for _, line := range strings.Split(chunkContent, "\n") {
		if tsTypeAliasPattern.MatchString(strings.TrimSpace(line)) {
			return true
		}
	}
	return false
}

func isEnumChunk(language, chunkContent string) bool {
	if language != "javascript" && language != "typescript" && language != "tsx" && language != "jsx" {
		return false
	}
	for _, line := range strings.Split(chunkContent, "\n") {
		if tsEnumPattern.MatchString(strings.TrimSpace(line)) {
			return true
		}
	}
	return false
}

func isAsyncChunk(language, chunkContent string) bool {
	if language != "javascript" && language != "typescript" && language != "tsx" && language != "jsx" {
		return false
	}
	for _, line := range strings.Split(chunkContent, "\n") {
		if tsAsyncPattern.MatchString(strings.TrimSpace(line)) {
			return true
		}
	}
	return false
}
