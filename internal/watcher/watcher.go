package watcher

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

type Watcher struct {
	fsWatcher *fsnotify.Watcher
	rootPath  string
	ignores   []string
	debounce  time.Duration
	onChange  func(changedPaths []string)
	onDelete  func(deletedPath string)

	mu      sync.Mutex
	pending map[string]struct{}
	timer   *time.Timer
	ctx     context.Context
	cancel  context.CancelFunc
}

func NewWatcher(rootPath string, ignorePatterns []string, debounce time.Duration, onChange func([]string), onDelete func(string)) (*Watcher, error) {
	fsw, err := fsnotify.NewWatcher()
	if err != nil { return nil, err }
	return &Watcher{fsWatcher: fsw, rootPath: rootPath, ignores: ignorePatterns, debounce: debounce, onChange: onChange, onDelete: onDelete, pending: make(map[string]struct{})}, nil
}

func (w *Watcher) addDirsRecursively(root string) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil { return nil }
		if !info.IsDir() { return nil }
		if w.shouldIgnore(path) { return filepath.SkipDir }
		if err := w.fsWatcher.Add(path); err != nil { log.Printf("watcher: failed to watch %s: %v", path, err) }
		return nil
	})
}

func (w *Watcher) shouldIgnore(path string) bool {
	base := filepath.Base(path)
	for _, pattern := range w.ignores {
		pattern = strings.TrimSpace(pattern)
		if pattern == "" { continue }
		if matched, _ := filepath.Match(pattern, base); matched { return true }
		rel, _ := filepath.Rel(w.rootPath, path)
		if matched, _ := filepath.Match(pattern, rel); matched { return true }
	}
	return false
}

func (w *Watcher) debounceEvent(path string) {
	w.mu.Lock(); defer w.mu.Unlock()
	w.pending[path] = struct{}{}
	if w.timer != nil { w.timer.Stop() }
	w.timer = time.AfterFunc(w.debounce, func() {
		w.mu.Lock(); paths := make([]string, 0, len(w.pending)); for p := range w.pending { paths = append(paths, p) }; w.pending = make(map[string]struct{}); w.mu.Unlock()
		if len(paths) > 0 && w.onChange != nil { w.onChange(paths) }
	})
}

func (w *Watcher) Start(ctx context.Context) error {
	w.ctx, w.cancel = context.WithCancel(ctx)
	if err := w.addDirsRecursively(w.rootPath); err != nil { return err }
	go w.eventLoop()
	log.Printf("watcher: started watching %s (debounce: %v, ignores: %v)", w.rootPath, w.debounce, w.ignores)
	return nil
}

func (w *Watcher) eventLoop() {
	for {
		select {
		case <-w.ctx.Done():
			return
		case event, ok := <-w.fsWatcher.Events:
			if !ok { return }
			w.handleEvent(event)
		case err, ok := <-w.fsWatcher.Errors:
			if !ok { return }
			log.Printf("watcher error: %v", err)
		}
	}
}

func (w *Watcher) handleEvent(event fsnotify.Event) {
	path := event.Name
	if w.shouldIgnore(path) { return }
	switch {
	case event.Has(fsnotify.Create):
		if info, err := os.Stat(path); err == nil && info.IsDir() { _ = w.addDirsRecursively(path) } else { w.debounceEvent(path) }
	case event.Has(fsnotify.Write), event.Has(fsnotify.Chmod):
		w.debounceEvent(path)
	case event.Has(fsnotify.Remove), event.Has(fsnotify.Rename):
		if !w.shouldIgnore(path) && w.onDelete != nil { w.onDelete(path) }
	}
}

func (w *Watcher) Stop() {
	if w.cancel != nil { w.cancel() }
	w.mu.Lock(); if w.timer != nil { w.timer.Stop() }; w.mu.Unlock()
	_ = w.fsWatcher.Close()
}
