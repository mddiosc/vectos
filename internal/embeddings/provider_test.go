package embeddings

import (
	"testing"
)

// mockSingleEmbedder implements Embedder.EmbedPassage only; used to verify
// that EmbedPassagesDefault delegates correctly.
type mockSingleEmbedder struct {
	embeddings map[string][]float32
}

func (m *mockSingleEmbedder) EmbedQuery(text string) ([]float32, error) {
	if v, ok := m.embeddings[text]; ok {
		return v, nil
	}
	return []float32{0.0}, nil
}

func (m *mockSingleEmbedder) EmbedQueries(texts []string) ([][]float32, error) {
	results := make([][]float32, len(texts))
	for i, text := range texts {
		vec, err := m.EmbedQuery(text)
		if err != nil {
			return nil, err
		}
		results[i] = vec
	}
	return results, nil
}

func (m *mockSingleEmbedder) EmbedPassage(text string) ([]float32, error) {
	if v, ok := m.embeddings[text]; ok {
		return v, nil
	}
	return []float32{0.0}, nil
}

func (m *mockSingleEmbedder) EmbedPassages(texts []string) ([][]float32, error) {
	return EmbedPassagesDefault(m, texts)
}

func TestEmbedPassagesDefault_EmptyBatch(t *testing.T) {
	e := &mockSingleEmbedder{embeddings: map[string][]float32{}}
	vecs, err := e.EmbedPassages(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(vecs) != 0 {
		t.Errorf("expected empty result, got %d vectors", len(vecs))
	}
}

func TestEmbedPassagesDefault_SingleText(t *testing.T) {
	e := &mockSingleEmbedder{embeddings: map[string][]float32{
		"hello": {1.0, 2.0, 3.0},
	}}
	vecs, err := e.EmbedPassages([]string{"hello"})
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

func TestEmbedPassagesDefault_MultipleTexts(t *testing.T) {
	e := &mockSingleEmbedder{embeddings: map[string][]float32{
		"a": {1.0},
		"b": {2.0},
		"c": {3.0},
	}}
	vecs, err := e.EmbedPassages([]string{"a", "b", "c"})
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

func TestEmbedPassagesDefault_PreservesOrder(t *testing.T) {
	e := &mockSingleEmbedder{embeddings: map[string][]float32{
		"first":  {10.0},
		"second": {20.0},
		"third":  {30.0},
	}}
	vecs, err := e.EmbedPassages([]string{"first", "second", "third"})
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