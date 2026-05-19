package storage

import "time"

type CodeChunk struct {
	ID        int64     `json:"id"`
	FilePath  string    `json:"file_path"`
	Content   string    `json:"content"`
	StartLine int       `json:"start_line"`
	EndLine   int       `json:"end_line"`
	Language  string    `json:"language"`
	Category  string    `json:"category,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	Score     float64   `json:"score,omitempty"` // Cosine similarity (vector search) or BM25 score (keyword search), depending on source
	Signature string    `json:"signature,omitempty"`
	Purpose   string    `json:"purpose,omitempty"`

	Vector []float32 `json:"-"`
}

type IndexMetadata struct {
	Provider         string    `json:"provider"`
	Model            string    `json:"model"`
	Dimensions       int       `json:"dimensions"`
	IndexFingerprint string    `json:"index_fingerprint,omitempty"`
	UpdatedAt        time.Time `json:"updated_at"`
}
