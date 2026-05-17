package main

import (
	"testing"

	"vectos/internal/config"
	"vectos/internal/indexer"
	"vectos/internal/storage"
	"vectos/internal/workspace"
)

func TestResolveAndPrintScopeFormatsNxLibs(t *testing.T) {
	scope := workspace.Scope{
		Name:          "test-app",
		WorkspaceRoot: "/workspace",
		PrimaryRoot:   "/workspace/apps/test-app",
		Roots: []string{
			"/workspace/apps/test-app",
			"/workspace/libs/lib-core",
			"/workspace/libs/lib-ui",
		},
		WorkspaceType: "nx",
	}
	if !scope.IsWorkspace() {
		t.Fatal("expected workspace scope")
	}
	if len(scope.Roots) != 3 {
		t.Fatalf("expected 3 roots, got %d", len(scope.Roots))
	}
	if scope.Roots[0] != scope.PrimaryRoot {
		t.Fatal("expected first root to be primary root")
	}
}

func TestBuildVectorIndexRebuildsAfterFileDeletion(t *testing.T) {
	store := newIndexMetadataTestStore(t)
	chunker := indexer.NewSimpleChunker(indexer.ChunkConfig{MaxLines: 32, MinLines: 1}, nil)

	seedChunk := func(path, content string, vector []float32) {
		t.Helper()
		if _, err := store.SaveChunk(storage.CodeChunk{
			FilePath:  path,
			Content:   content,
			StartLine: 1,
			EndLine:   1,
			Language:  "go",
			Vector:    vector,
		}); err != nil {
			t.Fatalf("SaveChunk(%s): %v", path, err)
		}
	}

	seedChunk("keep.go", "keep-token", []float32{1, 0, 0, 0})
	seedChunk("delete.go", "delete-token", []float32{0, 1, 0, 0})

	buildVectorIndex(store, chunker, config.VectorIndexConfig{})
	initialIndex, initialHash, _, _, err := store.LoadVectorIndex()
	if err != nil {
		t.Fatalf("LoadVectorIndex initial: %v", err)
	}
	if initialIndex == nil || initialIndex.Len() != 2 {
		t.Fatalf("expected initial vector index with 2 vectors, got %+v", initialIndex)
	}

	if err := store.RemoveDeletedFile("delete.go"); err != nil {
		t.Fatalf("RemoveDeletedFile: %v", err)
	}

	buildVectorIndex(store, chunker, config.VectorIndexConfig{})
	rebuiltIndex, rebuiltHash, _, _, err := store.LoadVectorIndex()
	if err != nil {
		t.Fatalf("LoadVectorIndex rebuilt: %v", err)
	}
	if rebuiltIndex == nil || rebuiltIndex.Len() != 1 {
		t.Fatalf("expected rebuilt vector index with 1 vector, got %+v", rebuiltIndex)
	}
	if rebuiltHash == initialHash {
		t.Fatal("expected vector index hash to change after file deletion")
	}

	results, err := store.SearchSemantic([]float32{0, 1, 0, 0}, 5, true)
	if err != nil {
		t.Fatalf("SearchSemantic: %v", err)
	}
	for _, result := range results {
		if result.FilePath == "delete.go" {
			t.Fatalf("deleted file still present in semantic search results: %+v", result)
		}
	}
	if len(results) == 0 || results[0].FilePath != "keep.go" {
		t.Fatalf("expected rebuilt index to return remaining file, got %+v", results)
	}
}
