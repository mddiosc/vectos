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

	mergeEmbeddedConfig(&dst.Embedded, src.Embedded)
	mergeRemoteConfig(&dst.Remote, src.Remote)
}

func mergeEmbeddedConfig(dst *EmbeddedProviderConfig, src embeddedProviderConfigDisk) {
	if src.ModelName != nil && strings.TrimSpace(*src.ModelName) != "" {
		dst.ModelName = strings.TrimSpace(*src.ModelName)
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
	if src.AssetBaseURL != nil && strings.TrimSpace(*src.AssetBaseURL) != "" {
		dst.AssetBaseURL = strings.TrimSpace(*src.AssetBaseURL)
	}
	if src.TimeoutS != nil && *src.TimeoutS > 0 {
		dst.TimeoutS = *src.TimeoutS
	}
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
