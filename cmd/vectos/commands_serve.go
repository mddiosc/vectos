package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"vectos/internal/config"
	"vectos/internal/content"
	"vectos/internal/embeddings"
	"vectos/internal/indexer"
	"vectos/internal/server"
	"vectos/internal/storage"
	"vectos/internal/watcher"
	"vectos/internal/workspace"
)

// storeCache keeps SQLite connections open across requests for reuse.
type storeCache struct {
	mu     sync.Mutex
	stores map[string]*storage.SQLiteStorage
	pm     *storage.ProjectManager
}

func newStoreCache(pm *storage.ProjectManager) *storeCache {
	return &storeCache{
		stores: make(map[string]*storage.SQLiteStorage),
		pm:     pm,
	}
}

func (sc *storeCache) getOrCreate(projectName string, docs bool) (*storage.SQLiteStorage, error) {
	key := projectName
	if docs {
		key = projectName + "-docs"
	}

	sc.mu.Lock()
	defer sc.mu.Unlock()

	if store, ok := sc.stores[key]; ok {
		return store, nil
	}

	var store *storage.SQLiteStorage
	var err error
	if docs {
		store, err = storage.NewSQLiteStorageForDocsProjectName(sc.pm, projectName)
	} else {
		store, err = storage.NewSQLiteStorageForProjectName(sc.pm, projectName)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to open storage for %s: %w", projectName, err)
	}

	sc.stores[key] = store
	return store, nil
}

func (sc *storeCache) closeAll() {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	for key, store := range sc.stores {
		if err := store.Close(); err != nil {
			log.Printf("error closing store %s: %v", key, err)
		}
		delete(sc.stores, key)
	}
}

func (sc *storeCache) Close() error {
	sc.closeAll()
	return nil
}

var _ io.Closer = (*storeCache)(nil)

func configureServeLogging() {
	home, _ := os.UserHomeDir()
	logPath := filepath.Join(home, ".vectos", "vectos-serve.log")
	if err := os.MkdirAll(filepath.Dir(logPath), 0755); err == nil {
		if f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err == nil {
			log.SetOutput(f)
			log.Printf("starting vectos serve")
			return
		}
	}
	log.SetOutput(os.Stderr)
}

func runServe(projectBaseDir string, embedConfig config.EmbeddingConfig, port int, watchEnabled bool, watchDebounce time.Duration, watchIgnore string) {
	configureServeLogging()

	pm, err := storage.NewProjectManager(projectBaseDir)
	if err != nil {
		log.Fatalf("error initializing project manager: %v", err)
	}

	cache := newStoreCache(pm)
	var chunker *indexer.SimpleChunker
	var providerInfo embeddings.ProviderInfo

	// Load embedding model once at startup while /health reports starting.
	embedClient, provider, err := embeddings.ResolveEmbedder(embedConfig)
	if err != nil {
		log.Fatalf("error resolving embedding provider: %v", err)
	}
	chunker = indexer.NewSimpleChunker(indexer.ChunkConfig{MaxLines: 10}, embedClient)
	providerInfo = provider

	var activeStore *storage.SQLiteStorage
	if scope, err := workspace.ResolveScope(projectBaseDir, ""); err == nil {
		if store, err := cache.getOrCreate(scope.Name, false); err == nil {
			activeStore = store
			if _, _, _, _, err := store.LoadVectorIndex(); err != nil {
				log.Printf("warning: vector index not available, using linear scan: %v", err)
			}
		}
	}

	reindexFn := func(req server.ReindexRequest) server.ReindexResponse {
		return reindexProject(cache, chunker, providerInfo, req)
	}
	embedFn := func(text string) ([]float32, error) { return embedClient.GetEmbedding(text) }
	srv := server.NewServer(port, reindexFn, embedFn, activeStore)
	srv.SetReady(false)
	srv.AddCloser(cache)

	serveErr := make(chan error, 1)
	go func() { serveErr <- srv.ListenAndServe() }()

	select {
	case err := <-serveErr:
		log.Fatalf("server error: %v", err)
	case <-time.After(100 * time.Millisecond):
	}
	srv.SetReady(true)
	if watchEnabled {
		ignorePatterns := strings.Split(watchIgnore, ",")
		for i := range ignorePatterns {
			ignorePatterns[i] = strings.TrimSpace(ignorePatterns[i])
		}

		if scope, err := workspace.ResolveScope(projectBaseDir, ""); err == nil {
			if store, err := cache.getOrCreate(scope.Name, false); err == nil {
				onChange := makeReindexCallback(store, embedClient)
				onDelete := makeDeleteHandler(store)
				if w, err := watcher.NewWatcher(projectBaseDir, ignorePatterns, watchDebounce, onChange, onDelete); err == nil {
					if err := w.Start(context.Background()); err == nil {
						srv.AddCloser(watcherCloser{w: w})
					} else {
						log.Printf("warning: failed to start watcher: %v", err)
					}
				} else {
					log.Printf("warning: failed to create watcher: %v", err)
				}
			}
		} else {
			log.Printf("warning: failed to resolve watcher scope: %v", err)
		}
	}

	if err := <-serveErr; err != nil {
		log.Fatalf("server error: %v", err)
	}
}

type watcherCloser struct{ w *watcher.Watcher }

func (wc watcherCloser) Close() error { wc.w.Stop(); return nil }

func makeDeleteHandler(store *storage.SQLiteStorage) func(string) {
	return func(path string) {
		if err := store.RemoveDeletedFile(path); err != nil {
			log.Printf("watcher: failed to remove deleted file %s: %v", path, err)
			return
		}
		log.Printf("watcher: removed deleted file from index: %s", path)
	}
}

func makeReindexCallback(store *storage.SQLiteStorage, embedClient embeddings.Embedder) func([]string) {
	return func(changedPaths []string) {
		log.Printf("watcher: detected %d changed files, triggering reindex", len(changedPaths))
		var actuallyChanged []string
		for _, path := range changedPaths {
			hash, err := computeFileHash(path)
			if err != nil { log.Printf("watcher: failed to hash %s: %v", path, err); continue }
			changed, err := store.HasFileChanged(path, hash)
			if err != nil { log.Printf("watcher: failed to check hash for %s: %v", path, err); continue }
			if changed { actuallyChanged = append(actuallyChanged, path) }
		}
		if len(actuallyChanged) == 0 { return }
		log.Printf("watcher: %d files actually changed, reindexing", len(actuallyChanged))
		if _, _, err := indexPaths(store, embedClient, actuallyChanged); err != nil {
			log.Printf("watcher: reindex failed: %v", err)
		}
	}
}

func reindexProject(cache *storeCache, chunker *indexer.SimpleChunker, providerInfo embeddings.ProviderInfo, req server.ReindexRequest) server.ReindexResponse {
	scope, err := workspace.ResolveScope(req.Path, req.Project)
	if err != nil {
		return server.ReindexResponse{Status: "error", Message: err.Error()}
	}

	store, err := cache.getOrCreate(scope.Name, req.Docs)
	if err != nil {
		return server.ReindexResponse{Status: "error", Message: err.Error()}
	}

	if err := store.SetIndexMetadata(storage.IndexMetadata{
		Provider:   providerInfo.Provider,
		Model:      providerInfo.Model,
		Dimensions: providerInfo.Dimensions,
	}); err != nil {
		return server.ReindexResponse{Status: "error", Message: err.Error()}
	}

	// Collect all indexable paths from scope roots.
	paths, skippedPaths, err := content.CollectIndexablePaths(scope.Roots, req.Docs)
	if err != nil {
		return server.ReindexResponse{Status: "error", Message: err.Error()}
	}

	// If changed paths are specified, filter to only those.
	changedPaths := content.ParseChangedPaths(req.Changed)
	if len(changedPaths) > 0 {
		paths, skippedPaths, err = content.FilterChangedPaths(scope, paths, skippedPaths, changedPaths)
		if err != nil {
			return server.ReindexResponse{Status: "error", Message: err.Error()}
		}
	}

	// Prepare store: full reindex clears all chunks, incremental does not.
	if len(changedPaths) == 0 {
		if err := store.DeleteAllChunks(); err != nil {
			return server.ReindexResponse{Status: "error", Message: err.Error()}
		}
	}

	// Index each path.
	indexedFiles, count := indexPathsIntoStore(store, chunker, paths, nil)
	buildVectorIndex(store, chunker)

	// Clean up excluded directories and skipped paths.
	cleanupExcludedAndSkipped(store, scope, skippedPaths)

	return server.ReindexResponse{
		Status:        "ok",
		FilesIndexed:  indexedFiles,
		ChunksIndexed: count,
		Project:       scope.Name,
	}
}

// progressInterval computes a print-every-N interval so that ~10 updates are
// shown for a full index, bounded to [1, 100].
func progressInterval(total int) int {
	switch {
	case total < 10:
		return 1
	case total > 1000:
		return 100
	default:
		return total / 10
	}
}

// indexSingleFile indexes one file and returns (indexedOK, chunksCreated).
// pendingChunk holds a raw chunk together with the file metadata needed to
// save it after batch embedding.
type pendingChunk struct {
	chunk    indexer.ChunkResult
	path     string
	language string
	hash     string
}

func indexPathsIntoStore(store *storage.SQLiteStorage, chunker *indexer.SimpleChunker, paths []string, progress io.Writer) (int, int) {
	indexedFiles := 0
	total := len(paths)
	reportEvery := progressInterval(total)
	var pending []pendingChunk

	// Phase 1: chunk every file (no embedding) and clear old rows.
	for _, path := range paths {
		fileHash, err := computeFileHash(path)
		if err != nil {
			log.Printf("warning: failed to hash %s: %v", path, err)
			continue
		}
		language, err := content.DetectLanguage(path)
		if err != nil {
			log.Printf("warning: skipping %s — unsupported language: %v", path, err)
			continue
		}
		chunks, err := chunker.ChunkFileRaw(path, language)
		if err != nil {
			log.Printf("warning: failed to chunk %s: %v", path, err)
			continue
		}
		if err := store.RemoveDeletedFile(path); err != nil {
			log.Printf("warning: failed to clear previous chunks for %s: %v", path, err)
			continue
		}
		for _, c := range chunks {
			pending = append(pending, pendingChunk{chunk: c, path: path, language: language, hash: fileHash})
		}
		indexedFiles++

		if progress != nil && indexedFiles%reportEvery == 0 && indexedFiles < total {
			fmt.Fprintf(progress, "Progress: %d/%d files chunked\n", indexedFiles, total)
		}
	}

	if progress != nil && indexedFiles > 0 {
		fmt.Fprintf(progress, "Progress: %d/%d files chunked — generating embeddings in batches...\n", indexedFiles, total)
	}

	// Phase 2: batch-embed all pending chunks.
	pendingChunks := make([]indexer.ChunkResult, len(pending))
	for i, p := range pending {
		pendingChunks[i] = p.chunk
	}
	if err := chunker.BatchEmbedChunks(pendingChunks, 0); err != nil {
		log.Printf("warning: batch embedding failed: %v", err)
	}

	// Phase 3: save all chunks to the store.
	savedChunks := 0
	for i, p := range pending {
		if pendingChunks[i].Vector == nil {
			continue // skip chunks that couldn't be embedded
		}
		if _, err := store.SaveChunk(storage.CodeChunk{
			FilePath:  p.path,
			Content:   pendingChunks[i].Content,
			StartLine: pendingChunks[i].StartLine,
			EndLine:   pendingChunks[i].EndLine,
			Language:  p.language,
			Category:  content.ClassifyCategory(p.language),
			Vector:    pendingChunks[i].Vector,
			Signature: pendingChunks[i].Signature,
			Purpose:   pendingChunks[i].Purpose,
		}); err != nil {
			log.Printf("warning: failed to save chunk for %s: %v", p.path, err)
			continue
		}
		if err := store.UpsertIndexedFile(p.path, p.hash); err != nil {
			log.Printf("warning: failed to store hash for %s: %v", p.path, err)
			continue
		}
		savedChunks++
	}

	if progress != nil && indexedFiles > 0 {
		fmt.Fprintf(progress, "Progress: %d/%d files, %d chunks indexed\n", indexedFiles, total, savedChunks)
	}
	return indexedFiles, savedChunks
}

func computeFileHash(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:]), nil
}

func cleanupExcludedAndSkipped(store *storage.SQLiteStorage, scope workspace.Scope, skippedPaths []string) {
	for _, root := range scope.Roots {
		for _, excludedDir := range content.CollectExcludedDirs(root) {
			if err := store.DeleteChunksByPathPrefix(excludedDir); err != nil {
				log.Printf("warning: failed to clean excluded dir %s: %v", excludedDir, err)
			}
		}
	}
	for _, skippedPath := range skippedPaths {
		if err := store.DeleteChunksByPath(skippedPath); err != nil {
			log.Printf("warning: failed to clear skipped path %s: %v", skippedPath, err)
		}
	}
}

func runServeCommand(app appContext, args []string) {
	if hasHelpFlag(args) {
		printSubcommandHelp("serve")
		return
	}
	if err := app.flags.serveCmd.Parse(args); err != nil {
		fatalErr(err)
	}

	port := *app.flags.servePort
	if port <= 0 || port > 65535 {
		log.Fatalf("invalid port: %d", port)
	}
	projectBaseDir := app.projectBaseDir
	if *app.flags.serveProjectBaseDir != "" {
		projectBaseDir = *app.flags.serveProjectBaseDir
	}

	runServe(projectBaseDir, app.embedConfig, port, *app.flags.watchEnabled, *app.flags.watchDebounce, *app.flags.watchIgnore)
}
