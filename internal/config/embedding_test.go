package config

import (
	"strings"
	"testing"
)

func TestValidateAssetBaseURL(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		wantErr string // substring to look for in error, empty means no error expected
	}{
		{"valid HTTPS URL", "https://cdn.example.com/models", ""},
		{"valid HTTPS with path", "https://huggingface.co/BAAI/bge-small-en-v1.5/resolve/main", ""},
		{"empty URL", "", ""},
		{"whitespace only", "   ", ""},
		{"HTTP rejected", "http://cdn.example.com/models", "must use HTTPS"},
		{"file scheme rejected", "file:///etc/passwd", "must use HTTPS"},
		{"path traversal", "https://cdn.example.com/../../etc/", "path traversal"},
		{"no host", "https:///path", "non-empty host"},
		{"over length", "https://example.com/" + strings.Repeat("a", 2048), "exceeds maximum length"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAssetBaseURL(tt.raw)
			if tt.wantErr == "" {
				if err != nil {
					t.Errorf("expected no error, got: %v", err)
				}
			} else {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.wantErr)
				} else if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("expected error containing %q, got: %v", tt.wantErr, err)
				}
			}
		})
	}
}
