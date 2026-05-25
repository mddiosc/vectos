package main

import (
	"fmt"

	"vectos/internal/indexer"
)

const currentChunkerVersion = 4 // tree-sitter AST chunking for TSX/TS/JS/Go/Python/Java/Shell

func indexChunkerConfig(batchSize int) indexer.ChunkConfig {
	return indexer.ChunkConfig{
		MaxLines:      10,
		BatchSize:     batchSize,
		TargetChars:   1200,
		MaxChars:      2500,
		MinChunkChars: 200,
	}
}

func currentIndexFingerprint(cfg indexer.ChunkConfig) string {
	return fmt.Sprintf(
		"chunker:v%d|maxlines=%d|target=%d|max=%d|min=%d",
		currentChunkerVersion,
		cfg.MaxLines,
		cfg.TargetChars,
		cfg.MaxChars,
		cfg.MinChunkChars,
	)
}
