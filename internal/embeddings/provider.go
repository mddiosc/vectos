package embeddings

// Embedder define el contrato para cualquier proveedor de embeddings.
// Esto permite que el sistema sea agnóstico al origen de los vectores (local o remoto).
type Embedder interface {
	// GetEmbedding convierte un texto en un vector de floats.
	GetEmbedding(text string) ([]float32, error)
}
