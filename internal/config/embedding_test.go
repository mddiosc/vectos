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

func TestDefaultEmbeddedModel_IsJinaV3(t *testing.T) {
	if DefaultEmbeddedModel != "jina-embeddings-v3" {
		t.Errorf("DefaultEmbeddedModel = %q, want jina-embeddings-v3", DefaultEmbeddedModel)
	}
}

func TestDefaultEmbeddedAssetBaseURL_IsJinaHuggingFace(t *testing.T) {
	expected := "https://huggingface.co/jinaai/jina-embeddings-v3/resolve/main"
	if DefaultEmbeddedAssetBaseURL != expected {
		t.Errorf("DefaultEmbeddedAssetBaseURL = %q, want %q", DefaultEmbeddedAssetBaseURL, expected)
	}
}

func TestSupportedEmbeddedModels_IncludesBoth(t *testing.T) {
	foundJina := false
	foundBGE := false
	for _, m := range SupportedEmbeddedModels {
		if m == "jina-embeddings-v3" {
			foundJina = true
		}
		if m == "bge-small-en-v1.5" {
			foundBGE = true
		}
	}
	if !foundJina {
		t.Error("SupportedEmbeddedModels should include jina-embeddings-v3")
	}
	if !foundBGE {
		t.Error("SupportedEmbeddedModels should include bge-small-en-v1.5")
	}
}

func TestIsSupportedEmbeddedModel(t *testing.T) {
	tests := []struct {
		name  string
		model string
		want  bool
	}{
		{"jina-v3 supported", "jina-embeddings-v3", true},
		{"bge-small supported", "bge-small-en-v1.5", true},
		{"unknown model rejected", "unknown-model", false},
		{"empty rejected", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isSupportedEmbeddedModel(tt.model); got != tt.want {
				t.Errorf("isSupportedEmbeddedModel(%q) = %v, want %v", tt.model, got, tt.want)
			}
		})
	}
}

func TestDefaultEmbeddingConfig_UsesDefaultModel(t *testing.T) {
	cfg := DefaultEmbeddingConfig("/tmp/fake-home")
	if cfg.Embedded.ModelName != DefaultEmbeddedModel {
		t.Errorf("default config model = %q, want %q", cfg.Embedded.ModelName, DefaultEmbeddedModel)
	}
	if cfg.Embedded.AssetBaseURL != DefaultEmbeddedAssetBaseURL {
		t.Errorf("default config asset_base_url = %q, want %q", cfg.Embedded.AssetBaseURL, DefaultEmbeddedAssetBaseURL)
	}
	if !cfg.Embedded.Enabled {
		t.Error("embedded should be enabled by default")
	}
	if !cfg.Embedded.AutoDownload {
		t.Error("auto_download should be enabled by default")
	}
}
