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
	"vectos/internal/usererr"
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
			// Write to both log file and stderr so user sees errors
			log.SetOutput(io.MultiWriter(f, os.Stderr))
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
	chunkerConfig := indexChunkerConfig(embedConfig.Embedded.BatchSize)
	chunker = indexer.NewSimpleChunker(chunkerConfig, embedClient)
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
		return reindexProject(cache, chunker, providerInfo, currentIndexFingerprint(chunkerConfig), req, embedConfig.VectorIndex, projectBaseDir)
	}
	embedFn := func(text string) ([]float32, error) { return embedClient.EmbedQuery(text) }
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
				onChange := makeReindexCallback(store, embedClient, embedConfig.Embedded.BatchSize)
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

func makeReindexCallback(store *storage.SQLiteStorage, embedClient embeddings.Embedder, batchSize int) func([]string) {
	return func(changedPaths []string) {
		log.Printf("watcher: detected %d changed files, triggering reindex", len(changedPaths))
		var actuallyChanged []string
		for _, path := range changedPaths {
			hash, err := computeFileHash(path)
			if err != nil {
				log.Printf("watcher: failed to hash %s: %v", path, err)
				continue
			}
			changed, err := store.HasFileChanged(path, hash)
			if err != nil {
				log.Printf("watcher: failed to check hash for %s: %v", path, err)
				continue
			}
			if changed {
				actuallyChanged = append(actuallyChanged, path)
			}
		}
		if len(actuallyChanged) == 0 {
			return
		}
		log.Printf("watcher: %d files actually changed, reindexing", len(actuallyChanged))
		if _, _, err := indexPaths(store, embedClient, actuallyChanged, batchSize); err != nil {
			log.Printf("watcher: reindex failed: %v", err)
		}
	}
}

func reindexProject(cache *storeCache, chunker *indexer.SimpleChunker, providerInfo embeddings.ProviderInfo, indexFingerprint string, req server.ReindexRequest, viCfg config.VectorIndexConfig, projectBaseDir string) server.ReindexResponse {
	scope, err := workspace.ResolveScope(req.Path, req.Project)
	if err != nil {
		return server.ReindexResponse{Status: "error", Message: err.Error()}
	}

	store, err := cache.getOrCreate(scope.Name, req.Docs)
	if err != nil {
		return server.ReindexResponse{Status: "error", Message: err.Error()}
	}

	metadataResult, err := syncIndexMetadata(store, providerInfo, indexFingerprint)
	if err != nil {
		return server.ReindexResponse{Status: "error", Message: err.Error()}
	}

	// Collect all indexable paths from scope roots with exclusion patterns.
	// Use scope.PrimaryRoot (the resolved project root for this reindex
	// request) for project-level config and .gitignore lookups so that, in
	// Nx-style workspaces, per-sub-project config and .gitignore files are
	// honored instead of always reading from projectBaseDir.
	// Fail open when GlobalConfigPath() errors: skip the global layer
	// rather than falling back to a project-local path (which would
	// silently re-introduce the original bug).
	globalCfgPath, err := config.GlobalConfigPath()
	if err != nil {
		log.Printf("warning: skipping global config layer: %v", err)
		globalCfgPath = ""
	}
	configRoot := scope.PrimaryRoot
	if configRoot == "" {
		configRoot = projectBaseDir
	}
	indexCfg := config.LoadIndexConfig(globalCfgPath, configRoot)
	excludePatterns := config.MergeExclusionPatterns(
		indexCfg.ExclusionPatterns(req.Docs),
		config.ReadGitignorePatterns(configRoot),
	)

	paths, skippedPaths, err := content.CollectIndexablePathsWithExclusions(scope.Roots, req.Docs, excludePatterns)
	if err != nil {
		return server.ReindexResponse{Status: "error", Message: err.Error()}
	}

	// If changed paths are specified, filter to only those.
	changedPaths := content.ParseChangedPaths(req.Changed)
	if len(changedPaths) > 0 && !metadataResult.FullRebuild {
		paths, skippedPaths, err = content.FilterChangedPaths(scope, paths, skippedPaths, changedPaths)
		if err != nil {
			return server.ReindexResponse{Status: "error", Message: err.Error()}
		}
	} else if len(changedPaths) > 0 && metadataResult.FullRebuild {
		log.Printf("reindex: ignoring changed-path filter for %s because index format changed", scope.Name)
	}

	// With hash-based caching, we no longer delete all chunks on full reindex.
	// Unchanged files are skipped automatically by indexPathsIntoStore.
	// Stale files (deleted from disk) are cleaned up by cleanupExcludedAndSkipped.

	// Index each path.
	indexedFiles, count := indexPathsIntoStore(store, chunker, paths, nil)
	buildVectorIndex(store, viCfg)

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
	cachedFiles := 0
	total := len(paths)
	reportEvery := progressInterval(total)
	var pending []pendingChunk

	phaseStart := time.Now()

	// Phase 1: scan files and chunk those whose hash has changed.
	// Files whose content hash matches the stored hash are skipped entirely.
	for i, path := range paths {
		fileHash, err := computeFileHash(path)
		if err != nil {
			log.Printf("warning: failed to hash %s: %v", path, err)
			continue
		}

		// Check if file is already indexed with the same hash (cache hit).
		// On error, treat as changed to avoid silently skipping files.
		changed, err := store.HasFileChanged(path, fileHash)
		if err != nil {
			log.Printf("warning: failed to check hash for %s: %v — treating as changed", path, err)
			changed = true
		}
		if !changed {
			cachedFiles++
			if progress != nil && (i+1)%reportEvery == 0 {
				fmt.Fprintf(progress, "Progress: %d/%d files scanned (%d cached, %d changed)\n", i+1, total, cachedFiles, indexedFiles)
			}
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

		if progress != nil && (i+1)%reportEvery == 0 {
			fmt.Fprintf(progress, "Progress: %d/%d files scanned (%d cached, %d changed)\n", i+1, total, cachedFiles, indexedFiles)
		}
	}

	chunkElapsed := time.Since(phaseStart)
	if progress != nil {
		fmt.Fprintf(progress, "Phase 1 (scan+chunk): %d files scanned, %d cached, %d to embed (%d chunks) in %v\n", total, cachedFiles, indexedFiles, len(pending), chunkElapsed)
	}

	if len(pending) == 0 {
		if progress != nil {
			fmt.Fprintf(progress, "All files up to date — no embedding needed\n")
		}
		return indexedFiles, 0
	}

	if progress != nil {
		fmt.Fprintf(progress, "Phase 2 (embeddings): generating embeddings for %d chunks...\n", len(pending))
	}

	// Phase 2: batch-embed all pending chunks.
	embedStart := time.Now()
	pendingChunks := make([]indexer.ChunkResult, len(pending))
	for i, p := range pending {
		pendingChunks[i] = p.chunk
	}
	var progressFn indexer.EmbedProgressFunc
	if progress != nil {
		progressFn = func(done, totalChunks int, batchDur time.Duration) {
			elapsed := time.Since(embedStart)
			rate := float64(done) / elapsed.Seconds()
			if rate > 0 {
				remaining := time.Duration(float64(totalChunks-done) / rate * float64(time.Second))
				fmt.Fprintf(progress, "  Embedding: %d/%d chunks (%.1f/sec, ETA %v)\n", done, totalChunks, rate, remaining.Round(time.Second))
			} else {
				fmt.Fprintf(progress, "  Embedding: %d/%d chunks\n", done, totalChunks)
			}
		}
	}
	if err := chunker.BatchEmbedChunksWithProgress(pendingChunks, 0, progressFn); err != nil {
		log.Printf("warning: batch embedding failed: %v", err)
	}
	embedElapsed := time.Since(embedStart)
	if progress != nil {
		fmt.Fprintf(progress, "Phase 2 (embeddings): %d chunks embedded in %v (%.1f chunks/sec)\n", len(pending), embedElapsed, float64(len(pending))/embedElapsed.Seconds())
	}

	// Phase 3: save all chunks to the store.
	saveStart := time.Now()
	savedChunks := 0
	seenPaths := make(map[string]bool)
	for i, p := range pending {
		if pendingChunks[i].Vector == nil {
			continue // skip chunks that couldn't be embedded
		}
		if _, err := store.SaveChunk(storage.CodeChunk{
			FilePath:       p.path,
			Content:        pendingChunks[i].Content,
			StartLine:      pendingChunks[i].StartLine,
			EndLine:        pendingChunks[i].EndLine,
			Language:       p.language,
			Category:       content.ClassifyCategory(p.language),
			Vector:         pendingChunks[i].Vector,
			Signature:      pendingChunks[i].Signature,
			Purpose:        pendingChunks[i].Purpose,
			PreviewSnippet: pendingChunks[i].PreviewSnippet,
		}); err != nil {
			log.Printf("warning: failed to save chunk for %s: %v", p.path, err)
			continue
		}
		savedChunks++
		// Upsert the file hash once per file, not once per chunk.
		if !seenPaths[p.path] {
			seenPaths[p.path] = true
			if err := store.UpsertIndexedFile(p.path, p.hash); err != nil {
				log.Printf("warning: failed to store hash for %s: %v", p.path, err)
			}
		}
	}
	saveElapsed := time.Since(saveStart)

	if progress != nil && indexedFiles > 0 {
		fmt.Fprintf(progress, "Phase 3 (storage): %d chunks saved in %v\n", savedChunks, saveElapsed)
		fmt.Fprintf(progress, "Total: %d files, %d chunks indexed (chunk: %v, embed: %v, save: %v)\n", indexedFiles, savedChunks, chunkElapsed, embedElapsed, saveElapsed)
	}
	return indexedFiles, savedChunks
}

func computeFileHash(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", usererr.WrapPathOp("read", "file", path, err)
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
