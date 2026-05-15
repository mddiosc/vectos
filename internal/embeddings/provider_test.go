package embeddings

import (
	"testing"
)

// mockSingleEmbedder implements Embedder.GetEmbedding only; used to verify
// that GetEmbeddingsDefault delegates correctly.
type mockSingleEmbedder struct {
	embeddings map[string][]float32
}

func (m *mockSingleEmbedder) GetEmbedding(text string) ([]float32, error) {
	if v, ok := m.embeddings[text]; ok {
		return v, nil
	}
	return []float32{0.0}, nil
}

func (m *mockSingleEmbedder) GetEmbeddings(texts []string) ([][]float32, error) {
	return GetEmbeddingsDefault(m, texts)
}

func TestGetEmbeddingsDefault_EmptyBatch(t *testing.T) {
	e := &mockSingleEmbedder{embeddings: map[string][]float32{}}
	vecs, err := e.GetEmbeddings(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(vecs) != 0 {
		t.Errorf("expected empty result, got %d vectors", len(vecs))
	}
}

func TestGetEmbeddingsDefault_SingleText(t *testing.T) {
	e := &mockSingleEmbedder{embeddings: map[string][]float32{
		"hello": {1.0, 2.0, 3.0},
	}}
	vecs, err := e.GetEmbeddings([]string{"hello"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(vecs) != 1 {
		t.Fatalf("expected 1 vector, got %d", len(vecs))
	}
	if len(vecs[0]) != 3 || vecs[0][0] != 1.0 || vecs[0][1] != 2.0 || vecs[0][2] != 3.0 {
		t.Errorf("unexpected vector: %v", vecs[0])
	}
}

func TestGetEmbeddingsDefault_MultipleTexts(t *testing.T) {
	e := &mockSingleEmbedder{embeddings: map[string][]float32{
		"a": {1.0},
		"b": {2.0},
		"c": {3.0},
	}}
	vecs, err := e.GetEmbeddings([]string{"a", "b", "c"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(vecs) != 3 {
		t.Fatalf("expected 3 vectors, got %d", len(vecs))
	}
	for i, expected := range []float32{1.0, 2.0, 3.0} {
		if vecs[i][0] != expected {
			t.Errorf("vecs[%d][0] = %v, want %v", i, vecs[i][0], expected)
		}
	}
}

func TestGetEmbeddingsDefault_PreservesOrder(t *testing.T) {
	e := &mockSingleEmbedder{embeddings: map[string][]float32{
		"first":  {10.0},
		"second": {20.0},
		"third":  {30.0},
	}}
	vecs, err := e.GetEmbeddings([]string{"first", "second", "third"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(vecs) != 3 {
		t.Fatalf("expected 3 vectors, got %d", len(vecs))
	}
	if vecs[0][0] != 10.0 || vecs[1][0] != 20.0 || vecs[2][0] != 30.0 {
		t.Errorf("order mismatch: %v", vecs)
	}
}
