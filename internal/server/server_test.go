package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHealthEndpoint_Ready(t *testing.T) {
	srv := NewServer(0, nil)
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
	srv := NewServer(0, reindexFn)
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
	srv := NewServer(0, reindexFn)
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
	srv := NewServer(0, nil)
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		rec := httptest.NewRecorder()
		srv.handleHealth(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("health request %d: expected 200, got %d", i+1, rec.Code)
		}
	}
}
