package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	ProviderEmbedded = "embedded"
	ProviderRemote   = "remote"

	DefaultEmbeddedModel = "bge-small-en-v1.5"
	DefaultEmbeddedAssetBaseURL = "https://huggingface.co/BAAI/bge-small-en-v1.5/resolve/main"
	DefaultRemoteModel   = "text-embedding-nomic-embed-text-v1.5"
)

type EmbeddingConfig struct {
	DefaultProvider string                 `json:"default_provider"`
	FallbackOrder   []string               `json:"fallback_order,omitempty"`
	Embedded        EmbeddedProviderConfig `json:"embedded"`
	Remote          RemoteProviderConfig   `json:"remote"`
}

type EmbeddedProviderConfig struct {
	Enabled      bool   `json:"enabled"`
	ModelName    string `json:"model_name"`
	ModelDir     string `json:"model_dir"`
	AutoDownload bool   `json:"auto_download,omitempty"`
	AssetBaseURL string `json:"asset_base_url,omitempty"`
	TimeoutS     int    `json:"timeout_seconds,omitempty"`
}

type RemoteProviderConfig struct {
	Enabled  bool   `json:"enabled"`
	BaseURL  string `json:"base_url"`
	Model    string `json:"model"`
	TimeoutS int    `json:"timeout_seconds"`
}

type embeddingConfigDisk struct {
	DefaultProvider *string                    `json:"default_provider"`
	FallbackOrder   []string                   `json:"fallback_order,omitempty"`
	Embedded        embeddedProviderConfigDisk `json:"embedded"`
	Remote          remoteProviderConfigDisk   `json:"remote"`
}

type embeddedProviderConfigDisk struct {
	Enabled      *bool   `json:"enabled"`
	ModelName    *string `json:"model_name"`
	ModelDir     *string `json:"model_dir"`
	AutoDownload *bool   `json:"auto_download,omitempty"`
	AssetBaseURL *string `json:"asset_base_url,omitempty"`
	TimeoutS     *int    `json:"timeout_seconds,omitempty"`
}

type remoteProviderConfigDisk struct {
	Enabled  *bool   `json:"enabled"`
	BaseURL  *string `json:"base_url"`
	Model    *string `json:"model"`
	TimeoutS *int    `json:"timeout_seconds"`
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

	mergeEmbeddingConfig(&config, disk.Embeddings)
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
		},
		Remote: RemoteProviderConfig{
			Enabled:  false,
			BaseURL:  "",
			Model:    DefaultRemoteModel,
			TimeoutS: 30,
		},
	}
}

func mergeEmbeddingConfig(dst *EmbeddingConfig, src embeddingConfigDisk) {
	if src.DefaultProvider != nil && strings.TrimSpace(*src.DefaultProvider) != "" {
		dst.DefaultProvider = strings.TrimSpace(*src.DefaultProvider)
	}
	if len(src.FallbackOrder) > 0 {
		dst.FallbackOrder = src.FallbackOrder
	}

	if src.Embedded.ModelName != nil && strings.TrimSpace(*src.Embedded.ModelName) != "" {
		dst.Embedded.ModelName = strings.TrimSpace(*src.Embedded.ModelName)
	}
	if src.Embedded.ModelDir != nil && strings.TrimSpace(*src.Embedded.ModelDir) != "" {
		dst.Embedded.ModelDir = strings.TrimSpace(*src.Embedded.ModelDir)
	}
	if src.Embedded.Enabled != nil {
		dst.Embedded.Enabled = *src.Embedded.Enabled
	}
	if src.Embedded.AutoDownload != nil {
		dst.Embedded.AutoDownload = *src.Embedded.AutoDownload
	}
	if src.Embedded.AssetBaseURL != nil && strings.TrimSpace(*src.Embedded.AssetBaseURL) != "" {
		dst.Embedded.AssetBaseURL = strings.TrimSpace(*src.Embedded.AssetBaseURL)
	}
	if src.Embedded.TimeoutS != nil && *src.Embedded.TimeoutS > 0 {
		dst.Embedded.TimeoutS = *src.Embedded.TimeoutS
	}

	if src.Remote.BaseURL != nil && strings.TrimSpace(*src.Remote.BaseURL) != "" {
		dst.Remote.BaseURL = strings.TrimSpace(*src.Remote.BaseURL)
	}
	if src.Remote.Model != nil && strings.TrimSpace(*src.Remote.Model) != "" {
		dst.Remote.Model = strings.TrimSpace(*src.Remote.Model)
	}
	if src.Remote.TimeoutS != nil && *src.Remote.TimeoutS > 0 {
		dst.Remote.TimeoutS = *src.Remote.TimeoutS
	}
	if src.Remote.Enabled != nil {
		dst.Remote.Enabled = *src.Remote.Enabled
	}
}
