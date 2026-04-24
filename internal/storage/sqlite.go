package storage

import (
	"database/sql"
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"sort"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// SQLiteStorage gestiona la persistencia de los trozos de código.
type SQLiteStorage struct {
	db     *sql.DB
	dbPath string
}

// IndexStats resume el estado del índice del proyecto activo.
type IndexStats struct {
	DatabasePath    string `json:"database_path"`
	DatabaseSize    int64  `json:"database_size_bytes"`
	ChunkCount      int64  `json:"chunk_count"`
	FileCount       int64  `json:"file_count"`
	EmbeddedCount   int64  `json:"embedded_count"`
	UnembeddedCount int64  `json:"unembedded_count"`
	Provider        string `json:"provider,omitempty"`
	Model           string `json:"model,omitempty"`
	Dimensions      int    `json:"dimensions,omitempty"`
}

// NewSQLiteStorage inicializa la base de datos utilizando el ProjectManager para determinar la ruta.
func NewSQLiteStorage(pm *ProjectManager) (*SQLiteStorage, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current working directory: %w", err)
	}

	return NewSQLiteStorageForProject(pm, currentDir)
}

	// NewSQLiteStorageForProject inicializa la base de datos para un proyecto explícito.
func NewSQLiteStorageForProject(pm *ProjectManager, projectDir string) (*SQLiteStorage, error) {
	_, err := pm.EnsureProjectDir(projectDir)
	if err != nil {
		return nil, fmt.Errorf("failed to ensure project directory: %w", err)
	}

	dbPath, err := pm.GetDatabasePath(projectDir)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve database path: %w", err)
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	storage := &SQLiteStorage{db: db, dbPath: dbPath}

	if err := storage.migrate(); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	return storage, nil
}

// NewSQLiteStorageForProjectName inicializa la base de datos para un nombre lógico de proyecto.
func NewSQLiteStorageForProjectName(pm *ProjectManager, projectName string) (*SQLiteStorage, error) {
	_, err := pm.EnsureProjectDirForName(projectName)
	if err != nil {
		return nil, fmt.Errorf("failed to ensure project directory: %w", err)
	}

	dbPath, err := pm.GetDatabasePathForName(projectName)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve database path: %w", err)
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	storage := &SQLiteStorage{db: db, dbPath: dbPath}

	if err := storage.migrate(); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	return storage, nil
}

// migrate crea las tablas necesarias si no existen.
func (s *SQLiteStorage) migrate() error {
	query := `
	CREATE TABLE IF NOT EXISTS code_chunks (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		file_path TEXT NOT NULL,
		content TEXT NOT NULL,
		start_line INTEGER NOT NULL,
		end_line INTEGER NOT NULL,
		language TEXT,
		category TEXT,
		created_at DATETIME NOT NULL,
		embedding BLOB
	);
	CREATE TABLE IF NOT EXISTS index_metadata (
		id INTEGER PRIMARY KEY CHECK (id = 1),
		provider TEXT NOT NULL,
		model TEXT NOT NULL,
		dimensions INTEGER NOT NULL,
		updated_at DATETIME NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_file_path ON code_chunks(file_path);
	`
	_, err := s.db.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to migrate database: %w", err)
	}
	_, err = s.db.Exec(`ALTER TABLE code_chunks ADD COLUMN category TEXT`)
	if err != nil && !strings.Contains(err.Error(), "duplicate column name") {
		return fmt.Errorf("failed to add category column: %w", err)
	}

	return nil
}

// SaveChunk guarda un trozo de código y su vector en la base de datos.
func (s *SQLiteStorage) SaveChunk(chunk CodeChunk) (int64, error) {
	var vectorBlob []byte
	if len(chunk.Vector) > 0 {
		// Convertimos []float32 a []byte para guardarlo como BLOB
		buf := make([]byte, len(chunk.Vector)*4)
		for i, f := range chunk.Vector {
			binary.LittleEndian.PutUint32(buf[i*4:], math.Float32bits(f))
		}
		vectorBlob = buf
	}

	query := `
	INSERT INTO code_chunks (file_path, content, start_line, end_line, language, category, created_at, embedding)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`
	res, err := s.db.Exec(query,
		chunk.FilePath,
		chunk.Content,
		chunk.StartLine,
		chunk.EndLine,
		chunk.Language,
		chunk.Category,
		time.Now(),
		vectorBlob,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to save chunk: %w", err)
	}

	return res.LastInsertId()
}

// DeleteChunksByPath elimina chunks previos de un archivo para evitar duplicados en reindexados.
func (s *SQLiteStorage) DeleteChunksByPath(filePath string) error {
	_, err := s.db.Exec(`DELETE FROM code_chunks WHERE file_path = ?`, filePath)
	if err != nil {
		return fmt.Errorf("failed to delete chunks by path: %w", err)
	}
	return nil
}

// DeleteChunksByPathPrefix elimina chunks de todos los archivos bajo un prefijo de ruta.
func (s *SQLiteStorage) DeleteChunksByPathPrefix(pathPrefix string) error {
	_, err := s.db.Exec(`DELETE FROM code_chunks WHERE file_path = ? OR file_path LIKE ?`, pathPrefix, pathPrefix+"/%")
	if err != nil {
		return fmt.Errorf("failed to delete chunks by path prefix: %w", err)
	}
	return nil
}

// SearchText realiza una búsqueda de texto simple (fallback/keyword search).
func (s *SQLiteStorage) SearchText(query string) ([]CodeChunk, error) {
	sqlQuery := `SELECT id, file_path, content, start_line, end_line, language, category, created_at 
	             FROM code_chunks 
	             WHERE content LIKE ?`

	rows, err := s.db.Query(sqlQuery, "%"+query+"%")
	if err != nil {
		return nil, fmt.Errorf("search query failed: %w", err)
	}
	defer rows.Close()

	var results []CodeChunk
	for rows.Next() {
		var c CodeChunk
		err := rows.Scan(&c.ID, &c.FilePath, &c.Content, &c.StartLine, &c.EndLine, &c.Language, &c.Category, &c.CreatedAt)
		if err != nil {
			return nil, err
		}
		results = append(results, c)
	}
	return results, nil
}

// Stats devuelve un resumen del índice del proyecto activo.
func (s *SQLiteStorage) Stats() (IndexStats, error) {
	stats := IndexStats{DatabasePath: s.dbPath}

	if info, err := os.Stat(s.dbPath); err == nil {
		stats.DatabaseSize = info.Size()
	}

	row := s.db.QueryRow(`
		SELECT
			COUNT(*) as chunk_count,
			COUNT(DISTINCT file_path) as file_count,
			SUM(CASE WHEN embedding IS NOT NULL AND length(embedding) > 0 THEN 1 ELSE 0 END) as embedded_count,
			SUM(CASE WHEN embedding IS NULL OR length(embedding) = 0 THEN 1 ELSE 0 END) as unembedded_count
		FROM code_chunks
	`)

	var embedded sql.NullInt64
	var unembedded sql.NullInt64
	if err := row.Scan(&stats.ChunkCount, &stats.FileCount, &embedded, &unembedded); err != nil {
		return IndexStats{}, fmt.Errorf("failed to query index stats: %w", err)
	}

	if embedded.Valid {
		stats.EmbeddedCount = embedded.Int64
	}
	if unembedded.Valid {
		stats.UnembeddedCount = unembedded.Int64
	}

	if metadata, err := s.GetIndexMetadata(); err == nil {
		stats.Provider = metadata.Provider
		stats.Model = metadata.Model
		stats.Dimensions = metadata.Dimensions
	}

	return stats, nil
}

func (s *SQLiteStorage) SetIndexMetadata(metadata IndexMetadata) error {
	_, err := s.db.Exec(`
		INSERT INTO index_metadata (id, provider, model, dimensions, updated_at)
		VALUES (1, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			provider = excluded.provider,
			model = excluded.model,
			dimensions = excluded.dimensions,
			updated_at = excluded.updated_at
	`, metadata.Provider, metadata.Model, metadata.Dimensions, time.Now())
	if err != nil {
		return fmt.Errorf("failed to set index metadata: %w", err)
	}
	return nil
}

func (s *SQLiteStorage) GetIndexMetadata() (IndexMetadata, error) {
	row := s.db.QueryRow(`SELECT provider, model, dimensions, updated_at FROM index_metadata WHERE id = 1`)
	var metadata IndexMetadata
	if err := row.Scan(&metadata.Provider, &metadata.Model, &metadata.Dimensions, &metadata.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return IndexMetadata{}, err
		}
		return IndexMetadata{}, fmt.Errorf("failed to query index metadata: %w", err)
	}
	return metadata, nil
}

func (s *SQLiteStorage) RequiresReindex(provider, model string, dimensions int) (bool, error) {
	metadata, err := s.GetIndexMetadata()
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}

	return metadata.Provider != provider || metadata.Model != model || metadata.Dimensions != dimensions, nil
}

// SearchSemantic realiza una búsqueda por similitud coseno sobre embeddings guardados.
func (s *SQLiteStorage) SearchSemantic(queryVector []float32, limit int) ([]CodeChunk, error) {
	if len(queryVector) == 0 {
		return nil, nil
	}

	sqlQuery := `SELECT id, file_path, content, start_line, end_line, language, category, created_at, embedding FROM code_chunks WHERE embedding IS NOT NULL AND length(embedding) > 0`
	rows, err := s.db.Query(sqlQuery)
	if err != nil {
		return nil, fmt.Errorf("semantic search query failed: %w", err)
	}
	defer rows.Close()

	var results []CodeChunk
	for rows.Next() {
		var c CodeChunk
		var embeddingBlob []byte
		if err := rows.Scan(&c.ID, &c.FilePath, &c.Content, &c.StartLine, &c.EndLine, &c.Language, &c.Category, &c.CreatedAt, &embeddingBlob); err != nil {
			return nil, err
		}

		vector := decodeVector(embeddingBlob)
		if len(vector) == 0 || len(vector) != len(queryVector) {
			continue
		}

		c.Score = cosineSimilarity(queryVector, vector)
		results = append(results, c)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}

	return results, nil
}

func decodeVector(blob []byte) []float32 {
	if len(blob) == 0 || len(blob)%4 != 0 {
		return nil
	}

	vector := make([]float32, len(blob)/4)
	for i := range vector {
		bits := binary.LittleEndian.Uint32(blob[i*4:])
		vector[i] = math.Float32frombits(bits)
	}

	return vector
}

func cosineSimilarity(a, b []float32) float64 {
	if len(a) == 0 || len(a) != len(b) {
		return 0
	}

	var dot, normA, normB float64
	for i := range a {
		av := float64(a[i])
		bv := float64(b[i])
		dot += av * bv
		normA += av * av
		normB += bv * bv
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}

// Close cierra la conexión a la base de datos.
func (s *SQLiteStorage) Close() error {
	return s.db.Close()
}
