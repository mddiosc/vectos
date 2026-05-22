package storage

import (
	"crypto/sha256"
	"database/sql"
	"encoding/binary"
	"fmt"
	"log"
	"math"
	"net/url"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"

	_ "github.com/mattn/go-sqlite3"
	"vectos/internal/vectorindex"
)

// SQLiteStorage gestiona la persistencia de los trozos de código.
type SQLiteStorage struct {
	db        *sql.DB
	dbPath    string
	vectIdx   *vectorindex.HNSW
	vectIdxMu sync.RWMutex

	// HNSW build parameters for lazy/auto-rebuild when the vector index
	// goes stale (e.g. after incremental chunk updates change the content hash).
	viM              int
	viEfConstruction int
	viEfSearch       int
	rebuildGroup     singleflight.Group // deduplicates concurrent rebuild calls
}

const maxGetAllEmbeddingsBytes = 128 << 20

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

	db, err := openSQLite(dbPath)
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

	db, err := openSQLite(dbPath)
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

	db, err := openSQLite(dbPath)
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

func openSQLite(dbPath string) (*sql.DB, error) {
	values := url.Values{}
	values.Set("_journal_mode", "WAL")
	values.Set("_busy_timeout", "5000")
	dsn := (&url.URL{Scheme: "file", Path: dbPath, RawQuery: values.Encode()}).String()
	return sql.Open("sqlite3", dsn)
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
		index_fingerprint TEXT,
		updated_at DATETIME NOT NULL
	);
	CREATE TABLE IF NOT EXISTS indexed_files (
		path TEXT PRIMARY KEY,
		hash TEXT NOT NULL,
		indexed_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
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
	_, err = s.db.Exec(`ALTER TABLE index_metadata ADD COLUMN index_fingerprint TEXT`)
	if err != nil && !strings.Contains(err.Error(), "duplicate column name") {
		return fmt.Errorf("failed to add index_fingerprint column: %w", err)
	}

	return nil
}

// UpsertIndexedFile inserts or updates the hash for a file path.
func (s *SQLiteStorage) UpsertIndexedFile(path, hash string) error {
	_, err := s.db.Exec(`
		INSERT OR REPLACE INTO indexed_files (path, hash, indexed_at)
		VALUES (?, ?, CURRENT_TIMESTAMP)
	`, path, hash)
	if err != nil {
		return fmt.Errorf("failed to upsert indexed file: %w", err)
	}
	return nil
}

// GetIndexedFileHash returns the stored hash for a file, or empty string if not found.
func (s *SQLiteStorage) GetIndexedFileHash(path string) (string, error) {
	row := s.db.QueryRow(`SELECT hash FROM indexed_files WHERE path = ?`, path)
	var hash string
	if err := row.Scan(&hash); err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", fmt.Errorf("failed to get indexed file hash: %w", err)
	}
	return hash, nil
}

// DeleteIndexedFile removes the hash record for a file path.
func (s *SQLiteStorage) DeleteIndexedFile(path string) error {
	_, err := s.db.Exec(`DELETE FROM indexed_files WHERE path = ?`, path)
	if err != nil {
		return fmt.Errorf("failed to delete indexed file: %w", err)
	}
	return nil
}

// HasFileChanged compares the current file hash against the stored hash.
func (s *SQLiteStorage) HasFileChanged(path string, currentHash string) (bool, error) {
	storedHash, err := s.GetIndexedFileHash(path)
	if err != nil {
		return false, err
	}
	if storedHash == "" {
		return true, nil
	}
	return storedHash != currentHash, nil
}

// RemoveDeletedFile cleans up all chunks and the hash record for a deleted file.
func (s *SQLiteStorage) RemoveDeletedFile(path string) error {
	if err := s.DeleteChunksByPath(path); err != nil {
		return err
	}
	return s.DeleteIndexedFile(path)
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

	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	return id, nil
}

// GetAllEmbeddings returns all chunk IDs with their embedding vectors.
func (s *SQLiteStorage) GetAllEmbeddings() (map[int][]float32, error) {
	totalBytes, err := s.totalEmbeddingBytes()
	if err != nil {
		return nil, err
	}
	if totalBytes > maxGetAllEmbeddingsBytes {
		return nil, fmt.Errorf("refusing to load %d bytes of embeddings into memory; use streaming iteration instead", totalBytes)
	}

	out := make(map[int][]float32)
	if err := s.ForEachEmbedding(func(id int, vector []float32) error {
		out[id] = vector
		return nil
	}); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *SQLiteStorage) totalEmbeddingBytes() (int64, error) {
	var total sql.NullInt64
	if err := s.db.QueryRow(`SELECT COALESCE(SUM(length(embedding)), 0) FROM code_chunks WHERE embedding IS NOT NULL AND length(embedding) > 0`).Scan(&total); err != nil {
		return 0, fmt.Errorf("failed to measure embedding footprint: %w", err)
	}
	if !total.Valid {
		return 0, nil
	}
	return total.Int64, nil
}

// ForEachEmbedding streams embeddings without materializing the full dataset.
func (s *SQLiteStorage) ForEachEmbedding(fn func(id int, vector []float32) error) error {
	rows, err := s.db.Query(`SELECT id, embedding FROM code_chunks WHERE embedding IS NOT NULL AND length(embedding) > 0 ORDER BY id`)
	if err != nil {
		return fmt.Errorf("failed to query embeddings: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id int
		var blob []byte
		if err := rows.Scan(&id, &blob); err != nil {
			return fmt.Errorf("failed to scan embedding: %w", err)
		}
		if err := fn(id, decodeVector(blob)); err != nil {
			return err
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("failed to iterate embeddings: %w", err)
	}
	return nil
}

// DeleteChunksByPath elimina chunks previos de un archivo para evitar duplicados en reindexados.
func (s *SQLiteStorage) DeleteChunksByPath(filePath string) error {
	_, err := s.db.Exec(`DELETE FROM code_chunks WHERE file_path = ?`, filePath)
	if err != nil {
		return err
	}
	return nil
}

// DeleteChunksByPathPrefix elimina chunks de todos los archivos bajo un prefijo de ruta.
func (s *SQLiteStorage) DeleteChunksByPathPrefix(pathPrefix string) error {
	_, err := s.db.Exec(`DELETE FROM code_chunks WHERE file_path = ? OR file_path LIKE ?`, pathPrefix, pathPrefix+"/%")
	return err
}

// DeleteAllChunks elimina todos los chunks del índice activo, preservando el metadata del índice.
func (s *SQLiteStorage) DeleteAllChunks() error {
	_, err := s.db.Exec(`DELETE FROM code_chunks`)
	if err != nil {
		return fmt.Errorf("failed to delete all chunks: %w", err)
	}
	return nil
}

// InvalidateEmbeddings clears all embedding vectors and file hashes, forcing
// a full re-embedding on the next index run. Chunk text/metadata is preserved.
// This should be called when the embedding model changes to avoid mixed
// dimensions in the database.
func (s *SQLiteStorage) InvalidateEmbeddings() error {
	if _, err := s.db.Exec(`UPDATE code_chunks SET embedding = NULL`); err != nil {
		return fmt.Errorf("failed to clear embeddings: %w", err)
	}
	if _, err := s.db.Exec(`DELETE FROM indexed_files`); err != nil {
		return fmt.Errorf("failed to clear indexed file hashes: %w", err)
	}
	// Discard the in-memory vector index since it's now stale.
	s.vectIdxMu.Lock()
	s.vectIdx = nil
	s.vectIdxMu.Unlock()
	return nil
}

// ClearIndexedData removes persisted chunks and file hashes so the next index
// run performs a full rebuild from source files.
func (s *SQLiteStorage) ClearIndexedData() error {
	if _, err := s.db.Exec(`DELETE FROM code_chunks`); err != nil {
		return fmt.Errorf("failed to clear code chunks: %w", err)
	}
	if _, err := s.db.Exec(`DELETE FROM indexed_files`); err != nil {
		return fmt.Errorf("failed to clear indexed file hashes: %w", err)
	}
	s.vectIdxMu.Lock()
	s.vectIdx = nil
	s.vectIdxMu.Unlock()
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

// SearchTextRanked performs a ranked keyword search returning results with scores.
// Uses term-frequency scoring in Go (no FTS5 dependency). Results are ordered by score descending.
func (s *SQLiteStorage) SearchTextRanked(query string, limit int) ([]CodeChunk, error) {
	if limit <= 0 {
		limit = 25
	}

	// Split query into terms and build OR-connected LIKE clauses so
	// multi-word queries match documents containing any of the terms.
	terms := meaningfulTerms(query)
	if len(terms) == 0 {
		return nil, nil
	}

	clauses := make([]string, 0, len(terms)+1)
	args := make([]interface{}, 0, len(terms)+2)
	// Always include full phrase match
	clauses = append(clauses, "content LIKE ?")
	args = append(args, "%"+escapeLikeTerm(query)+"%")
	for _, term := range terms {
		clauses = append(clauses, "content LIKE ?")
		args = append(args, "%"+escapeLikeTerm(term)+"%")
	}
	args = append(args, limit*5)

	sqlQuery := fmt.Sprintf(`SELECT id, file_path, content, start_line, end_line, language, category, created_at, signature, purpose
	             FROM code_chunks
	             WHERE %s
	             LIMIT ?`, strings.Join(clauses, " OR "))

	rows, err := s.db.Query(sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("keyword search failed: %w", err)
	}
	defer rows.Close()

	var candidates []CodeChunk
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
		c.Score = computeKeywordScore(c.Content, c.FilePath, query)
		candidates = append(candidates, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Sort by score descending
	sort.SliceStable(candidates, func(i, j int) bool {
		return candidates[i].Score > candidates[j].Score
	})

	// Limit to top results
	if len(candidates) > limit {
		candidates = candidates[:limit]
	}
	return candidates, nil
}

// meaningfulTerms splits a query into significant terms (>=2 chars, not stopwords).
func meaningfulTerms(query string) []string {
	words := strings.Fields(strings.ToLower(query))
	terms := make([]string, 0, len(words))
	for _, w := range words {
		if len(w) < 2 {
			continue
		}
		if isStopWord(w) {
			continue
		}
		terms = append(terms, w)
	}
	return terms
}

func isStopWord(w string) bool {
	switch w {
	case "the", "and", "for", "with", "that", "this", "from", "into", "only", "part", "flow", "code", "how", "does", "what", "where", "when", "why":
		return true
	}
	return false
}

// computeKeywordScore calculates a relevance score for a chunk given a query.
// Uses term frequency in content + filename match bonus.
func computeKeywordScore(content, filePath, query string) float64 {
	contentLower := strings.ToLower(content)
	// Extract base filename without importing path/filepath
	base := filePath
	if idx := strings.LastIndex(filePath, "/"); idx >= 0 {
		base = filePath[idx+1:]
	}
	pathLower := strings.ToLower(base)
	queryLower := strings.ToLower(query)

	score := 0.0

	// Exact phrase match gets highest bonus
	if count := strings.Count(contentLower, queryLower); count > 0 {
		score += float64(count) * 3.0
	}

	// Per-term matches
	for _, term := range strings.Fields(queryLower) {
		if len(term) < 2 {
			continue
		}
		if count := strings.Count(contentLower, term); count > 0 {
			score += float64(count) * 1.0
		}
		// Filename match bonus
		if strings.Contains(pathLower, term) {
			score += 0.5
		}
	}

	// Config/lock files get reduced keyword weight — they contain
	// code-related words as configuration values, not as implementations.
	if isKeywordNoiseFile(base) {
		score *= 0.3
	}

	// Documentation files get a relevance boost — actual project docs
	// should rank above blog posts and other non-documentation content.
	if isDocFilePath(filePath) {
		score *= 1.5
	}

	return score
}

// isKeywordNoiseFile returns true for config files, lockfiles, and other
// files whose keyword matches are usually misleading noise.
func isKeywordNoiseFile(filename string) bool {
	lower := strings.ToLower(filename)
	for _, pattern := range keywordNoisePatterns {
		if strings.Contains(lower, pattern) {
			return true
		}
	}
	return false
}

// isDocFilePath returns true for documentation files that should get
// relevance boosts: files under a project docs/ directory (not arbitrary
// substrings like "vendor/somelib/docs-helper/") and README.md files.
//
// The match is intentionally strict:
//   - "docs/" must appear as a path segment, not as a substring;
//   - the file must be a documentation-shaped extension (.md, .mdx, .rst,
//     .txt) so we don't boost arbitrary files that happen to live under a
//     docs/ folder (images, scripts, etc.);
//   - README.md (any case) at any depth is always considered a doc.
func isDocFilePath(filePath string) bool {
	// Normalize to forward slashes for consistent matching across OSes.
	p := strings.ReplaceAll(filePath, "\\", "/")

	base := p
	if idx := strings.LastIndex(p, "/"); idx >= 0 {
		base = p[idx+1:]
	}

	// README.md at any depth is always a doc.
	if strings.EqualFold(base, "readme.md") {
		return true
	}

	if !isDocExtension(base) {
		return false
	}

	// Require "docs" to appear as a real path segment.
	for _, segment := range strings.Split(p, "/") {
		if segment == "docs" {
			return true
		}
	}
	return false
}

// isDocExtension reports whether the file's extension is one we treat as
// documentation content.
func isDocExtension(base string) bool {
	lower := strings.ToLower(base)
	switch {
	case strings.HasSuffix(lower, ".md"),
		strings.HasSuffix(lower, ".mdx"),
		strings.HasSuffix(lower, ".rst"),
		strings.HasSuffix(lower, ".txt"):
		return true
	}
	return false
}

var keywordNoisePatterns = []string{
	"eslint.config",
	"tailwind.config",
	"vite.config",
	"vitest.config",
	"playwright.config",
	"tsconfig",
	"postcss.config",
	".lock",
	"package.json",
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
		INSERT INTO index_metadata (id, provider, model, dimensions, index_fingerprint, updated_at)
		VALUES (1, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			provider = excluded.provider,
			model = excluded.model,
			dimensions = excluded.dimensions,
			index_fingerprint = excluded.index_fingerprint,
			updated_at = excluded.updated_at
	`, metadata.Provider, metadata.Model, metadata.Dimensions, metadata.IndexFingerprint, time.Now())
	if err != nil {
		return fmt.Errorf("failed to set index metadata: %w", err)
	}
	return nil
}

func (s *SQLiteStorage) GetIndexMetadata() (IndexMetadata, error) {
	row := s.db.QueryRow(`SELECT provider, model, dimensions, COALESCE(index_fingerprint, ''), updated_at FROM index_metadata WHERE id = 1`)
	var metadata IndexMetadata
	if err := row.Scan(&metadata.Provider, &metadata.Model, &metadata.Dimensions, &metadata.IndexFingerprint, &metadata.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return IndexMetadata{}, err
		}
		return IndexMetadata{}, fmt.Errorf("failed to query index metadata: %w", err)
	}
	return metadata, nil
}

func (s *SQLiteStorage) RequiresReindex(provider, model string, dimensions int, indexFingerprint string) (bool, error) {
	metadata, err := s.GetIndexMetadata()
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}

	return metadata.Provider != provider || metadata.Model != model || metadata.Dimensions != dimensions || metadata.IndexFingerprint != indexFingerprint, nil
}

// SearchSemantic performs cosine similarity search over stored embeddings.
// When a vector index is loaded, it uses HNSW approximate nearest neighbor
// search. Otherwise it falls back to a full-table cosine scan with a warning.
func (s *SQLiteStorage) SearchSemantic(queryVector []float32, limit int, includeDocs bool) ([]CodeChunk, error) {
	if len(queryVector) == 0 {
		return nil, nil
	}

	// Use vector index when available and chunk count is sufficient.
	s.vectIdxMu.RLock()
	idx := s.vectIdx
	s.vectIdxMu.RUnlock()

	if idx == nil {
		// Try loading the index from disk. If the content hash does not match
		// (chunks changed since the index was built), LoadVectorIndex will
		// clean up the stale index but return an error. In that case, attempt
		// an automatic rebuild from the embeddings already stored in the DB.
		if _, _, _, _, err := s.LoadVectorIndex(); err != nil {
			if rebuildErr := s.RebuildVectorIndex(); rebuildErr != nil {
				log.Printf("vectorindex: auto-rebuild failed: %v — falling back to linear scan", rebuildErr)
			}
		}
		s.vectIdxMu.RLock()
		idx = s.vectIdx
		s.vectIdxMu.RUnlock()
	}

	// For small indexes (< 1000 chunks), linear scan is faster and more accurate
	// than HNSW approximate search.
	if idx != nil && s.chunkCount() >= 1000 && idx.Dimension() == len(queryVector) {
		return s.searchViaIndex(idx, queryVector, limit, includeDocs)
	}

	if idx != nil && idx.Dimension() != len(queryVector) {
		log.Printf("vectorindex: dimension mismatch (index=%d, query=%d) — falling back to linear scan", idx.Dimension(), len(queryVector))
	}
	if idx == nil {
		log.Println("vectorindex: no index loaded — falling back to linear scan")
	}
	return s.searchLinearScan(queryVector, limit, includeDocs)
}

func (s *SQLiteStorage) chunkCount() int {
	var count int
	s.db.QueryRow("SELECT COUNT(*) FROM code_chunks WHERE embedding IS NOT NULL AND length(embedding) > 0").Scan(&count)
	return count
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
		if limit <= 0 {
			results = append(results, c)
			continue
		}
		if len(results) < limit {
			results = append(results, c)
			continue
		}
		minIdx := 0
		for i := 1; i < len(results); i++ {
			if results[i].Score < results[minIdx].Score {
				minIdx = i
			}
		}
		if c.Score > results[minIdx].Score {
			results[minIdx] = c
		}
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

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

// SetVectorIndexParams configures HNSW build parameters for auto-rebuild.
// Callers should invoke this after opening storage and before using
// SearchSemantic so that the storage can rebuild the index on its own
// if it becomes stale. Pass zero values to use internal defaults.
func (s *SQLiteStorage) SetVectorIndexParams(m, efConstruction, efSearch int) {
	s.viM = m
	s.viEfConstruction = efConstruction
	s.viEfSearch = efSearch
}

// RebuildVectorIndex reads all stored embeddings, constructs a new HNSW
// index, persists it to disk alongside the current chunk-table content hash,
// and sets the in-memory index on this storage. It is safe to call after
// chunk content changes as it only uses data already in the database.
// If no embeddings exist the method is a no-op.
// Concurrent callers are deduplicated via singleflight.
func (s *SQLiteStorage) RebuildVectorIndex() error {
	_, err, _ := s.rebuildGroup.Do("rebuild", func() (any, error) {
		return nil, s.rebuildVectorIndexLocked()
	})
	return err
}

func (s *SQLiteStorage) rebuildVectorIndexLocked() error {

	// Determine dominant dimension — same logic as buildVectorIndex in commands_index.go.
	dimCounts := make(map[int]int)
	if err := s.ForEachEmbedding(func(_ int, vector []float32) error {
		dimCounts[len(vector)]++
		return nil
	}); err != nil {
		return fmt.Errorf("vectorindex: count dimensions: %w", err)
	}
	if len(dimCounts) == 0 {
		return nil // no embeddings, nothing to build
	}
	var dimension, maxCount int
	for dim, c := range dimCounts {
		if c > maxCount {
			dimension = dim
			maxCount = c
		}
	}
	if dimension == 0 || maxCount == 0 {
		return nil
	}

	m := s.viM
	if m <= 0 { m = 16 }
	efCons := s.viEfConstruction
	if efCons <= 0 { efCons = 200 }
	efSearch := s.viEfSearch
	if efSearch <= 0 { efSearch = 200 }

	idx := vectorindex.NewHNSW(dimension, vectorindex.Config{M: m, EfConstruction: efCons, EfSearch: efSearch})
	inserted := 0
	if err := s.ForEachEmbedding(func(id int, vector []float32) error {
		if len(vector) != dimension {
			return nil
		}
		if err := idx.Insert(id, vector); err != nil {
			return err
		}
		inserted++
		return nil
	}); err != nil {
		return fmt.Errorf("vectorindex: build: %w", err)
	}

	if inserted == 0 {
		return nil
	}

	// Persist the rebuilt index with the current content hash.
	contentHash, err := s.ChunkTableContentHash()
	if err != nil {
		return fmt.Errorf("vectorindex: content hash for rebuild: %w", err)
	}
	if err := idx.Save(s.VectorIndexPath(), contentHash, "none", nil); err != nil {
		return fmt.Errorf("vectorindex: save rebuilt index: %w", err)
	}

	log.Printf("vectorindex: auto-rebuilt (%d vectors, %d-d, M=%d)", inserted, dimension, m)
	s.SetVectorIndex(idx)
	return nil
}

// LoadVectorIndex attempts to load the HNSW index from disk and sets it
// on this storage. It validates the content hash against the current chunk
// table state. If the hash doesn't match (chunks changed since index was built),
// the index is not set and an error is returned. Callers should fall back to
// linear scan.
func (s *SQLiteStorage) LoadVectorIndex() (*vectorindex.HNSW, [sha256.Size]byte, string, *vectorindex.SQ8Params, error) {
	idx, hash, compression, params, err := vectorindex.LoadIndex(s.VectorIndexPath())
	if err != nil {
		return nil, [sha256.Size]byte{}, "", nil, err
	}

	// Validate the loaded index matches current chunk content.
	currentHash, hashErr := s.ChunkTableContentHash()
	if hashErr != nil {
		return nil, hash, compression, params, fmt.Errorf("vectorindex: failed to validate content hash: %w", hashErr)
	}
	if currentHash != hash {
		s.vectIdxMu.Lock()
		s.vectIdx = nil
		s.vectIdxMu.Unlock()
		return nil, hash, compression, params, fmt.Errorf("vectorindex: content hash mismatch (chunks changed since index was built)")
	}
	if compression == "sq8" && params == nil {
		return nil, hash, compression, params, fmt.Errorf("vectorindex: sq8 index missing quantization params")
	}
	if idx != nil && params != nil && params.Dim != 0 && idx.Dimension() != params.Dim {
		return nil, hash, compression, params, fmt.Errorf("vectorindex: index dimension %d does not match sq8 params dimension %d", idx.Dimension(), params.Dim)
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
