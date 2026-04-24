## Why

Vectos currently depends on a single hardcoded remote embedding endpoint. That is too rigid for a local-first tool whose value proposition includes privacy, offline resilience, and portability across developer setups.

The most important product requirement is that Vectos must work standalone without requiring the user to supply an external embeddings provider. Embedded local embeddings are not just one provider option, they are the default operating mode the tool must support out of the box.

## What Changes

Introduce a provider strategy for embeddings with an embedded local runtime as the default option and an OpenAI-compatible URL provider as an alternative.

- **New Capability: `embedding-provider`**: Select and execute embeddings through an embedded local provider or a remote URL provider.
- **New Capability: `provider-configuration`**: Configure provider selection, model details, and fallback behavior.
- **Embedded-first runtime**: Ship Vectos with an in-process embedded embeddings runtime so indexing and semantic search work without an externally managed provider.
- **Index compatibility metadata**: Track the embedding provider/model used for each index so Vectos can detect when reindexing is required.

## Capabilities

### New Capabilities
- `embedding-provider`: Supports local embedded embeddings and remote OpenAI-compatible embedding endpoints.
- `provider-configuration`: Allows users to choose provider type and configure priority/fallback behavior.

## Impact

- New configuration surface for provider selection.
- New embedded runtime integration path and local model management workflow.
- Refactoring of current embedding client code into provider implementations.
- New project index metadata to track embedding-space compatibility.
