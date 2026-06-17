package embeddings

import "fmt"

// Embedder defines the contract for any embedding provider.
// The interface distinguishes between query and passage embeddings so that
// models supporting task-specific adapters (e.g. jina-embeddings-v3) can
// route to the correct adapter at inference time.
type Embedder interface {
	// EmbedQuery embeds a single search query (task_id=0 for jina-v3).
	EmbedQuery(text string) ([]float32, error)
	// EmbedQueries embeds multiple search queries in one call.
	EmbedQueries(texts []string) ([][]float32, error)
	// EmbedPassage embeds a single passage/document for indexing (task_id=1 for jina-v3).
	EmbedPassage(text string) ([]float32, error)
	// EmbedPassages embeds multiple passages/documents in one call.
	EmbedPassages(texts []string) ([][]float32, error)
}

// EmbedPassagesDefault implements EmbedPassages by calling EmbedPassage in a
// loop. Useful for embedders that lack a native batch implementation.
func EmbedPassagesDefault(e Embedder, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}
	results := make([][]float32, len(texts))
	for i, text := range texts {
		vec, err := e.EmbedPassage(text)
		if err != nil {
			return nil, fmt.Errorf("batch embedding text %d: %w", i, err)
		}
		results[i] = vec
	}
	return results, nil
}
