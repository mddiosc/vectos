package embeddings

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
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

	dimensions, err := embedder.detectDimensions()
	if err != nil {
		return nil, ProviderInfo{}, err
	}

	return embedder, ProviderInfo{
		Provider:   config.ProviderRemote,
		Model:      cfg.Model,
		Dimensions: dimensions,
	}, nil
}

func InspectRemoteProvider(cfg config.RemoteProviderConfig) ProviderInfo {
	if !cfg.Enabled {
		return ProviderInfo{
			Provider: config.ProviderRemote,
			Model:    strings.TrimSpace(cfg.Model),
			Ready:    false,
			Message:  "remote provider disabled",
		}
	}

	embedder, info, err := NewRemoteEmbedderFromConfig(cfg)
	if err != nil {
		return ProviderInfo{
			Provider: config.ProviderRemote,
			Model:    strings.TrimSpace(cfg.Model),
			Ready:    false,
			Message:  err.Error(),
		}
	}

	info.Ready = true
	info.Message = fmt.Sprintf("remote provider ready (%d dimensions)", info.Dimensions)
	_ = embedder
	return info
}

func (r *RemoteEmbedder) detectDimensions() (int, error) {
	vector, err := r.EmbedPassage("vectos healthcheck")
	if err != nil {
		return 0, fmt.Errorf("remote provider probe failed: %w", err)
	}
	if len(vector) == 0 {
		return 0, fmt.Errorf("remote provider probe returned empty embedding")
	}
	return len(vector), nil
}

// EmbedQuery embeds a single search query via the remote API.
func (r *RemoteEmbedder) EmbedQuery(text string) ([]float32, error) {
	vecs, err := r.embed([]string{text})
	if err != nil {
		return nil, err
	}
	return vecs[0], nil
}

// EmbedPassage embeds a single passage for indexing via the remote API.
func (r *RemoteEmbedder) EmbedPassage(text string) ([]float32, error) {
	vecs, err := r.embed([]string{text})
	if err != nil {
		return nil, err
	}
	return vecs[0], nil
}

// EmbedQueries embeds multiple search queries in one remote API call.
func (r *RemoteEmbedder) EmbedQueries(texts []string) ([][]float32, error) {
	return r.embed(texts)
}

// EmbedPassages embeds multiple passages in one remote API call.
func (r *RemoteEmbedder) EmbedPassages(texts []string) ([][]float32, error) {
	return r.embed(texts)
}

func (r *RemoteEmbedder) embed(texts []string) ([][]float32, error) {
	reqBody := EmbeddingRequest{
		Input: texts,
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
		return nil, r.wrapRequestError(url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, describeRemoteStatus(url, resp)
	}

	var embResp EmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&embResp); err != nil {
		return nil, fmt.Errorf("embedding API returned an invalid JSON response: %w", err)
	}

	if len(embResp.Data) == 0 {
		return nil, fmt.Errorf("received empty embedding response")
	}

	results := make([][]float32, len(embResp.Data))
	for i, d := range embResp.Data {
		if len(d.Embedding) == 0 {
			return nil, fmt.Errorf("received empty embedding at index %d", i)
		}
		results[i] = d.Embedding
	}

	return results, nil
}

func (r *RemoteEmbedder) wrapRequestError(endpoint string, err error) error {
	var urlErr *url.Error
	if errors.As(err, &urlErr) && urlErr.Timeout() {
		return fmt.Errorf("embedding API request to %s timed out after %v; check the remote provider or increase embeddings.remote.timeout_seconds: %w", endpoint, r.httpClient.Timeout, err)
	}
	if errors.As(err, &urlErr) {
		return fmt.Errorf("failed to reach embedding API at %s: %w", endpoint, err)
	}
	return fmt.Errorf("failed to reach embedding API at %s: %w", endpoint, err)
}

func describeRemoteStatus(endpoint string, resp *http.Response) error {
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
	detail := strings.TrimSpace(string(body))
	if detail != "" {
		detail = fmt.Sprintf(": %s", detail)
	}

	switch resp.StatusCode {
	case http.StatusUnauthorized, http.StatusForbidden:
		return fmt.Errorf("embedding API rejected the request (%s); check embeddings.remote.base_url and upstream authentication%s", resp.Status, detail)
	case http.StatusNotFound:
		return fmt.Errorf("embedding API endpoint not found at %s; check embeddings.remote.base_url%s", endpoint, detail)
	case http.StatusTooManyRequests:
		return fmt.Errorf("embedding API rate limited the request (%s); retry later or lower request concurrency%s", resp.Status, detail)
	default:
		if resp.StatusCode >= 500 {
			return fmt.Errorf("embedding API is temporarily unavailable (%s)%s", resp.Status, detail)
		}
		return fmt.Errorf("embedding API returned %s%s", resp.Status, detail)
	}
}
