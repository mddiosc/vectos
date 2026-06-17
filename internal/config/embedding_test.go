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

func TestDefaultEmbeddedModel_IsGranite(t *testing.T) {
	if DefaultEmbeddedModel != GraniteEmbeddedModel {
		t.Errorf("DefaultEmbeddedModel = %q, want %q", DefaultEmbeddedModel, GraniteEmbeddedModel)
	}
}

func TestDefaultEmbeddedAssetBaseURL_IsJinaHuggingFace(t *testing.T) {
	expected := "https://huggingface.co/jinaai/jina-embeddings-v3/resolve/main"
	if DefaultEmbeddedAssetBaseURL != expected {
		t.Errorf("DefaultEmbeddedAssetBaseURL = %q, want %q", DefaultEmbeddedAssetBaseURL, expected)
	}
}

func TestSupportedEmbeddedModels_IncludesAll(t *testing.T) {
	foundJina := false
	foundBGE := false
	foundGranite := false
	for _, m := range SupportedEmbeddedModels {
		if m == "jina-embeddings-v3" {
			foundJina = true
		}
		if m == "bge-small-en-v1.5" {
			foundBGE = true
		}
		if m == GraniteEmbeddedModel {
			foundGranite = true
		}
	}
	if !foundJina {
		t.Error("SupportedEmbeddedModels should include jina-embeddings-v3")
	}
	if !foundBGE {
		t.Error("SupportedEmbeddedModels should include bge-small-en-v1.5")
	}
	if !foundGranite {
		t.Error("SupportedEmbeddedModels should include granite-embedding-97m-multilingual-r2")
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
		{"granite supported", GraniteEmbeddedModel, true},
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
	if cfg.Embedded.AssetBaseURL != embeddedAssetBaseURL(DefaultEmbeddedModel) {
		t.Errorf("default config asset_base_url = %q, want %q", cfg.Embedded.AssetBaseURL, embeddedAssetBaseURL(DefaultEmbeddedModel))
	}
	if !cfg.Embedded.Enabled {
		t.Error("embedded should be enabled by default")
	}
	if !cfg.Embedded.AutoDownload {
		t.Error("auto_download should be enabled by default")
	}
}

func TestDefaultEmbeddingConfig_DefaultDimensions(t *testing.T) {
	cfg := DefaultEmbeddingConfig("/tmp/fake-home")
	want := DefaultEmbeddedDimensionsForModel(DefaultEmbeddedModel)
	if cfg.Embedded.Dimensions != want {
		t.Errorf("default dimensions = %d, want %d (from DefaultEmbeddedDimensionsForModel)", cfg.Embedded.Dimensions, want)
	}
}

func TestIsValidMatryoshkaDimension(t *testing.T) {
	tests := []struct {
		dim  int
		want bool
	}{
		{32, true},
		{64, true},
		{128, true},
		{256, true},
		{512, true},
		{768, true},
		{1024, true},
		{0, false},
		{1, false},
		{100, false},
		{384, true}, // granite native dim, also a valid Matryoshka size
		{2048, false},
		{-1, false},
	}
	for _, tt := range tests {
		if got := IsValidMatryoshkaDimension(tt.dim); got != tt.want {
			t.Errorf("IsValidMatryoshkaDimension(%d) = %v, want %v", tt.dim, got, tt.want)
		}
	}
}

func TestMergeEmbeddedConfig_DimensionsValidation(t *testing.T) {
	// Valid dimension should be accepted.
	dst := EmbeddedProviderConfig{ModelName: "jina-embeddings-v3", Dimensions: 512}
	dim256 := 256
	err := mergeEmbeddedConfig(&dst, embeddedProviderConfigDisk{Dimensions: &dim256})
	if err != nil {
		t.Fatalf("unexpected error for valid dimension 256: %v", err)
	}
	if dst.Dimensions != 256 {
		t.Errorf("dimensions = %d, want 256", dst.Dimensions)
	}

	// Invalid dimension should be rejected.
	dst2 := EmbeddedProviderConfig{ModelName: "jina-embeddings-v3", Dimensions: 512}
	dim100 := 100
	err = mergeEmbeddedConfig(&dst2, embeddedProviderConfigDisk{Dimensions: &dim100})
	if err == nil {
		t.Fatal("expected error for invalid dimension 100, got nil")
	}
	if !strings.Contains(err.Error(), "unsupported embedding dimensions") {
		t.Errorf("error should mention unsupported dimensions, got: %v", err)
	}
}

func TestMergeEmbeddedConfig_ResetsDimensionsForNonMatryoshkaModel(t *testing.T) {
	dst := EmbeddedProviderConfig{ModelName: DefaultEmbeddedModel, Dimensions: DefaultMatryoshkaDimensions}
	bge := "bge-small-en-v1.5"
	if err := mergeEmbeddedConfig(&dst, embeddedProviderConfigDisk{ModelName: &bge}); err != nil {
		t.Fatalf("mergeEmbeddedConfig returned error: %v", err)
	}
	if dst.Dimensions != 0 {
		t.Fatalf("expected dimensions to reset for non-Matryoshka model, got %d", dst.Dimensions)
	}
}

func TestValidateEmbeddedDimensions_NonMatryoshkaModel(t *testing.T) {
	err := ValidateEmbeddedDimensions("bge-small-en-v1.5", 256)
	if err == nil {
		t.Fatal("expected error for non-Matryoshka model")
	}
	if !strings.Contains(err.Error(), "does not support configurable dimensions") {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := ValidateEmbeddedDimensions("bge-small-en-v1.5", 0); err != nil {
		t.Fatalf("expected zero dimensions to be allowed, got %v", err)
	}
}

func TestDefaultEmbeddedDimensionsForModel(t *testing.T) {
	if got := DefaultEmbeddedDimensionsForModel("jina-embeddings-v3"); got != 1024 {
		t.Fatalf("jina default dimensions = %d, want 1024", got)
	}
	if got := DefaultEmbeddedDimensionsForModel("bge-small-en-v1.5"); got != 0 {
		t.Fatalf("bge default dimensions = %d, want 0", got)
	}
	if got := DefaultEmbeddedDimensionsForModel(GraniteEmbeddedModel); got != 0 {
		t.Fatalf("granite default dimensions = %d, want 0 (native 384)", got)
	}
}

func TestMatryoshkaDimensions_AllValid(t *testing.T) {
	for _, dim := range MatryoshkaDimensions {
		if !IsValidMatryoshkaDimension(dim) {
			t.Errorf("MatryoshkaDimensions contains %d but IsValidMatryoshkaDimension returns false", dim)
		}
	}
}
