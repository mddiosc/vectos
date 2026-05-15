## ADDED Requirements

### Requirement: Vectos SHALL support a code-aware embedding model
The system SHALL support the `jina-embeddings-v2-base-code` model as an embedded embedding provider option. This model SHALL produce code-aware embeddings that capture programming language syntax, structural patterns, and semantic meaning of code beyond general-purpose text embeddings.

#### Scenario: Code-aware model is the default provider
- **WHEN** Vectos is configured with default settings and no explicit model override
- **THEN** the embedded provider SHALL use `jina-embeddings-v2-base-code` as the default model

#### Scenario: Code-aware model downloads assets automatically
- **WHEN** the embedded provider is configured with `model_name: "jina-embeddings-v2-base-code"` and `auto_download: true`
- **THEN** the system SHALL download required assets (model.onnx, tokenizer.json, config.json) from the configured asset base URL into the model directory

#### Scenario: Code-aware model creates valid embedding vectors
- **WHEN** the embedded provider generates embeddings using the code-aware model
- **THEN** the output vectors SHALL have 768 dimensions and be normalized to unit length

#### Scenario: Code search uses code-aware embeddings
- **WHEN** a project is indexed with the code-aware model and a semantic search query is executed
- **THEN** the search SHALL use the code-aware embedding space for vector similarity ranking

### Requirement: Vectos SHALL maintain backward compatibility with the general-purpose model
The system SHALL continue to support `bge-small-en-v1.5` as a valid embedded model option. Users who prefer the smaller, faster model may configure it explicitly.

#### Scenario: User configures bge-small explicitly
- **WHEN** the user sets `model_name: "bge-small-en-v1.5"` in the embedded provider configuration
- **THEN** the system SHALL use bge-small for embeddings with 384-dimensional vectors

#### Scenario: bge-small assets download correctly
- **WHEN** a user switches from the default code-aware model to bge-small
- **THEN** the system SHALL download bge-small model assets if not already present in the model directory

### Requirement: Model dimensions SHALL be detected from the ONNX model at runtime
The system SHALL detect the output embedding dimensions from the ONNX model metadata rather than relying on hardcoded constants. The detected dimension SHALL be reflected in the provider status and stored in index metadata.

#### Scenario: Dimension is detected from code-aware model
- **WHEN** the code-aware model ONNX session is created
- **THEN** the system SHALL inspect the output shape and set the provider status dimensions to the detected value (768)

#### Scenario: Dimension is detected from bge-small model
- **WHEN** the bge-small model ONNX session is created
- **THEN** the system SHALL inspect the output shape and set the provider status dimensions to the detected value (384)

#### Scenario: Index metadata records provider and model
- **WHEN** a project is indexed
- **THEN** the system SHALL store the embedding provider name, model name, and detected dimensions in the project's index metadata

#### Scenario: Reindex is required when model changes
- **WHEN** a previously indexed project is searched with a different embedding model (different name or dimensions)
- **THEN** the system SHALL detect the mismatch and warn that reindexing is required for accurate semantic search results

### Requirement: Code-aware model tokenizer SHALL support longer sequences
The code-aware model SHALL support input sequences of at least 8192 tokens, allowing large functions, components, or files to be embedded without truncation that would lose context.

#### Scenario: Large function is embedded without truncation
- **WHEN** a source code chunk exceeds 512 tokens but is under 8192 tokens
- **THEN** the code-aware model SHALL embed the full content without truncating the input

#### Scenario: Sequence longer than model limit is truncated safely
- **WHEN** a source code chunk exceeds the model's maximum sequence length
- **THEN** the system SHALL truncate to the model limit and generate an embedding from the truncated content
