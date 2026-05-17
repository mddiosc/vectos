package embeddings

import (
	"math"
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
		{"text/plain allowed", "text/plain", false},
		{"application/json allowed", "application/json", false},
		{"text/html rejected", "text/html", true},
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

	// Should have exactly 4 assets: config.json, tokenizer.json, model.onnx, model.onnx_data
	if len(assets) != 4 {
		t.Fatalf("expected 4 assets, got %d", len(assets))
	}

	assetNames := make(map[string]string)
	for _, a := range assets {
		assetNames[a.LocalName] = a.RemotePath
	}

	if assetNames["model.onnx"] != "onnx/model.onnx" {
		t.Errorf("model.onnx RemotePath = %q, want onnx/model.onnx", assetNames["model.onnx"])
	}
	if assetNames["model.onnx_data"] != "onnx/model.onnx_data" {
		t.Errorf("model.onnx_data RemotePath = %q, want onnx/model.onnx_data", assetNames["model.onnx_data"])
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
	missing := missingEmbeddedAssets("", "")
	// All 3 required assets should be missing in empty dir (no model-specific extras for empty model)
	if len(missing) != 3 {
		t.Fatalf("expected 3 missing assets for empty dir, got %d: %v", len(missing), missing)
	}
	if !sort.StringsAreSorted(missing) {
		t.Error("missingEmbeddedAssets should return sorted names")
	}
}

func TestMissingEmbeddedAssets_JinaV3(t *testing.T) {
	missing := missingEmbeddedAssets("jina-embeddings-v3", "")
	// jina-embeddings-v3 has 4 assets (includes model.onnx_data)
	if len(missing) != 4 {
		t.Fatalf("expected 4 missing assets for jina-embeddings-v3 in empty dir, got %d: %v", len(missing), missing)
	}
}

// l2norm computes the L2 norm of a float32 vector.
func l2norm(v []float32) float64 {
	var sum float64
	for _, x := range v {
		sum += float64(x * x)
	}
	return math.Sqrt(sum)
}

func TestTruncateAndNormalize_Basic(t *testing.T) {
	// Create a normalized 1024d vector with known values.
	original := make([]float32, 1024)
	for i := range original {
		original[i] = float32(i + 1)
	}
	// Normalize it first.
	norm := l2norm(original)
	for i := range original {
		original[i] = float32(float64(original[i]) / norm)
	}

	tests := []struct {
		name      string
		targetDim int
		wantLen   int
	}{
		{"truncate to 512", 512, 512},
		{"truncate to 256", 256, 256},
		{"truncate to 128", 128, 128},
		{"truncate to 64", 64, 64},
		{"truncate to 32", 32, 32},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateAndNormalize(original, tt.targetDim)
			if len(result) != tt.wantLen {
				t.Fatalf("got len %d, want %d", len(result), tt.wantLen)
			}

			// Result must be L2-normalized (unit length).
			resultNorm := l2norm(result)
			if math.Abs(resultNorm-1.0) > 1e-5 {
				t.Errorf("result norm = %f, want ~1.0", resultNorm)
			}

			// Values should match the first N dimensions of the original
			// (before re-normalization), scaled by the new norm.
			// Just verify they're proportional: result[0]/result[1] == original[0]/original[1]
			if tt.targetDim >= 2 {
				origRatio := float64(original[0]) / float64(original[1])
				resultRatio := float64(result[0]) / float64(result[1])
				if math.Abs(origRatio-resultRatio) > 1e-5 {
					t.Errorf("ratio mismatch: original[0]/[1] = %f, result[0]/[1] = %f", origRatio, resultRatio)
				}
			}
		})
	}
}

func TestTruncateAndNormalize_NoOpCases(t *testing.T) {
	vec := []float32{0.5, 0.5, 0.5, 0.5}

	// targetDim >= len(vec) → no truncation
	result := truncateAndNormalize(vec, 4)
	if len(result) != 4 {
		t.Errorf("expected no truncation for targetDim == len, got len %d", len(result))
	}

	result = truncateAndNormalize(vec, 10)
	if len(result) != 4 {
		t.Errorf("expected no truncation for targetDim > len, got len %d", len(result))
	}

	// targetDim <= 0 → no truncation
	result = truncateAndNormalize(vec, 0)
	if len(result) != 4 {
		t.Errorf("expected no truncation for targetDim == 0, got len %d", len(result))
	}

	result = truncateAndNormalize(vec, -1)
	if len(result) != 4 {
		t.Errorf("expected no truncation for targetDim < 0, got len %d", len(result))
	}
}

func TestTruncateAndNormalize_ZeroVector(t *testing.T) {
	vec := make([]float32, 8)
	result := truncateAndNormalize(vec, 4)
	if len(result) != 4 {
		t.Fatalf("got len %d, want 4", len(result))
	}
	// Zero vector stays zero (no division by zero).
	for i, v := range result {
		if v != 0 {
			t.Errorf("result[%d] = %f, want 0", i, v)
		}
	}
}

func TestTruncateAndNormalize_DoesNotMutateOriginal(t *testing.T) {
	original := []float32{0.6, 0.8, 0.0, 0.0}
	originalCopy := make([]float32, len(original))
	copy(originalCopy, original)

	_ = truncateAndNormalize(original, 2)

	for i := range original {
		if original[i] != originalCopy[i] {
			t.Errorf("original[%d] mutated: got %f, want %f", i, original[i], originalCopy[i])
		}
	}
}
