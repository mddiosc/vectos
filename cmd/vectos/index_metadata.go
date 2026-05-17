package main

import (
	"fmt"
	"os"

	"vectos/internal/embeddings"
	"vectos/internal/storage"
)

func syncIndexMetadata(store *storage.SQLiteStorage, providerInfo embeddings.ProviderInfo) (string, error) {
	needsInvalidation, err := store.RequiresReindex(providerInfo.Provider, providerInfo.Model, providerInfo.Dimensions)
	if err != nil {
		return "", fmt.Errorf("check index metadata: %w", err)
	}

	message := ""
	if needsInvalidation {
		if prev, err := store.GetIndexMetadata(); err == nil {
			message = fmt.Sprintf("Embedding model changed (%s/%dd → %s/%dd) — invalidating existing embeddings", prev.Model, prev.Dimensions, providerInfo.Model, providerInfo.Dimensions)
		} else {
			message = "Embedding model changed — invalidating existing embeddings"
		}
		if err := store.InvalidateEmbeddings(); err != nil {
			return "", fmt.Errorf("invalidate stale embeddings: %w", err)
		}
		if err := os.Remove(store.VectorIndexPath()); err != nil && !os.IsNotExist(err) {
			return "", fmt.Errorf("remove stale vector index: %w", err)
		}
	}

	if err := store.SetIndexMetadata(storage.IndexMetadata{
		Provider:   providerInfo.Provider,
		Model:      providerInfo.Model,
		Dimensions: providerInfo.Dimensions,
	}); err != nil {
		return "", fmt.Errorf("save index metadata: %w", err)
	}

	return message, nil
}
