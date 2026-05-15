package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"vectos/internal/storage"
)

func newTestStorage(t *testing.T) *storage.SQLiteStorage {
	t.Helper()
	pm, err := storage.NewProjectManager(t.TempDir())
	if err != nil {
		t.Fatalf("new project manager: %v", err)
	}
	store, err := storage.NewSQLiteStorageForProjectName(pm, "myproject")
	if err != nil {
		t.Fatalf("new sqlite storage: %v", err)
	}
	return store
}

func newTestServer(t *testing.T, embedFn EmbedFunc) *Server {
	t.Helper()
	return NewServer(0, func(req ReindexRequest) ReindexResponse { return ReindexResponse{Status: "ok"} }, embedFn, newTestStorage(t))
}

func TestHealthEndpoint_Ready(t *testing.T) {
	srv := NewServer(0, nil, nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	srv.handleHealth(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestReindexEndpoint_Success(t *testing.T) {
	tmpDir := t.TempDir()
	reindexFn := func(req ReindexRequest) ReindexResponse {
		return ReindexResponse{Status: "ok", FilesIndexed: 5, ChunksIndexed: 10, Project: "test-project"}
	}
	srv := NewServer(0, reindexFn, nil, nil)
	body := fmt.Sprintf(`{"path":"%s"}`, tmpDir)
	req := httptest.NewRequest(http.MethodPost, "/reindex", strings.NewReader(body))
	rec := httptest.NewRecorder()
	srv.handleReindex(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestRateLimitBurst(t *testing.T) {
	tmpDir := t.TempDir()
	reindexFn := func(req ReindexRequest) ReindexResponse { return ReindexResponse{Status: "ok", FilesIndexed: 1} }
	srv := NewServer(0, reindexFn, nil, nil)
	body := `{"path":"` + strings.ReplaceAll(tmpDir, `\`, `\\`) + `"}`

	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodPost, "/reindex", strings.NewReader(body))
		rec := httptest.NewRecorder()
		srv.handleReindex(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("request %d: expected 200, got %d: %s", i+1, rec.Code, rec.Body.String())
		}
	}

	req := httptest.NewRequest(http.MethodPost, "/reindex", strings.NewReader(body))
	rec := httptest.NewRecorder()
	srv.handleReindex(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", rec.Code)
	}

	var resp map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode rate limit response: %v", err)
	}
	if resp["status"] != "error" || resp["message"] != "rate limit exceeded" {
		t.Fatalf("unexpected response: %v", resp)
	}
}

func TestHandleReindexNoRateLimitOnHealth(t *testing.T) {
	srv := NewServer(0, nil, nil, nil)
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		rec := httptest.NewRecorder()
		srv.handleHealth(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("health request %d: expected 200, got %d", i+1, rec.Code)
		}
	}
}

func TestHandleSearch_ValidRequest(t *testing.T) {
	srv := newTestServer(t, func(text string) ([]float32, error) { return []float32{0.1, 0.2}, nil })
	req := httptest.NewRequest(http.MethodPost, "/search", strings.NewReader(`{"query":"hello"}`))
	rec := httptest.NewRecorder()
	srv.handleSearchCode(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestHandleSearch_MissingQuery(t *testing.T) {
	srv := newTestServer(t, func(text string) ([]float32, error) { return []float32{0.1}, nil })
	req := httptest.NewRequest(http.MethodPost, "/search", strings.NewReader(`{"query":""}`))
	rec := httptest.NewRecorder()
	srv.handleSearchCode(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "INVALID_QUERY") {
		t.Fatalf("expected INVALID_QUERY, got %s", rec.Body.String())
	}
}

func TestHandleSearch_InvalidMethod(t *testing.T) {
	srv := newTestServer(t, func(text string) ([]float32, error) { return []float32{0.1}, nil })
	req := httptest.NewRequest(http.MethodGet, "/search", nil)
	rec := httptest.NewRecorder()
	srv.handleSearchCode(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

func TestHandleSearchCode_EndToEnd(t *testing.T) {
	srv := newTestServer(t, func(text string) ([]float32, error) { return []float32{0.1, 0.2}, nil })
	req := httptest.NewRequest(http.MethodPost, "/search/code", strings.NewReader(`{"query":"hello","limit":5}`))
	rec := httptest.NewRecorder()
	srv.handleSearchCode(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var resp SearchResult
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Mode == "" {
		t.Fatalf("expected response mode")
	}
}

func TestHandleSearchDocs_EndToEnd(t *testing.T) {
	srv := newTestServer(t, func(text string) ([]float32, error) { return []float32{0.1, 0.2}, nil })
	req := httptest.NewRequest(http.MethodPost, "/search/docs", strings.NewReader(`{"query":"hello","limit":5}`))
	rec := httptest.NewRecorder()
	srv.handleSearchDocs(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestHandleMetrics_ValidRequest(t *testing.T) {
	srv := newTestServer(t, nil)
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	srv.handleMetrics(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var resp map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode metrics: %v", err)
	}
	if _, ok := resp["chunk_count"]; !ok {
		t.Fatalf("expected chunk_count in response")
	}
}

func TestHandleMetrics_InvalidMethod(t *testing.T) {
	srv := newTestServer(t, nil)
	req := httptest.NewRequest(http.MethodPost, "/metrics", nil)
	rec := httptest.NewRecorder()
	srv.handleMetrics(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

func TestHandleStatus_ValidRequest(t *testing.T) {
	srv := newTestServer(t, nil)
	req := httptest.NewRequest(http.MethodGet, "/status/myproject", nil)
	rec := httptest.NewRecorder()
	srv.handleStatus(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestHandleStatus_InvalidProject(t *testing.T) {
	srv := newTestServer(t, nil)
	escaped := url.PathEscape("../../etc")
	req := httptest.NewRequest(http.MethodGet, "/status/"+escaped, nil)
	rec := httptest.NewRecorder()
	srv.handleStatus(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandleStatus_InvalidMethod(t *testing.T) {
	srv := newTestServer(t, nil)
	req := httptest.NewRequest(http.MethodPost, "/status/anything", nil)
	rec := httptest.NewRecorder()
	srv.handleStatus(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}
