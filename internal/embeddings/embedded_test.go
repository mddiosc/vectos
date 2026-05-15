package embeddings

import (
	"net/http"
	"sort"
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

func TestEmbeddedModelAssets_JinaV3Entry(t *testing.T) {
	assets, ok := embeddedModelAssets["jina-embeddings-v3"]
	if !ok {
		t.Fatal("jina-embeddings-v3 not found in embeddedModelAssets")
	}

	// Should have exactly 3 assets: config.json, tokenizer.json, model.onnx
	if len(assets) != 3 {
		t.Fatalf("expected 3 assets, got %d", len(assets))
	}

	assetNames := make(map[string]string)
	for _, a := range assets {
		assetNames[a.LocalName] = a.RemotePath
	}

	if assetNames["model.onnx"] != "onnx/model.onnx" {
		t.Errorf("model.onnx RemotePath = %q, want onnx/model.onnx", assetNames["model.onnx"])
	}
	if assetNames["tokenizer.json"] != "tokenizer.json" {
		t.Errorf("tokenizer.json RemotePath = %q, want tokenizer.json", assetNames["tokenizer.json"])
	}
	if assetNames["config.json"] != "config.json" {
		t.Errorf("config.json RemotePath = %q, want config.json", assetNames["config.json"])
	}
}

func TestEmbeddedModelAssets_BGESmallEntryStillExists(t *testing.T) {
	assets, ok := embeddedModelAssets["bge-small-en-v1.5"]
	if !ok {
		t.Fatal("bge-small-en-v1.5 not found in embeddedModelAssets")
	}
	if len(assets) != 3 {
		t.Fatalf("expected 3 bge-small assets, got %d", len(assets))
	}
}

func TestDetectEmbeddingSize_UnknownDimensions(t *testing.T) {
	// detectEmbeddingSize with empty inputs should return DefaultEmbeddedDimensions
	size := detectEmbeddingSize(nil)
	if size != DefaultEmbeddedDimensions {
		t.Errorf("detectEmbeddingSize(nil) = %d, want %d", size, DefaultEmbeddedDimensions)
	}
}

func TestRequiredEmbeddedAssets(t *testing.T) {
	expected := []string{"config.json", "model.onnx", "tokenizer.json"}
	if len(requiredEmbeddedAssets) != 3 {
		t.Fatalf("expected 3 required assets, got %d", len(requiredEmbeddedAssets))
	}
	for _, e := range expected {
		found := false
		for _, a := range requiredEmbeddedAssets {
			if a == e {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("required asset %q not found", e)
		}
	}
}

func TestMissingEmbeddedAssets_AllMissing(t *testing.T) {
	missing := missingEmbeddedAssets("")
	// All 3 should be missing in empty dir
	if len(missing) != 3 {
		t.Fatalf("expected 3 missing assets for empty dir, got %d: %v", len(missing), missing)
	}
	if !sort.StringsAreSorted(missing) {
		t.Error("missingEmbeddedAssets should return sorted names")
	}
}
