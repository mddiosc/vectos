## Purpose
Define support for a code-aware embedding model that produces higher-quality vectors for code search, understanding TypeScript/JavaScript/Go syntax patterns in the embedding space.

## Requirements

### Requirement: Vectos SHALL support a code+text-aware embedding model
The system SHALL support the `jina-embeddings-v3` model as the default embedded embedding provider. This model SHALL produce embeddings that capture programming language syntax, structural patterns, and semantic meaning of both code and natural language text.

#### Scenario: Code+text model is the default provider
- **WHEN** Vectos is configured with default settings and no explicit model override
- **THEN** the embedded provider SHALL use `jina-embeddings-v3` as the default model

#### Scenario: Code+text model downloads assets automatically
- **WHEN** the embedded provider is configured with `model_name: "jina-embeddings-v3"` and `auto_download: true`
- **THEN** the system SHALL download required assets (model.onnx, tokenizer.json, config.json) from the configured asset base URL into the model directory

#### Scenario: Code+text model creates valid embedding vectors
- **WHEN** the embedded provider generates embeddings using the code+text model
- **THEN** the output vectors SHALL have 1024 dimensions and be normalized to unit length

#### Scenario: Code search uses code+text embeddings
- **WHEN** a project is indexed with the code+text model and a semantic search query is executed
- **THEN** the search SHALL use the embedding space for vector similarity ranking

### Requirement: Vectos SHALL maintain backward compatibility with the general-purpose model
The system SHALL continue to support `bge-small-en-v1.5` as a valid embedded model option. Users who prefer the smaller, faster model may configure it explicitly.

#### Scenario: User configures bge-small explicitly
- **WHEN** the user sets `model_name: "bge-small-en-v1.5"` in the embedded provider configuration
- **THEN** the system SHALL use bge-small for embeddings with 384-dimensional vectors

#### Scenario: bge-small assets download correctly
- **WHEN** a user switches from the default code+text model to bge-small
- **THEN** the system SHALL download bge-small model assets if not already present in the model directory

### Requirement: Model dimensions SHALL be detected from the ONNX model at runtime
The system SHALL detect the output embedding dimensions from the ONNX model metadata rather than relying on hardcoded constants. The detected dimension SHALL be reflected in the provider status and stored in index metadata.

#### Scenario: Dimension is detected from code+text model
- **WHEN** the code+text model ONNX session is created
- **THEN** the system SHALL inspect the output shape and set the provider status dimensions to the detected value (1024)

#### Scenario: Dimension is detected from bge-small model
- **WHEN** the bge-small model ONNX session is created
- **THEN** the system SHALL inspect the output shape and set the provider status dimensions to the detected value (384)

#### Scenario: Index metadata records provider and model
- **WHEN** a project is indexed
- **THEN** the system SHALL store the embedding provider name, model name, and detected dimensions in the project's index metadata

#### Scenario: Reindex is required when model changes
- **WHEN** a previously indexed project is searched with a different embedding model (different name or dimensions)
- **THEN** the system SHALL detect the mismatch and warn that reindexing is required for accurate semantic search results

### Requirement: Code+text model tokenizer SHALL support longer sequences
The code+text model SHALL support input sequences of at least 8192 tokens, allowing large functions, components, or files to be embedded without truncation that would lose context.

#### Scenario: Large function is embedded without truncation
- **WHEN** a source code chunk exceeds 512 tokens but is under 8192 tokens
- **THEN** the code+text model SHALL embed the full content without truncating the input

#### Scenario: Sequence longer than model limit is truncated safely
- **WHEN** a source code chunk exceeds the model's maximum sequence length
- **THEN** the system SHALL truncate to the model limit and generate an embedding from the truncated content
