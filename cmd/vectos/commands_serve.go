package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"vectos/internal/config"
	"vectos/internal/content"
	"vectos/internal/embeddings"
	"vectos/internal/indexer"
	"vectos/internal/server"
	"vectos/internal/storage"
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

func runServe(projectBaseDir string, embedConfig config.EmbeddingConfig, port int) {
	configureServeLogging()

	pm, err := storage.NewProjectManager(projectBaseDir)
	if err != nil {
		log.Fatalf("error initializing project manager: %v", err)
	}

	cache := newStoreCache(pm)
	var chunker *indexer.SimpleChunker
	var providerInfo embeddings.ProviderInfo

	reindexFn := func(req server.ReindexRequest) server.ReindexResponse {
		return reindexProject(cache, chunker, providerInfo, req)
	}

	srv := server.NewServer(port, reindexFn)
	srv.SetReady(false)
	srv.AddCloser(cache)

	serveErr := make(chan error, 1)
	go func() {
		serveErr <- srv.ListenAndServe()
	}()

	select {
	case err := <-serveErr:
		log.Fatalf("server error: %v", err)
	case <-time.After(100 * time.Millisecond):
	}

	// Load embedding model once at startup while /health reports starting.
	embedClient, provider, err := embeddings.ResolveEmbedder(embedConfig)
	if err != nil {
		log.Fatalf("error resolving embedding provider: %v", err)
	}
	chunker = indexer.NewSimpleChunker(indexer.ChunkConfig{MaxLines: 10}, embedClient)
	providerInfo = provider
	srv.SetReady(true)

	if err := <-serveErr; err != nil {
		log.Fatalf("server error: %v", err)
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
func indexSingleFile(store *storage.SQLiteStorage, chunker *indexer.SimpleChunker, path string) (bool, int) {
	language, err := content.DetectLanguage(path)
	if err != nil {
		log.Printf("warning: skipping %s — unsupported language: %v", path, err)
		return false, 0
	}
	chunks, err := chunker.ChunkFile(path, language)
	if err != nil {
		log.Printf("warning: failed to chunk %s: %v", path, err)
		return false, 0
	}
	if err := store.DeleteChunksByPath(path); err != nil {
		log.Printf("warning: failed to clear previous chunks for %s: %v", path, err)
		return false, 0
	}

	created := 0
	for _, c := range chunks {
		if _, err := store.SaveChunk(storage.CodeChunk{
			FilePath:  path,
			Content:   c.Content,
			StartLine: c.StartLine,
			EndLine:   c.EndLine,
			Language:  language,
			Category:  content.ClassifyCategory(language),
			Vector:    c.Vector,
			Signature: c.Signature,
			Purpose:   c.Purpose,
		}); err != nil {
			log.Printf("warning: failed to save chunk for %s: %v", path, err)
			continue
		}
		created++
	}
	return true, created
}

func indexPathsIntoStore(store *storage.SQLiteStorage, chunker *indexer.SimpleChunker, paths []string, progress io.Writer) (int, int) {
	indexedFiles := 0
	count := 0
	total := len(paths)
	reportEvery := progressInterval(total)

	for _, path := range paths {
		ok, created := indexSingleFile(store, chunker, path)
		if !ok {
			continue
		}
		indexedFiles++
		count += created

		if progress != nil && indexedFiles%reportEvery == 0 && indexedFiles < total {
			fmt.Fprintf(progress, "Progress: %d/%d files, %d chunks indexed\n", indexedFiles, total, count)
		}
	}

	if progress != nil && indexedFiles > 0 {
		fmt.Fprintf(progress, "Progress: %d/%d files, %d chunks indexed\n", indexedFiles, total, count)
	}
	return indexedFiles, count
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

	runServe(projectBaseDir, app.embedConfig, port)
}
