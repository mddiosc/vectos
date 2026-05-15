package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"golang.org/x/time/rate"
)

// ReindexRequest is the JSON body accepted by the /reindex endpoint.
type ReindexRequest struct {
	Path    string `json:"path"`
	Changed string `json:"changed,omitempty"`
	Project string `json:"project,omitempty"`
	Docs    bool   `json:"docs,omitempty"`
}

// ReindexResponse is the JSON response returned by the /reindex endpoint.
type ReindexResponse struct {
	Status        string `json:"status"`
	FilesIndexed  int    `json:"files_indexed"`
	ChunksIndexed int    `json:"chunks_indexed"`
	Project       string `json:"project"`
	Message       string `json:"message,omitempty"`
}

// ReindexFunc is a function that performs a reindex operation.
type ReindexFunc func(ReindexRequest) ReindexResponse

// Server is an HTTP server that exposes health and reindex endpoints.
type Server struct {
	port      int
	ready     atomic.Bool
	reindexFn ReindexFunc
	reindexLimiter *rate.Limiter
	httpSrv   *http.Server
	closers   []io.Closer
	closersMu sync.Mutex
	mu        sync.Mutex // serializes /reindex requests
}

// NewServer creates a new Server that listens on the given port.
func NewServer(port int, reindexFn ReindexFunc) *Server {
	s := &Server{
		port:           port,
		reindexFn:       reindexFn,
		reindexLimiter: rate.NewLimiter(rate.Limit(1), 5),
	}
	s.ready.Store(true)
	return s
}

// SetReady updates whether the server can accept reindex requests.
func (s *Server) SetReady(ready bool) {
	s.ready.Store(ready)
}

// ListenAndServe starts the HTTP server and blocks until a fatal error or
// shutdown. It listens on 127.0.0.1 to ensure localhost-only access.
func (s *Server) ListenAndServe() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/reindex", s.handleReindex)

	s.httpSrv = &http.Server{
		Addr:    fmt.Sprintf("127.0.0.1:%d", s.port),
		Handler: mux,
	}

	// Graceful shutdown on SIGINT/SIGTERM.
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		s.shutdown()
	}()

	log.Printf("vectos serve listening on 127.0.0.1:%d", s.port)
	if err := s.httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("server error: %w", err)
	}
	return nil
}

// AddCloser registers an io.Closer that will be closed during graceful shutdown.
func (s *Server) AddCloser(c io.Closer) {
	s.closersMu.Lock()
	defer s.closersMu.Unlock()
	s.closers = append(s.closers, c)
}

func (s *Server) shutdown() {
	log.Println("shutting down vectos serve...")

	// Stop accepting new requests.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := s.httpSrv.Shutdown(ctx); err != nil {
		log.Printf("error shutting down HTTP server: %v", err)
	}

	// Close all registered closers (e.g. cached SQLite connections).
	s.closersMu.Lock()
	defer s.closersMu.Unlock()
	for _, c := range s.closers {
		if err := c.Close(); err != nil {
			log.Printf("error closing resource: %v", err)
		}
	}
	log.Println("vectos serve shutdown complete")
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if s.ready.Load() {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	} else {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "starting"})
	}
}

func (s *Server) handleReindex(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if !s.ready.Load() {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "starting"})
		return
	}

	if !s.reindexLimiter.Allow() {
		writeJSON(w, http.StatusTooManyRequests, map[string]string{
			"status":  "error",
			"message": "rate limit exceeded",
		})
		return
	}

	var req ReindexRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	if req.Path == "" {
		writeJSONError(w, http.StatusBadRequest, "path is required")
		return
	}

	// Validate that the path exists.
	if fi, err := os.Stat(req.Path); err != nil || !fi.IsDir() {
		writeJSONError(w, http.StatusBadRequest, "path does not exist")
		return
	}

	// Serialize reindex requests so only one runs at a time.
	s.mu.Lock()
	defer s.mu.Unlock()

	resp := s.reindexFn(req)
	if resp.Status == "error" {
		writeJSONError(w, http.StatusInternalServerError, resp.Message)
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"status": "error", "message": message})
}
