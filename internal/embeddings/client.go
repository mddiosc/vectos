package embeddings

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
	"vectos/internal/config"
)

// EmbeddingRequest define la estructura para enviar una petición a la API de embeddings.
type EmbeddingRequest struct {
	Input []string `json:"input"`
	Model string   `json:"model"`
}

// EmbeddingResponse define la estructura de respuesta de la API.
type EmbeddingResponse struct {
	Data []struct {
		Embedding []float32 `json:"embedding"`
	} `json:"data"`
}

// RemoteEmbedder es una implementación de Embedder que utiliza un servidor remoto vía HTTP.
type RemoteEmbedder struct {
	baseURL    string
	httpClient *http.Client
	model      string
}

// NewRemoteEmbedder crea una nueva instancia del cliente remoto.
func NewRemoteEmbedder(baseURL, model string) *RemoteEmbedder {
	return &RemoteEmbedder{
		baseURL: baseURL,
		model:   model,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func NewRemoteEmbedderFromConfig(cfg config.RemoteProviderConfig) (*RemoteEmbedder, ProviderInfo, error) {
	if !cfg.Enabled {
		return nil, ProviderInfo{}, fmt.Errorf("remote provider disabled")
	}
	if cfg.BaseURL == "" {
		return nil, ProviderInfo{}, fmt.Errorf("remote base URL is required")
	}
	if cfg.Model == "" {
		return nil, ProviderInfo{}, fmt.Errorf("remote model is required")
	}

	embedder := NewRemoteEmbedder(cfg.BaseURL, cfg.Model)
	if cfg.TimeoutS > 0 {
		embedder.httpClient.Timeout = time.Duration(cfg.TimeoutS) * time.Second
	}

	return embedder, ProviderInfo{
		Provider: config.ProviderRemote,
		Model:    cfg.Model,
	}, nil
}

// GetEmbedding implementa la interfaz Embedder llamando al servidor remoto.
func (r *RemoteEmbedder) GetEmbedding(text string) ([]float32, error) {
	reqBody := EmbeddingRequest{
		Input: []string{text},
		Model: r.model,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Asegurar que la URL termina correctamente para el endpoint
	url := r.baseURL
	if url != "" && url[len(url)-1] != '/' {
		url += "/"
	}
	url += "embeddings"

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call embedding API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("embedding API returned non-200 status: %d", resp.StatusCode)
	}

	var embResp EmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&embResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(embResp.Data) == 0 || len(embResp.Data[0].Embedding) == 0 {
		return nil, fmt.Errorf("received empty embedding response")
	}

	return embResp.Data[0].Embedding, nil
}
