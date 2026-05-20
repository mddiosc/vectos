package main

import (
	"database/sql"
	"errors"
	"fmt"
	"os"

	"vectos/internal/embeddings"
	"vectos/internal/storage"
)

type syncIndexMetadataResult struct {
	Message     string
	FullRebuild bool
}

func syncIndexMetadata(store *storage.SQLiteStorage, providerInfo embeddings.ProviderInfo, indexFingerprint string) (syncIndexMetadataResult, error) {
	prev, err := store.GetIndexMetadata()
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return syncIndexMetadataResult{}, fmt.Errorf("check index metadata: %w", err)
	}

	result := syncIndexMetadataResult{}
	if err == nil {
		embeddingChanged := prev.Provider != providerInfo.Provider || prev.Model != providerInfo.Model || prev.Dimensions != providerInfo.Dimensions
		fingerprintChanged := prev.IndexFingerprint != indexFingerprint

		switch {
		case fingerprintChanged:
			result.FullRebuild = true
			result.Message = fmt.Sprintf("Index format changed (%q → %q) — rebuilding full index", prev.IndexFingerprint, indexFingerprint)
			if prev.IndexFingerprint == "" {
				result.Message = fmt.Sprintf("Legacy index detected — rebuilding full index with format %q", indexFingerprint)
			}
			if err := store.ClearIndexedData(); err != nil {
				return syncIndexMetadataResult{}, fmt.Errorf("clear stale index data: %w", err)
			}
			if err := removeVectorIndexFile(store); err != nil {
				return syncIndexMetadataResult{}, err
			}
		case embeddingChanged:
			result.Message = fmt.Sprintf("Embedding model changed (%s/%dd → %s/%dd) — invalidating existing embeddings", prev.Model, prev.Dimensions, providerInfo.Model, providerInfo.Dimensions)
			if err := store.InvalidateEmbeddings(); err != nil {
				return syncIndexMetadataResult{}, fmt.Errorf("invalidate stale embeddings: %w", err)
			}
			if err := removeVectorIndexFile(store); err != nil {
				return syncIndexMetadataResult{}, err
			}
		}
	}

	if err := store.SetIndexMetadata(storage.IndexMetadata{
		Provider:         providerInfo.Provider,
		Model:            providerInfo.Model,
		Dimensions:       providerInfo.Dimensions,
		IndexFingerprint: indexFingerprint,
	}); err != nil {
		return syncIndexMetadataResult{}, fmt.Errorf("save index metadata: %w", err)
	}

	return result, nil
}

func removeVectorIndexFile(store *storage.SQLiteStorage) error {
	if err := os.Remove(store.VectorIndexPath()); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove stale vector index: %w", err)
	}
	return nil
}
