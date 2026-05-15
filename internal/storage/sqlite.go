package storage

import (
	"crypto/sha256"
	"database/sql"
	"encoding/binary"
	"fmt"
	"log"
	"math"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"vectos/internal/vectorindex"
)

type embeddingRow struct {
	ID      int
	Vector  []float32
}

// SQLiteStorage gestiona la persistencia de los trozos de código.
type SQLiteStorage struct {
	db        *sql.DB
	dbPath    string
	vectIdx   *vectorindex.HNSW
	vectIdxMu sync.RWMutex
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

// NewSQLiteStorageForDocsProjectName inicializa la base de datos de documentación para un nombre lógico de proyecto.
func NewSQLiteStorageForDocsProjectName(pm *ProjectManager, projectName string) (*SQLiteStorage, error) {
	_, err := pm.EnsureProjectDirForName(projectName)
	if err != nil {
		return nil, fmt.Errorf("failed to ensure project directory: %w", err)
	}

	dbPath, err := pm.GetDatabasePathForName(projectName, "-docs")
	if err != nil {
		return nil, fmt.Errorf("failed to resolve docs database path: %w", err)
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open docs database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping docs database: %w", err)
	}

	storage := &SQLiteStorage{db: db, dbPath: dbPath}

	if err := storage.migrate(); err != nil {
		return nil, fmt.Errorf("failed to migrate docs database: %w", err)
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
	_, err = s.db.Exec(`ALTER TABLE code_chunks ADD COLUMN signature TEXT`)
	if err != nil && !strings.Contains(err.Error(), "duplicate column name") {
		return fmt.Errorf("failed to add signature column: %w", err)
	}
	_, err = s.db.Exec(`ALTER TABLE code_chunks ADD COLUMN purpose TEXT`)
	if err != nil && !strings.Contains(err.Error(), "duplicate column name") {
		return fmt.Errorf("failed to add purpose column: %w", err)
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
	INSERT INTO code_chunks (file_path, content, start_line, end_line, language, category, created_at, embedding, signature, purpose)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
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
		chunk.Signature,
		chunk.Purpose,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to save chunk: %w", err)
	}

	return res.LastInsertId()
}

// GetAllEmbeddings returns all chunk IDs with their embedding vectors.
func (s *SQLiteStorage) GetAllEmbeddings() (map[int][]float32, error) {
	rows, err := s.db.Query(`SELECT id, embedding FROM code_chunks WHERE embedding IS NOT NULL AND length(embedding) > 0 ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("failed to query embeddings: %w", err)
	}
	defer rows.Close()

	out := make(map[int][]float32)
	for rows.Next() {
		var id int
		var blob []byte
		if err := rows.Scan(&id, &blob); err != nil {
			return nil, fmt.Errorf("failed to scan embedding: %w", err)
		}
		out[id] = decodeVector(blob)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate embeddings: %w", err)
	}
	return out, nil
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

// DeleteAllChunks elimina todos los chunks del índice activo, preservando el metadata del índice.
func (s *SQLiteStorage) DeleteAllChunks() error {
	_, err := s.db.Exec(`DELETE FROM code_chunks`)
	if err != nil {
		return fmt.Errorf("failed to delete all chunks: %w", err)
	}
	return nil
}

// escapeLikeTerm escapes SQL LIKE wildcard characters (% and _) and the
// default escape character (\) in user-provided search terms to prevent
// wildcard injection attacks.
func escapeLikeTerm(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch r {
		case '\\':
			b.WriteString(`\\`)
		case '%':
			b.WriteString(`\%`)
		case '_':
			b.WriteString(`\_`)
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// SearchText performs a simple text search (fallback/keyword search).
func (s *SQLiteStorage) SearchText(query string) ([]CodeChunk, error) {
	sqlQuery := `SELECT id, file_path, content, start_line, end_line, language, category, created_at, signature, purpose
	             FROM code_chunks
	             WHERE content LIKE ?`

	rows, err := s.db.Query(sqlQuery, "%"+escapeLikeTerm(query)+"%")
	if err != nil {
		return nil, fmt.Errorf("search query failed: %w", err)
	}
	defer rows.Close()

	var results []CodeChunk
	for rows.Next() {
		var c CodeChunk
		var category sql.NullString
		var signature sql.NullString
		var purpose sql.NullString
		err := rows.Scan(&c.ID, &c.FilePath, &c.Content, &c.StartLine, &c.EndLine, &c.Language, &category, &c.CreatedAt, &signature, &purpose)
		if err != nil {
			return nil, err
		}
		c.Category = categoryOrDefault(category, c.Language)
		c.Signature = signature.String
		c.Purpose = purpose.String
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

// SearchSemantic performs cosine similarity search over stored embeddings.
// When a vector index is loaded, it uses HNSW approximate nearest neighbor
// search. Otherwise it falls back to a full-table cosine scan with a warning.
func (s *SQLiteStorage) SearchSemantic(queryVector []float32, limit int, includeDocs bool) ([]CodeChunk, error) {
	if len(queryVector) == 0 {
		return nil, nil
	}

	// Use vector index when available.
	s.vectIdxMu.RLock()
	idx := s.vectIdx
	s.vectIdxMu.RUnlock()

	if idx != nil {
		return s.searchViaIndex(idx, queryVector, limit, includeDocs)
	}

	log.Println("vectorindex: no index loaded — falling back to linear scan")
	return s.searchLinearScan(queryVector, limit, includeDocs)
}

// searchViaIndex uses the HNSW index for approximate nearest neighbor search,
// then fetches full chunk rows from SQLite.
func (s *SQLiteStorage) searchViaIndex(idx *vectorindex.HNSW, queryVector []float32, limit int, includeDocs bool) ([]CodeChunk, error) {
	// HNSW SearchScored returns IDs with cosine distances.
	scored := idx.SearchScored(queryVector, limit)
	if len(scored) == 0 {
		return nil, nil
	}

	// Collect IDs and precompute scores (cosine similarity = 1 - distance).
	ids := make([]int, len(scored))
	scoreByID := make(map[int]float64, len(scored))
	for i, s := range scored {
		ids[i] = s.ID
		scoreByID[s.ID] = 1.0 - s.Distance
	}

	chunks, err := s.GetChunksByIDs(ids)
	if err != nil {
		return nil, err
	}

	results := make([]CodeChunk, 0, len(chunks))
	for _, c := range chunks {
		if !includeDocs && (c.Category == "docs" || c.Category == "dependency_metadata") {
			continue
		}
		c.Score = scoreByID[int(c.ID)]
		results = append(results, c)
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}

	return results, nil
}

// searchLinearScan performs a full-table cosine similarity scan (legacy path).
func (s *SQLiteStorage) searchLinearScan(queryVector []float32, limit int, includeDocs bool) ([]CodeChunk, error) {
	sqlQuery := `SELECT id, file_path, content, start_line, end_line, language, category, created_at, embedding, signature, purpose FROM code_chunks WHERE embedding IS NOT NULL AND length(embedding) > 0`
	if !includeDocs {
		sqlQuery += ` AND (category IS NULL OR category NOT IN ('docs', 'dependency_metadata'))`
	}
	rows, err := s.db.Query(sqlQuery)
	if err != nil {
		return nil, fmt.Errorf("semantic search query failed: %w", err)
	}
	defer rows.Close()

	var results []CodeChunk
	for rows.Next() {
		var c CodeChunk
		var category sql.NullString
		var signature sql.NullString
		var purpose sql.NullString
		var embeddingBlob []byte
		if err := rows.Scan(&c.ID, &c.FilePath, &c.Content, &c.StartLine, &c.EndLine, &c.Language, &category, &c.CreatedAt, &embeddingBlob, &signature, &purpose); err != nil {
			return nil, err
		}
		c.Category = categoryOrDefault(category, c.Language)
		c.Signature = signature.String
		c.Purpose = purpose.String

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

func categoryOrDefault(category sql.NullString, language string) string {
	if category.Valid && strings.TrimSpace(category.String) != "" {
		return category.String
	}

	switch {
	case language == "dockerfile", strings.HasPrefix(language, "yaml"), strings.HasPrefix(language, "bazel"):
		return "infra_config"
	default:
		return "source"
	}
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

// ChunkTableContentHash returns a SHA-256 hash of the chunk table state.
// It hashes the row count plus each chunk's ID and last update timestamp.
// If the chunk table is empty, it returns a zero hash.
func (s *SQLiteStorage) ChunkTableContentHash() ([sha256.Size]byte, error) {
	h := sha256.New()

	var count int64
	if err := s.db.QueryRow("SELECT COUNT(*) FROM code_chunks").Scan(&count); err != nil {
		return [sha256.Size]byte{}, fmt.Errorf("chunk content hash: count: %w", err)
	}
	binary.Write(h, binary.LittleEndian, count)
	if count == 0 {
		return [sha256.Size]byte{}, nil
	}

	rows, err := s.db.Query("SELECT id, created_at FROM code_chunks ORDER BY id")
	if err != nil {
		return [sha256.Size]byte{}, fmt.Errorf("chunk content hash: query: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id int64
		var t time.Time
		if err := rows.Scan(&id, &t); err != nil {
			return [sha256.Size]byte{}, fmt.Errorf("chunk content hash: scan: %w", err)
		}
		binary.Write(h, binary.LittleEndian, id)
		binary.Write(h, binary.LittleEndian, t.UnixNano())
	}
	if err := rows.Err(); err != nil {
		return [sha256.Size]byte{}, fmt.Errorf("chunk content hash: rows: %w", err)
	}

	var out [sha256.Size]byte
	h.Sum(out[:0])
	return out, nil
}

// SetVectorIndex sets the in-memory HNSW vector index for semantic search.
func (s *SQLiteStorage) SetVectorIndex(idx *vectorindex.HNSW) {
	s.vectIdxMu.Lock()
	defer s.vectIdxMu.Unlock()
	s.vectIdx = idx
}

// HasVectorIndex reports whether a vector index is loaded.
func (s *SQLiteStorage) HasVectorIndex() bool {
	s.vectIdxMu.RLock()
	defer s.vectIdxMu.RUnlock()
	return s.vectIdx != nil
}

// VectorIndexPath returns the file path for the .vectorindex file alongside
// the SQLite database.
func (s *SQLiteStorage) VectorIndexPath() string {
	return s.dbPath + ".vectorindex"
}

// LoadVectorIndex attempts to load the HNSW index from disk and sets it
// on this storage. It returns the loaded index and content hash, or an error
// if the file cannot be loaded.
// Note: the caller is responsible for validating the content hash against
// the current chunk table state.
func (s *SQLiteStorage) LoadVectorIndex() (*vectorindex.HNSW, [sha256.Size]byte, string, *vectorindex.SQ8Params, error) {
	idx, hash, compression, params, err := vectorindex.LoadIndex(s.VectorIndexPath())
	if err != nil {
		return nil, [sha256.Size]byte{}, "", nil, err
	}
	s.SetVectorIndex(idx)
	return idx, hash, compression, params, nil
}

// GetChunksByIDs fetches chunks by their primary key IDs. Used to resolve
// HNSW search results (node IDs) back to full chunk rows.
func (s *SQLiteStorage) GetChunksByIDs(ids []int) ([]CodeChunk, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	// Build IN clause placeholders.
	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		placeholders[i] = "?"
		args[i] = id
	}

	query := fmt.Sprintf(
		`SELECT id, file_path, content, start_line, end_line, language, category, created_at, signature, purpose
		 FROM code_chunks WHERE id IN (%s)`,
		strings.Join(placeholders, ","),
	)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("get chunks by ids: %w", err)
	}
	defer rows.Close()

	var results []CodeChunk
	for rows.Next() {
		var c CodeChunk
		var category sql.NullString
		var signature sql.NullString
		var purpose sql.NullString
		if err := rows.Scan(&c.ID, &c.FilePath, &c.Content, &c.StartLine, &c.EndLine, &c.Language, &category, &c.CreatedAt, &signature, &purpose); err != nil {
			return nil, err
		}
		c.Category = categoryOrDefault(category, c.Language)
		c.Signature = signature.String
		c.Purpose = purpose.String
		results = append(results, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Sort to match input ID order (for score assignment below).
	order := make(map[int64]int, len(ids))
	for i, id := range ids {
		order[int64(id)] = i
	}
	sort.Slice(results, func(i, j int) bool {
		return order[results[i].ID] < order[results[j].ID]
	})

	return results, nil
}
