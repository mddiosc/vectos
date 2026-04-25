package embeddings

import (
	"fmt"
	"strings"
	"vectos/internal/config"
)

type ProviderInfo struct {
	Provider   string `json:"provider"`
	Model      string `json:"model"`
	Dimensions int    `json:"dimensions"`
	Ready      bool   `json:"ready,omitempty"`
	Message    string `json:"message,omitempty"`
}

func ResolveEmbedder(cfg config.EmbeddingConfig) (Embedder, ProviderInfo, error) {
	providerOrder := buildProviderOrder(cfg)
	var lastErr error

	for _, provider := range providerOrder {
		switch provider {
		case config.ProviderEmbedded:
			embedder, info, err := NewEmbeddedEmbedder(cfg.Embedded)
			if err == nil {
				return embedder, info, nil
			}
			lastErr = err
		case config.ProviderRemote:
			embedder, info, err := NewRemoteEmbedderFromConfig(cfg.Remote)
			if err == nil {
				return embedder, info, nil
			}
			lastErr = err
		}
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("no embedding providers available")
	}

	return nil, ProviderInfo{}, lastErr
}

func InspectProviders(cfg config.EmbeddingConfig) []ProviderInfo {
	providerOrder := buildProviderOrder(cfg)
	statuses := make([]ProviderInfo, 0, len(providerOrder))

	for _, provider := range providerOrder {
		switch provider {
		case config.ProviderEmbedded:
			status := InspectEmbeddedProvider(cfg.Embedded)
			statuses = append(statuses, providerInfoFromStatus(status))
		case config.ProviderRemote:
			statuses = append(statuses, InspectRemoteProvider(cfg.Remote))
		}
	}

	return statuses
}

func buildProviderOrder(cfg config.EmbeddingConfig) []string {
	seen := map[string]bool{}
	var order []string
	appendProvider := func(provider string) {
		provider = strings.TrimSpace(provider)
		if provider == "" || seen[provider] {
			return
		}
		seen[provider] = true
		order = append(order, provider)
	}

	appendProvider(cfg.DefaultProvider)
	for _, provider := range cfg.FallbackOrder {
		appendProvider(provider)
	}
	appendProvider(config.ProviderEmbedded)
	appendProvider(config.ProviderRemote)

	return order
}
