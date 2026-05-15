package embeddings

import (
	"net/http"
	"testing"
)

func TestValidateDownloadContentType(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		wantErr     bool
	}{
		{"octet-stream allowed", "application/octet-stream", false},
		{"gzip allowed", "application/gzip", false},
		{"x-gzip allowed", "application/x-gzip", false},
		{"x-tar allowed", "application/x-tar", false},
		{"octet-stream with charset", "application/octet-stream; charset=binary", false},
		{"empty header", "", false},
		{"text/html rejected", "text/html", true},
		{"application/json rejected", "application/json", true},
		{"image/png rejected", "image/png", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &http.Response{
				Header: make(http.Header),
			}
			if tt.contentType != "" {
				resp.Header.Set("Content-Type", tt.contentType)
			}
			err := validateDownloadContentType(resp)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateDownloadContentType() error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}
