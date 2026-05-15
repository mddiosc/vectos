package config

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

const (
	ProviderEmbedded = "embedded"
	ProviderRemote   = "remote"

	DefaultEmbeddedModel          = "jina-embeddings-v3"
	DefaultEmbeddedAssetBaseURL   = "https://huggingface.co/jinaai/jina-embeddings-v3/resolve/main"
	DefaultBGEAssetBaseURL        = "https://huggingface.co/BAAI/bge-small-en-v1.5/resolve/main"
	DefaultRemoteModel            = "text-embedding-nomic-embed-text-v1.5"
)

// SupportedEmbeddedModels lists the model names accepted in embedded model_name config.
var SupportedEmbeddedModels = []string{"jina-embeddings-v3", "bge-small-en-v1.5"}

// ValidateAssetBaseURL validates the embedded provider's asset_base_url.
// Returns nil if the URL is empty; otherwise checks scheme (HTTPS only),
// path traversal (no ".."), non-empty host, and max length (2048 chars).
func ValidateAssetBaseURL(raw string) error {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil
	}
	if len(trimmed) > 2048 {
		return fmt.Errorf("asset_base_url exceeds maximum length of 2048 characters")
	}
	if strings.Contains(trimmed, "..") {
		return fmt.Errorf("asset_base_url must not contain path traversal segments")
	}
	u, err := url.Parse(trimmed)
	if err != nil {
		return fmt.Errorf("asset_base_url is not a valid URL: %w", err)
	}
	if u.Scheme != "https" {
		return fmt.Errorf("asset_base_url must use HTTPS scheme, got %q", u.Scheme)
	}
	if u.Host == "" {
		return fmt.Errorf("asset_base_url must have a non-empty host")
	}
	return nil
}

type EmbeddingConfig struct {
	DefaultProvider string                 `json:"default_provider"`
	FallbackOrder   []string               `json:"fallback_order,omitempty"`
	Embedded        EmbeddedProviderConfig `json:"embedded"`
	Remote          RemoteProviderConfig   `json:"remote"`
	VectorIndex     VectorIndexConfig      `json:"vector_index"`
}

type EmbeddedProviderConfig struct {
	Enabled      bool   `json:"enabled"`
	ModelName    string `json:"model_name"`
	ModelDir     string `json:"model_dir"`
	AutoDownload bool   `json:"auto_download,omitempty"`
	AssetBaseURL string `json:"asset_base_url,omitempty"`
	TimeoutS     int    `json:"timeout_seconds,omitempty"`
	BatchSize    int    `json:"batch_size,omitempty"`
}

// DefaultEmbeddedBatchSize is the number of texts to embed in a single
// inference call when the user does not configure batch_size explicitly.
const DefaultEmbeddedBatchSize = 32

type RemoteProviderConfig struct {
	Enabled  bool   `json:"enabled"`
	BaseURL  string `json:"base_url"`
	Model    string `json:"model"`
	TimeoutS int    `json:"timeout_seconds"`
}

// VectorIndexConfig holds configuration for the approximate nearest neighbor index.
type VectorIndexConfig struct {
	IndexType           string `json:"index_type"`
	HNSW_M              int    `json:"hnsw_m"`
	HNSW_EfConstruction int    `json:"hnsw_ef_construction"`
	HNSW_EfSearch       int    `json:"hnsw_ef_search"`
	Compression         string `json:"compression,omitempty"`
}

type embeddingConfigDisk struct {
	DefaultProvider *string                    `json:"default_provider"`
	FallbackOrder   []string                   `json:"fallback_order,omitempty"`
	Embedded        embeddedProviderConfigDisk `json:"embedded"`
	Remote          remoteProviderConfigDisk   `json:"remote"`
	VectorIndex     vectorIndexConfigDisk      `json:"vector_index"`
}

type embeddedProviderConfigDisk struct {
	Enabled      *bool   `json:"enabled"`
	ModelName    *string `json:"model_name"`
	ModelDir     *string `json:"model_dir"`
	AutoDownload *bool   `json:"auto_download,omitempty"`
	AssetBaseURL *string `json:"asset_base_url,omitempty"`
	TimeoutS     *int    `json:"timeout_seconds,omitempty"`
	BatchSize    *int    `json:"batch_size,omitempty"`
}

type remoteProviderConfigDisk struct {
	Enabled  *bool   `json:"enabled"`
	BaseURL  *string `json:"base_url"`
	Model    *string `json:"model"`
	TimeoutS *int    `json:"timeout_seconds"`
}

type vectorIndexConfigDisk struct {
	IndexType           *string `json:"index_type"`
	HNSW_M              *int    `json:"hnsw_m"`
	HNSW_EfConstruction *int    `json:"hnsw_ef_construction"`
	HNSW_EfSearch       *int    `json:"hnsw_ef_search"`
	Compression         *string `json:"compression,omitempty"`
}

func LoadEmbeddingConfig(homeDir string) (EmbeddingConfig, error) {
	config := DefaultEmbeddingConfig(homeDir)
	configPath := filepath.Join(homeDir, ".vectos", "config.json")

	content, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return config, nil
		}
		return EmbeddingConfig{}, fmt.Errorf("failed to read vectos config: %w", err)
	}

	var disk struct {
		Embeddings embeddingConfigDisk `json:"embeddings"`
	}
	if err := json.Unmarshal(content, &disk); err != nil {
		return EmbeddingConfig{}, fmt.Errorf("failed to parse vectos config: %w", err)
	}

	if err := mergeEmbeddingConfig(&config, disk.Embeddings); err != nil {
		return EmbeddingConfig{}, err
	}
	return config, nil
}

func DefaultEmbeddingConfig(homeDir string) EmbeddingConfig {
	modelDir := filepath.Join(homeDir, ".vectos", "models", DefaultEmbeddedModel)
	return EmbeddingConfig{
		DefaultProvider: ProviderEmbedded,
		FallbackOrder:   []string{ProviderEmbedded, ProviderRemote},
		Embedded: EmbeddedProviderConfig{
			Enabled:      true,
			ModelName:    DefaultEmbeddedModel,
			ModelDir:     modelDir,
			AutoDownload: true,
			AssetBaseURL: DefaultEmbeddedAssetBaseURL,
			TimeoutS:     60,
			BatchSize:    DefaultEmbeddedBatchSize,
		},
		Remote: RemoteProviderConfig{
			Enabled:  false,
			BaseURL:  "",
			Model:    DefaultRemoteModel,
			TimeoutS: 30,
		},
		VectorIndex: VectorIndexConfig{
			IndexType:           "hnsw",
			HNSW_M:              16,
			HNSW_EfConstruction: 200,
			HNSW_EfSearch:       100,
			Compression:         "none",
		},
	}
}

func mergeEmbeddingConfig(dst *EmbeddingConfig, src embeddingConfigDisk) error {
	if src.DefaultProvider != nil && strings.TrimSpace(*src.DefaultProvider) != "" {
		dst.DefaultProvider = strings.TrimSpace(*src.DefaultProvider)
	}
	if len(src.FallbackOrder) > 0 {
		dst.FallbackOrder = src.FallbackOrder
	}

	if err := mergeEmbeddedConfig(&dst.Embedded, src.Embedded); err != nil {
		return err
	}
	mergeRemoteConfig(&dst.Remote, src.Remote)
	if err := mergeVectorIndexConfig(&dst.VectorIndex, src.VectorIndex); err != nil {
		return err
	}
	return nil
}

func mergeEmbeddedConfig(dst *EmbeddedProviderConfig, src embeddedProviderConfigDisk) error {
	if src.ModelName != nil && strings.TrimSpace(*src.ModelName) != "" {
		modelName := strings.TrimSpace(*src.ModelName)
		if !isSupportedEmbeddedModel(modelName) {
			return fmt.Errorf("unsupported embedded model %q: must be one of %v", modelName, SupportedEmbeddedModels)
		}
		dst.ModelName = modelName
	}
	if src.ModelDir != nil && strings.TrimSpace(*src.ModelDir) != "" {
		dst.ModelDir = strings.TrimSpace(*src.ModelDir)
	}
	if src.Enabled != nil {
		dst.Enabled = *src.Enabled
	}
	if src.AutoDownload != nil {
		dst.AutoDownload = *src.AutoDownload
	}
	if src.BatchSize != nil && *src.BatchSize > 0 {
		dst.BatchSize = *src.BatchSize
	}
	if src.AssetBaseURL != nil && strings.TrimSpace(*src.AssetBaseURL) != "" {
		trimmed := strings.TrimSpace(*src.AssetBaseURL)
		if err := ValidateAssetBaseURL(trimmed); err != nil {
			return fmt.Errorf("invalid asset_base_url: %w", err)
		}
		dst.AssetBaseURL = trimmed
	}
	if src.TimeoutS != nil && *src.TimeoutS > 0 {
		dst.TimeoutS = *src.TimeoutS
	}
	return nil
}

func mergeRemoteConfig(dst *RemoteProviderConfig, src remoteProviderConfigDisk) {
	if src.BaseURL != nil && strings.TrimSpace(*src.BaseURL) != "" {
		dst.BaseURL = strings.TrimSpace(*src.BaseURL)
	}
	if src.Model != nil && strings.TrimSpace(*src.Model) != "" {
		dst.Model = strings.TrimSpace(*src.Model)
	}
	if src.TimeoutS != nil && *src.TimeoutS > 0 {
		dst.TimeoutS = *src.TimeoutS
	}
	if src.Enabled != nil {
		dst.Enabled = *src.Enabled
	}
}

func mergeVectorIndexConfig(dst *VectorIndexConfig, src vectorIndexConfigDisk) error {
	if src.IndexType != nil && strings.TrimSpace(*src.IndexType) != "" {
		dst.IndexType = strings.TrimSpace(*src.IndexType)
	}
	if src.HNSW_M != nil && *src.HNSW_M > 0 {
		dst.HNSW_M = *src.HNSW_M
	}
	if src.HNSW_EfConstruction != nil && *src.HNSW_EfConstruction > 0 {
		dst.HNSW_EfConstruction = *src.HNSW_EfConstruction
	}
	if src.HNSW_EfSearch != nil && *src.HNSW_EfSearch > 0 {
		dst.HNSW_EfSearch = *src.HNSW_EfSearch
	}
	if src.Compression != nil && strings.TrimSpace(*src.Compression) != "" {
		dst.Compression = strings.TrimSpace(*src.Compression)
	}
	return nil
}

func isSupportedEmbeddedModel(name string) bool {
	for _, m := range SupportedEmbeddedModels {
		if m == name {
			return true
		}
	}
	return false
}
