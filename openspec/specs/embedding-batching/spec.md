## ADDED Requirements

### Requirement: Embedder SHALL support batch embedding generation
The `Embedder` interface SHALL provide a method to generate embeddings for multiple texts in a single call, reducing per-call overhead during indexing.

#### Scenario: Batch generate embeddings for multiple chunks
- **WHEN** the embedder receives a slice of text strings
- **THEN** it SHALL return a corresponding slice of embedding vectors in the same order

#### Scenario: Batch with single text works
- **WHEN** the embedder receives a slice containing a single text
- **THEN** it SHALL return a slice containing a single embedding, producing the same result as the single-text method

#### Scenario: Empty batch returns empty result
- **WHEN** the embedder receives an empty text slice
- **THEN** it SHALL return an empty embedding slice without error

### Requirement: Embedded ONNX embedder SHALL batch tokenized inputs
The local ONNX embedder SHALL combine multiple tokenized texts into a single batched inference pass when generating embeddings in batch mode.

#### Scenario: Batch inference with ONNX
- **WHEN** the embedded embedder receives N texts for batch embedding
- **THEN** it SHALL tokenize all texts, pad to the maximum sequence length in the batch, and run a single ONNX inference with shape `[N, max_seq_len]`, then extract and normalize each output vector

#### Scenario: Batch preserves per-text semantics
- **WHEN** the same text is embedded via batch and via single-call
- **THEN** the resulting vectors SHALL be identical (within floating-point tolerance)

### Requirement: Remote embedder SHALL use multi-input API for batching
The remote HTTP embedder SHALL send all texts in a single request using the existing `Input []string` field of the OpenAI-compatible embeddings API.

#### Scenario: Batch request to remote provider
- **WHEN** the remote embedder receives N texts for batch embedding
- **THEN** it SHALL send a single POST request with all N texts in the `input` field and return all N embeddings from the response

### Requirement: Indexing pipeline SHALL embed chunks in configurable batch sizes
The indexing pipeline SHALL group chunks into batches of a configurable size (default 32) before calling the batch embedder.

#### Scenario: Chunks batched during indexing
- **WHEN** indexing produces more chunks than the configured batch size (default 32)
- **THEN** the system SHALL call the embedder in groups of at most that batch size, processing all remaining chunks in a final partial batch

#### Scenario: Batch size is configurable
- **WHEN** the user sets `batch_size` in the embedding configuration
- **THEN** the system SHALL use that value as the maximum number of texts per embedding call
