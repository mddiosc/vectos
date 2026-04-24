package storage

import "time"

// CodeChunk representa un fragmento de código procesado y listo para ser buscado.
type CodeChunk struct {
	ID        int64     `json:"id"`         // ID único en la base de datos
	FilePath  string    `json:"file_path"`  // Ruta absoluta del archivo
	Content   string    `json:"content"`    // El texto real del código
	StartLine int       `json:"start_line"` // Línea donde empieza el fragmento
	EndLine   int       `json:"end_line"`   // Línea donde termina
	Language  string    `json:"language"`   // Lenguaje de programación (ej: "go", "typescript")
	Category  string    `json:"category,omitempty"`
	CreatedAt time.Time `json:"created_at"` // Fecha de indexación
	Score     float64   `json:"score,omitempty"`

	// El campo Vector se usará para la búsqueda semántica (embeddings).
	// En SQLite se guardará como un BLOB o mediante una extensión vectorial.
	Vector []float32 `json:"-"`
}

type IndexMetadata struct {
	Provider   string    `json:"provider"`
	Model      string    `json:"model"`
	Dimensions int       `json:"dimensions"`
	UpdatedAt  time.Time `json:"updated_at"`
}
