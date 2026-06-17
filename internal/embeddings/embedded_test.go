package embeddings

import (
	"math"
	"net/http"
	"sort"
	"strings"
	"testing"

	"vectos/internal/config"
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

func TestEmbeddedModelAssets_GraniteEntry(t *testing.T) {
	assets, ok := embeddedModelAssets[config.GraniteEmbeddedModel]
	if !ok {
		t.Fatal("granite-embedding-97m-multilingual-r2 not found in embeddedModelAssets")
	}
	if len(assets) != 3 {
		t.Fatalf("expected 3 granite assets, got %d", len(assets))
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

func TestPoolingForModel(t *testing.T) {
	if got := poolingForModel(config.GraniteEmbeddedModel); got != "cls" {
		t.Errorf("poolingForModel(granite) = %q, want cls", got)
	}
	if got := poolingForModel("jina-embeddings-v3"); got != "mean" {
		t.Errorf("poolingForModel(jina) = %q, want mean", got)
	}
	if got := poolingForModel("bge-small-en-v1.5"); got != "mean" {
		t.Errorf("poolingForModel(bge) = %q, want mean", got)
	}
}

func TestClsPoolAndNormalize(t *testing.T) {
	// 2 tokens, 4 hidden dims: [[1,2,3,4], [5,6,7,8]]
	data := []float32{1, 2, 3, 4, 5, 6, 7, 8}
	result := clsPoolAndNormalize(data, 4)
	if len(result) != 4 {
		t.Fatalf("expected 4 dims, got %d", len(result))
	}
	// CLS takes first token: [1,2,3,4], L2 norm = sqrt(1+4+9+16) = sqrt(30)
	// normalized: [1/sqrt30, 2/sqrt30, 3/sqrt30, 4/sqrt30]
	norm := l2norm(result)
	if math.Abs(norm-1.0) > 1e-5 {
		t.Errorf("result norm = %f, want ~1.0", norm)
	}
	// Verify values are proportional to [1,2,3,4]
	if result[0] <= 0 || result[1] <= 0 {
		t.Error("expected positive values from CLS pooling of positive input")
	}
	ratio := float64(result[1]) / float64(result[0])
	if math.Abs(ratio-2.0) > 1e-5 {
		t.Errorf("ratio result[1]/result[0] = %f, want 2.0", ratio)
	}
}

func TestClsPoolAndNormalize_ZeroVector(t *testing.T) {
	data := make([]float32, 8)
	result := clsPoolAndNormalize(data, 4)
	if len(result) != 4 {
		t.Fatalf("expected 4 dims, got %d", len(result))
	}
	for i, v := range result {
		if v != 0 {
			t.Errorf("zero input result[%d] = %f, want 0", i, v)
		}
	}
}

func TestClsPoolAndNormalize_InsufficientData(t *testing.T) {
	data := []float32{1, 2}
	result := clsPoolAndNormalize(data, 4)
	if result != nil {
		t.Errorf("expected nil for insufficient data, got %v", result)
	}
}

func TestNewEmbeddedEmbedderWithStatusRejectsDimensionsForNonMatryoshkaModel(t *testing.T) {
	_, _, err := NewEmbeddedEmbedderWithStatus(config.EmbeddedProviderConfig{
		Enabled:    true,
		ModelName:  "bge-small-en-v1.5",
		ModelDir:   t.TempDir(),
		Dimensions: 256,
	})
	if err == nil {
		t.Fatal("expected error for non-Matryoshka dimensions")
	}
	if got := err.Error(); got == "" || !strings.Contains(got, "does not support configurable dimensions") {
		t.Fatalf("unexpected error: %v", err)
	}
}
