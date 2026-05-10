package server

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// mockCloser implements io.Closer and tracks whether Close() was called.
type mockCloser struct {
	mu     sync.Mutex
	closed bool
}

func (m *mockCloser) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	return nil
}

func (m *mockCloser) WasClosed() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.closed
}

// waitForServer polls the health endpoint until the server is ready or
// a 5-second deadline expires.
func waitForServer(t *testing.T, port int) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	url := fmt.Sprintf("http://127.0.0.1:%d/health", port)
	for time.Now().Before(deadline) {
		resp, err := http.Get(url)
		if err == nil {
			resp.Body.Close()
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("server did not start within 5 seconds on port %d", port)
}

// ---------------------------------------------------------------------------
// 6.1 Unit tests for HTTP handlers
// ---------------------------------------------------------------------------

func TestHealthEndpoint_Ready(t *testing.T) {
	srv := NewServer(0, nil)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	srv.handleHealth(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	var body map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	if body["status"] != "ok" {
		t.Errorf("expected status=ok, got %q", body["status"])
	}
}

func TestHealthEndpoint_NotReady(t *testing.T) {
	srv := NewServer(0, nil)
	srv.ready.Store(false)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	srv.handleHealth(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("expected status 503, got %d", resp.StatusCode)
	}

	var body map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	if body["status"] != "starting" {
		t.Errorf("expected status=starting, got %q", body["status"])
	}
}

func TestHealthEndpoint_MethodNotAllowed(t *testing.T) {
	srv := NewServer(0, nil)

	req := httptest.NewRequest(http.MethodPost, "/health", nil)
	w := httptest.NewRecorder()
	srv.handleHealth(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", resp.StatusCode)
	}

	var body map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	if body["status"] != "error" {
		t.Errorf("expected status=error, got %q", body["status"])
	}
	if body["message"] != "method not allowed" {
		t.Errorf("expected message='method not allowed', got %q", body["message"])
	}
}

func TestReindexEndpoint_Success(t *testing.T) {
	tmpDir := t.TempDir()

	reindexFn := func(req ReindexRequest) ReindexResponse {
		return ReindexResponse{
			Status:        "ok",
			FilesIndexed:  5,
			ChunksIndexed: 10,
			Project:       "test-project",
		}
	}

	srv := NewServer(0, reindexFn)

	body := fmt.Sprintf(`{"path":"%s"}`, tmpDir)
	req := httptest.NewRequest(http.MethodPost, "/reindex", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.handleReindex(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	var result ReindexResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	if result.Status != "ok" {
		t.Errorf("expected status ok, got %s", result.Status)
	}
	if result.FilesIndexed != 5 {
		t.Errorf("expected FilesIndexed=5, got %d", result.FilesIndexed)
	}
	if result.ChunksIndexed != 10 {
		t.Errorf("expected ChunksIndexed=10, got %d", result.ChunksIndexed)
	}
	if result.Project != "test-project" {
		t.Errorf("expected Project=test-project, got %s", result.Project)
	}
}

func TestReindexEndpoint_MissingPath(t *testing.T) {
	srv := NewServer(0, nil)

	body := `{"project":"test"}`
	req := httptest.NewRequest(http.MethodPost, "/reindex", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.handleReindex(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", resp.StatusCode)
	}

	var result map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	if result["message"] != "path is required" {
		t.Errorf("expected message='path is required', got %q", result["message"])
	}
}

func TestReindexEndpoint_NotReady(t *testing.T) {
	srv := NewServer(0, nil)
	srv.SetReady(false)

	body := fmt.Sprintf(`{"path":"%s"}`, t.TempDir())
	req := httptest.NewRequest(http.MethodPost, "/reindex", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.handleReindex(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("expected status 503, got %d", resp.StatusCode)
	}

	var result map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	if result["status"] != "starting" {
		t.Errorf("expected status=starting, got %q", result["status"])
	}
}

func TestReindexEndpoint_InvalidJSON(t *testing.T) {
	srv := NewServer(0, nil)

	body := `{invalid json}`
	req := httptest.NewRequest(http.MethodPost, "/reindex", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.handleReindex(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", resp.StatusCode)
	}

	var result map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	if result["message"] != "invalid JSON body" {
		t.Errorf("expected message='invalid JSON body', got %q", result["message"])
	}
}

func TestReindexEndpoint_NonexistentPath(t *testing.T) {
	srv := NewServer(0, nil)

	body := `{"path":"/nonexistent/path/that/definitely/does/not/exist"}`
	req := httptest.NewRequest(http.MethodPost, "/reindex", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.handleReindex(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", resp.StatusCode)
	}

	var result map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	if result["message"] != "path does not exist" {
		t.Errorf("expected message='path does not exist', got %q", result["message"])
	}
}

func TestReindexEndpoint_MethodNotAllowed(t *testing.T) {
	srv := NewServer(0, nil)

	req := httptest.NewRequest(http.MethodGet, "/reindex", nil)
	w := httptest.NewRecorder()
	srv.handleReindex(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", resp.StatusCode)
	}

	var result map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	if result["status"] != "error" {
		t.Errorf("expected status=error, got %q", result["status"])
	}
	if result["message"] != "method not allowed" {
		t.Errorf("expected message='method not allowed', got %q", result["message"])
	}
}

func TestReindexEndpoint_ErrorFromReindexFn(t *testing.T) {
	tmpDir := t.TempDir()

	reindexFn := func(req ReindexRequest) ReindexResponse {
		return ReindexResponse{
			Status:  "error",
			Message: "something failed",
		}
	}

	srv := NewServer(0, reindexFn)

	body := fmt.Sprintf(`{"path":"%s"}`, tmpDir)
	req := httptest.NewRequest(http.MethodPost, "/reindex", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.handleReindex(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", resp.StatusCode)
	}

	var result map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	if result["status"] != "error" {
		t.Errorf("expected status=error, got %q", result["status"])
	}
	if result["message"] != "something failed" {
		t.Errorf("expected message='something failed', got %q", result["message"])
	}
}

// ---------------------------------------------------------------------------
// 6.2 Integration test
// ---------------------------------------------------------------------------

func TestReindexIntegration(t *testing.T) {
	tmpDir := t.TempDir()

	var callCount atomic.Int32

	reindexFn := func(req ReindexRequest) ReindexResponse {
		callCount.Add(1)
		return ReindexResponse{
			Status:        "ok",
			FilesIndexed:  3,
			ChunksIndexed: 7,
			Project:       "integration-test",
		}
	}

	srv := NewServer(0, reindexFn)
	mux := http.NewServeMux()
	mux.HandleFunc("/health", srv.handleHealth)
	mux.HandleFunc("/reindex", srv.handleReindex)

	ts := httptest.NewServer(mux)
	defer ts.Close()

	body := fmt.Sprintf(`{"path":"%s","project":"integration-test"}`, tmpDir)
	resp, err := http.Post(ts.URL+"/reindex", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("POST /reindex failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	var result ReindexResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	if result.Status != "ok" {
		t.Errorf("expected status ok, got %s", result.Status)
	}
	if result.FilesIndexed != 3 {
		t.Errorf("expected FilesIndexed=3, got %d", result.FilesIndexed)
	}
	if result.ChunksIndexed != 7 {
		t.Errorf("expected ChunksIndexed=7, got %d", result.ChunksIndexed)
	}
	if result.Project != "integration-test" {
		t.Errorf("expected Project=integration-test, got %s", result.Project)
	}

	if c := callCount.Load(); c != 1 {
		t.Errorf("expected reindexFn to be called 1 time, got %d", c)
	}
}

// ---------------------------------------------------------------------------
// 6.3 Concurrent request serialization
// ---------------------------------------------------------------------------

func TestReindexConcurrentSerialization(t *testing.T) {
	tmpDir := t.TempDir()

	var (
		callMu         sync.Mutex
		callOrder      []int
		callCount      int
		maxActiveCount int
		activeCount    int
	)

	slowReindexFn := func(req ReindexRequest) ReindexResponse {
		callMu.Lock()
		callCount++
		order := callCount
		callOrder = append(callOrder, order)
		activeCount++
		if activeCount > maxActiveCount {
			maxActiveCount = activeCount
		}
		callMu.Unlock()

		// Simulate a slow reindex operation.
		time.Sleep(100 * time.Millisecond)

		callMu.Lock()
		activeCount--
		callMu.Unlock()

		return ReindexResponse{
			Status:        "ok",
			FilesIndexed:  order,
			ChunksIndexed: 0,
			Project:       "serialization-test",
		}
	}

	srv := NewServer(0, slowReindexFn)
	mux := http.NewServeMux()
	mux.HandleFunc("/reindex", srv.handleReindex)

	ts := httptest.NewServer(mux)
	defer ts.Close()

	body := fmt.Sprintf(`{"path":"%s"}`, tmpDir)

	errs := sendReindexRequests(ts.URL+"/reindex", body, 3)
	for _, err := range errs {
		t.Errorf("request error: %v", err)
	}

	callMu.Lock()
	defer callMu.Unlock()

	assertSerializedCalls(t, callOrder, maxActiveCount)
}

func sendReindexRequests(url, body string, count int) []error {
	var wg sync.WaitGroup
	errs := make(chan error, count)

	for i := 0; i < count; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := http.Post(url, "application/json", strings.NewReader(body))
			if err != nil {
				errs <- err
				return
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				errs <- fmt.Errorf("unexpected status: %d", resp.StatusCode)
			}
		}()
	}

	wg.Wait()
	close(errs)

	var result []error
	for err := range errs {
		result = append(result, err)
	}
	return result
}

func assertSerializedCalls(t *testing.T, callOrder []int, maxActiveCount int) {
	t.Helper()

	if len(callOrder) != 3 {
		t.Fatalf("expected 3 reindexFn calls, got %d", len(callOrder))
	}

	seen := make(map[int]bool)
	for _, order := range callOrder {
		if order < 1 || order > 3 {
			t.Errorf("unexpected call order value: %d", order)
		}
		seen[order] = true
	}
	if len(seen) != 3 {
		t.Errorf("expected 3 unique call orders, got %d: %v", len(seen), callOrder)
	}

	if maxActiveCount != 1 {
		t.Errorf("expected maxActiveCount=1 (serialized), got %d", maxActiveCount)
	}
}

// ---------------------------------------------------------------------------
// 6.4 Graceful shutdown
// ---------------------------------------------------------------------------

func TestGracefulShutdown_ClosesResources(t *testing.T) {
	srv := NewServer(0, nil)

	// Set up the mux and httpSrv manually so shutdown() doesn't panic on a nil
	// httpSrv.  We use a real listener on a random port to exercise the full
	// shutdown path.
	mux := http.NewServeMux()
	mux.HandleFunc("/health", srv.handleHealth)
	mux.HandleFunc("/reindex", srv.handleReindex)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to create listener: %v", err)
	}

	port := listener.Addr().(*net.TCPAddr).Port
	srv.httpSrv = &http.Server{
		Handler: mux,
	}

	closer := &mockCloser{}
	srv.AddCloser(closer)

	// Start the HTTP server in a background goroutine.
	go func() {
		_ = srv.httpSrv.Serve(listener)
	}()

	waitForServer(t, port)

	// Trigger graceful shutdown.
	srv.shutdown()

	if !closer.WasClosed() {
		t.Error("expected registered closer to be closed after shutdown")
	}
}
