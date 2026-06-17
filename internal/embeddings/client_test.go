package embeddings

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestRemoteEmbedderTimeoutErrorIsActionable(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":[{"embedding":[1,2,3]}]}`))
	}))
	defer ts.Close()

	r := NewRemoteEmbedder(ts.URL, "test-model")
	r.httpClient.Timeout = 10 * time.Millisecond

	_, err := r.EmbedQuery("hello")
	if err == nil {
		t.Fatal("expected timeout error")
	}
	msg := err.Error()
	if !strings.Contains(msg, "timed out") || !strings.Contains(msg, "timeout_seconds") {
		t.Fatalf("unexpected error: %v", err)
	}
	var urlErr *url.Error
	if !errors.As(err, &urlErr) {
		t.Fatalf("expected wrapped *url.Error, got %T", err)
	}
}

func TestRemoteEmbedderRateLimitErrorIsActionable(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "slow down", http.StatusTooManyRequests)
	}))
	defer ts.Close()

	r := NewRemoteEmbedder(ts.URL, "test-model")
	_, err := r.EmbedQuery("hello")
	if err == nil {
		t.Fatal("expected rate-limit error")
	}
	if !strings.Contains(err.Error(), "rate limited") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRemoteEmbedderUnauthorizedErrorIsActionable(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "missing token", http.StatusUnauthorized)
	}))
	defer ts.Close()

	r := NewRemoteEmbedder(ts.URL, "test-model")
	_, err := r.EmbedQuery("hello")
	if err == nil {
		t.Fatal("expected unauthorized error")
	}
	if !strings.Contains(err.Error(), "upstream authentication") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRemoteEmbedderInvalidJSONErrorIsActionable(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`not-json`))
	}))
	defer ts.Close()

	r := NewRemoteEmbedder(ts.URL, "test-model")
	_, err := r.EmbedQuery("hello")
	if err == nil {
		t.Fatal("expected decode error")
	}
	if !strings.Contains(err.Error(), "invalid JSON response") {
		t.Fatalf("unexpected error: %v", err)
	}
}
