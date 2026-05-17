package storage

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
)

func TestIndexedFilesCRUD(t *testing.T) {
	store, cleanup := newTestSQLiteStorage(t)
	defer cleanup()

	path := "src/main.go"
	hash := "abc123"

	if got, err := store.GetIndexedFileHash(path); err != nil || got != "" {
		t.Fatalf("GetIndexedFileHash before insert = %q, %v; want empty, nil", got, err)
	}
	if err := store.UpsertIndexedFile(path, hash); err != nil {
		t.Fatalf("UpsertIndexedFile: %v", err)
	}
	got, err := store.GetIndexedFileHash(path)
	if err != nil {
		t.Fatalf("GetIndexedFileHash: %v", err)
	}
	if got != hash {
		t.Fatalf("GetIndexedFileHash = %q, want %q", got, hash)
	}
	if err := store.DeleteIndexedFile(path); err != nil {
		t.Fatalf("DeleteIndexedFile: %v", err)
	}
	got, err = store.GetIndexedFileHash(path)
	if err != nil {
		t.Fatalf("GetIndexedFileHash after delete: %v", err)
	}
	if got != "" {
		t.Fatalf("GetIndexedFileHash after delete = %q, want empty", got)
	}
}

func TestHasFileChanged(t *testing.T) {
	store, cleanup := newTestSQLiteStorage(t)
	defer cleanup()

	path := "src/main.go"
	if changed, err := store.HasFileChanged(path, "any"); err != nil || !changed {
		t.Fatalf("HasFileChanged new file = %v, %v; want true, nil", changed, err)
	}
	if err := store.UpsertIndexedFile(path, "hash1"); err != nil {
		t.Fatalf("UpsertIndexedFile: %v", err)
	}
	if changed, err := store.HasFileChanged(path, "hash1"); err != nil || changed {
		t.Fatalf("HasFileChanged same hash = %v, %v; want false, nil", changed, err)
	}
	if changed, err := store.HasFileChanged(path, "hash2"); err != nil || !changed {
		t.Fatalf("HasFileChanged different hash = %v, %v; want true, nil", changed, err)
	}
}

func TestRemoveDeletedFile(t *testing.T) {
	store, cleanup := newTestSQLiteStorage(t)
	defer cleanup()

	path := "src/main.go"
	if _, err := store.SaveChunk(CodeChunk{FilePath: path, Content: "x", StartLine: 1, EndLine: 1, Language: "go"}); err != nil {
		t.Fatalf("SaveChunk: %v", err)
	}
	if err := store.UpsertIndexedFile(path, "hash1"); err != nil {
		t.Fatalf("UpsertIndexedFile: %v", err)
	}
	if err := store.RemoveDeletedFile(path); err != nil {
		t.Fatalf("RemoveDeletedFile: %v", err)
	}

	var count int
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM code_chunks WHERE file_path = ?`, path).Scan(&count); err != nil {
		t.Fatalf("count chunks: %v", err)
	}
	if count != 0 {
		t.Fatalf("chunks count = %d, want 0", count)
	}
	if got, err := store.GetIndexedFileHash(path); err != nil || got != "" {
		t.Fatalf("GetIndexedFileHash after RemoveDeletedFile = %q, %v; want empty, nil", got, err)
	}
}

func TestReindexSameFileReplacesOldChunksAndHash(t *testing.T) {
	store, cleanup := newTestSQLiteStorage(t)
	defer cleanup()

	path := "src/main.go"
	if _, err := store.SaveChunk(CodeChunk{FilePath: path, Content: "old-token", StartLine: 1, EndLine: 1, Language: "go", Vector: []float32{1, 0, 0, 0}}); err != nil {
		t.Fatalf("SaveChunk old: %v", err)
	}
	if _, err := store.SaveChunk(CodeChunk{FilePath: path, Content: "old-token-two", StartLine: 2, EndLine: 2, Language: "go", Vector: []float32{1, 0, 0, 0}}); err != nil {
		t.Fatalf("SaveChunk old second chunk: %v", err)
	}
	if err := store.UpsertIndexedFile(path, "hash-old"); err != nil {
		t.Fatalf("UpsertIndexedFile old: %v", err)
	}

	if err := store.RemoveDeletedFile(path); err != nil {
		t.Fatalf("RemoveDeletedFile before reindex: %v", err)
	}
	if _, err := store.SaveChunk(CodeChunk{FilePath: path, Content: "new-token", StartLine: 1, EndLine: 1, Language: "go", Vector: []float32{0, 1, 0, 0}}); err != nil {
		t.Fatalf("SaveChunk new: %v", err)
	}
	if err := store.UpsertIndexedFile(path, "hash-new"); err != nil {
		t.Fatalf("UpsertIndexedFile new: %v", err)
	}

	var count int
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM code_chunks WHERE file_path = ?`, path).Scan(&count); err != nil {
		t.Fatalf("count chunks after reindex: %v", err)
	}
	if count != 1 {
		t.Fatalf("chunks count after reindex = %d, want 1", count)
	}

	if got, err := store.GetIndexedFileHash(path); err != nil || got != "hash-new" {
		t.Fatalf("GetIndexedFileHash after reindex = %q, %v; want %q, nil", got, err, "hash-new")
	}

	if results, err := store.SearchText("old-token"); err != nil {
		t.Fatalf("SearchText old-token: %v", err)
	} else if len(results) != 0 {
		t.Fatalf("expected old chunks to be removed, got %d results", len(results))
	}

	if results, err := store.SearchText("new-token"); err != nil {
		t.Fatalf("SearchText new-token: %v", err)
	} else if len(results) != 1 || results[0].FilePath != path {
		t.Fatalf("unexpected new-token results: %+v", results)
	}
}

var _ *sql.DB
var _ = filepath.Join
var _ = os.TempDir
