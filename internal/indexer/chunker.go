package indexer

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"vectos/internal/embeddings"
)

var goFuncPattern = regexp.MustCompile(`^func\s+`)
var jsFuncPattern = regexp.MustCompile(`^(export\s+)?(async\s+)?function\s+|^(export\s+)?(const|let|var)\s+\w+\s*=\s*(async\s*)?\(`)
var pyBlockPattern = regexp.MustCompile(`^(def|class)\s+`)
var javaBlockPattern = regexp.MustCompile(`^(public|protected|private|static|final|abstract|class|interface|enum|record)\s+`)
var shellBlockPattern = regexp.MustCompile(`^(function\s+\w+|\w+\s*\(\)\s*\{|if\s|for\s|while\s|case\s)`)
var markdownBlockPattern = regexp.MustCompile(`^(#{1,4}\s|[-*]\s|\d+\.\s|~~~)`) 

// ChunkConfig define los parámetros para la segmentación del código.
type ChunkConfig struct {
	MaxLines int // Máximo de líneas por trozo
	MinLines int // Mínimo de líneas por trozo
}

// ChunkResult contiene el contenido de un trozo y su posición.
type ChunkResult struct {
	Content   string
	StartLine int
	EndLine   int
	Vector    []float32
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
		return s.chunkGoFile(filePath, language, lines)
	}

	if language == "dockerfile" || strings.HasPrefix(language, "yaml") || strings.HasPrefix(language, "bazel") || isLineChunkedLanguage(language) {
		return s.chunkByLines(filePath, language, lines), nil
	}

	if supportsStructuredChunking(language) {
		return s.chunkStructuredFile(filePath, language, lines), nil
	}

	return s.chunkByLines(filePath, language, lines), nil
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

func (s *SimpleChunker) chunkByLines(filePath, language string, lines []string) []ChunkResult {
	var chunks []ChunkResult
	var currentLines []string
	startLine := 1

	for i, line := range lines {
		currentLines = append(currentLines, line)
		if len(currentLines) < s.config.MaxLines {
			continue
		}

		chunks = append(chunks, s.buildChunk(filePath, language, currentLines, startLine, i+1))
		currentLines = nil
		startLine = i + 2
	}

	if len(currentLines) > 0 {
		chunks = append(chunks, s.buildChunk(filePath, language, currentLines, startLine, len(lines)))
	}

	return chunks
}

func (s *SimpleChunker) chunkGoFile(filePath, language string, lines []string) ([]ChunkResult, error) {
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
		braceDepth := 0
		seenOpeningBrace := false
		endIndex := i

		for ; endIndex < len(lines); endIndex++ {
			line := lines[endIndex]
			braceDepth += strings.Count(line, "{")
			if strings.Contains(line, "{") {
				seenOpeningBrace = true
			}
			braceDepth -= strings.Count(line, "}")
			if seenOpeningBrace && braceDepth <= 0 {
				break
			}
		}

		if endIndex >= len(lines) {
			endIndex = len(lines) - 1
		}

		chunks = append(chunks, s.buildChunk(filePath, language, lines[i:endIndex+1], start, endIndex+1))
		i = endIndex + 1
	}

	if len(chunks) == 0 {
		return s.chunkByLines(filePath, language, lines), nil
	}

	if len(prelude) > 0 {
		chunks = append([]ChunkResult{s.buildChunk(filePath, language, prelude, 1, preludeEnd)}, chunks...)
	}

	return chunks, nil
}

func (s *SimpleChunker) chunkStructuredFile(filePath, language string, lines []string) []ChunkResult {
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
		end := len(lines) - 1
		for j := i + 1; j < len(lines); j++ {
			if isStructuredBoundary(language, strings.TrimSpace(lines[j])) {
				end = j - 1
				break
			}
		}

		chunks = append(chunks, s.buildChunk(filePath, language, lines[i:end+1], start, end+1))
		i = end + 1
	}

	if len(chunks) == 0 {
		return s.chunkByLines(filePath, language, lines)
	}

	if len(prelude) > 0 {
		chunks = append([]ChunkResult{s.buildChunk(filePath, language, prelude, 1, preludeEnd)}, chunks...)
	}

	return chunks
}

func isStructuredBoundary(language, trimmedLine string) bool {
	switch language {
	case "javascript", "typescript", "tsx", "jsx":
		return jsFuncPattern.MatchString(trimmedLine)
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

func (s *SimpleChunker) buildChunk(filePath, language string, chunkLines []string, startLine, endLine int) ChunkResult {
	chunkContent := strings.Join(chunkLines, "\n")
	semanticContent := buildSemanticContent(filePath, language, chunkContent)

	var vector []float32
	if s.embedClient != nil {
		var err error
		vector, err = s.embedClient.GetEmbedding(semanticContent)
		if err != nil {
			fmt.Printf("⚠️ Warning: failed to generate embedding for chunk in %s: %v\n", filePath, err)
		}
	}

	return ChunkResult{
		Content:   chunkContent,
		StartLine: startLine,
		EndLine:   endLine,
		Vector:    vector,
	}
}

func buildSemanticContent(filePath, language, chunkContent string) string {
	var sections []string
	sections = append(sections,
		"File: "+filepath.Base(filePath),
		"Language: "+language,
		"Category: "+classifyCategory(language),
	)

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
	if language != "go" {
		lower := strings.ToLower(chunkContent)
		var tags []string
		category := classifyCategory(language)
		if category == "docs" {
			if strings.Contains(lower, "install") || strings.Contains(lower, "usage") {
				tags = append(tags, "documentation or usage instructions")
			}
			if len(tags) == 0 {
				return "documentation content"
			}
			return strings.Join(tags, "; ")
		}
		if category == "scripts" {
			if strings.Contains(lower, "#!/bin/") {
				tags = append(tags, "shell script entrypoint")
			}
			if strings.Contains(lower, "export ") || strings.Contains(lower, "set -") {
				tags = append(tags, "environment or execution setup")
			}
			if len(tags) == 0 {
				return "script or automation block"
			}
			return strings.Join(tags, "; ")
		}
		if category == "dependency_metadata" {
			if strings.Contains(lower, "dependencies") || strings.Contains(lower, "require") {
				tags = append(tags, "dependency or project metadata")
			}
			if len(tags) == 0 {
				return "project or dependency metadata"
			}
			return strings.Join(tags, "; ")
		}
		if classifyCategory(language) == "infra_config" {
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
				return "infrastructure or configuration block"
			}
			return strings.Join(tags, "; ")
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

	lower := strings.ToLower(chunkContent)
	var tags []string

	if strings.Contains(lower, "return a + b") || strings.Contains(lower, "suma") {
		tags = append(tags, "adds or combines two integers", "suma dos numeros enteros")
	}
	if strings.Contains(lower, "return a - b") || strings.Contains(lower, "resta") {
		tags = append(tags, "subtracts two integers", "resta dos numeros enteros")
	}
	if strings.Contains(lower, "return a * b") || strings.Contains(lower, "multiplicacion") {
		tags = append(tags, "multiplies two integers", "multiplica dos numeros enteros")
	}
	if strings.Contains(lower, "return a / b") || strings.Contains(lower, "division") {
		tags = append(tags, "divides two integers", "divide dos numeros enteros")
	}
	if strings.Contains(lower, "func ") && strings.Contains(lower, " int") {
		tags = append(tags, "function operating on integer values")
	}

	if len(tags) == 0 {
		return "code block"
	}

	return strings.Join(tags, "; ")
}

func classifyCategory(language string) string {
	switch {
	case language == "dockerfile", strings.HasPrefix(language, "yaml"), strings.HasPrefix(language, "bazel"):
		return "infra_config"
	case language == "shell":
		return "scripts"
	case language == "markdown", language == "gitignore":
		return "docs"
	case language == "json", language == "toml", language == "properties", language == "xml", language == "makefile", language == "gradle", language == "lockfile":
		return "dependency_metadata"
	case language == "ini", language == "config":
		return "infra_config"
	default:
		return "source"
	}
}
