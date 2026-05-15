package embeddings

import "fmt"

// Embedder define el contrato para cualquier proveedor de embeddings.
// Esto permite que el sistema sea agnóstico al origen de los vectores (local o remoto).
type Embedder interface {
	// GetEmbedding convierte un texto en un vector de floats.
	GetEmbedding(text string) ([]float32, error)

	// GetEmbeddings convierte múltiples textos en vectores de floats en una
	// sola llamada. Los vectores se devuelven en el mismo orden que los textos
	// de entrada.
	GetEmbeddings(texts []string) ([][]float32, error)
}

// GetEmbeddingsDefault implementa GetEmbeddings llamando a GetEmbedding en un
// bucle. Es útil para backward compatibility o para embedders que no tienen
// una implementación nativa de batch.
func GetEmbeddingsDefault(e Embedder, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}
	results := make([][]float32, len(texts))
	for i, text := range texts {
		vec, err := e.GetEmbedding(text)
		if err != nil {
			return nil, fmt.Errorf("batch embedding text %d: %w", i, err)
		}
		results[i] = vec
	}
	return results, nil
}
