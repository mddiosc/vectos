## Purpose
Define the supported embedding providers and the safeguards around embedded model asset configuration and downloads.
## Requirements
### Requirement: Vectos SHALL support multiple embedding provider types
Vectos SHALL support at least one embedded local provider and one remote URL-based provider for embeddings.

#### Scenario: Use embedded provider by default
- **WHEN** Vectos starts without an explicit provider override
- **THEN** it SHALL attempt to use the embedded provider first

### Requirement: Vectos SHALL support standalone local embeddings
Vectos SHALL be usable for indexing and semantic search without requiring the user to configure or operate an external embeddings provider.

#### Scenario: Index project with no external provider configured
- **WHEN** the user runs Vectos with default configuration and no remote embeddings endpoint configured
- **THEN** Vectos SHALL use its embedded local embeddings runtime to generate vectors

#### Scenario: Search indexed project with embedded provider
- **WHEN** a project index was created with the embedded provider
- **THEN** Vectos SHALL support semantic search against that index without requiring a remote embeddings endpoint

#### Scenario: Use remote provider by configuration
- **WHEN** the user configures a remote provider endpoint
- **THEN** Vectos SHALL generate embeddings through that endpoint instead of the embedded provider

### Requirement: Remote providers SHALL use an OpenAI-compatible embeddings API
Vectos SHALL treat remote URL providers as OpenAI-compatible embeddings endpoints in the initial implementation.

#### Scenario: Connect to local or remote compatible endpoint
- **WHEN** the configured endpoint implements the OpenAI-compatible embeddings contract
- **THEN** Vectos SHALL use it as a valid remote provider

### Requirement: Asset base URL SHALL be validated at configuration time
When the embedded provider is configured with an `asset_base_url`, the system SHALL validate the URL before accepting it. Invalid URLs SHALL cause the configuration to be rejected with an error.

#### Scenario: Valid HTTPS URL is accepted
- **WHEN** the `asset_base_url` is set to a valid HTTPS URL (e.g., `https://cdn.example.com/models`)
- **THEN** the configuration SHALL be accepted and the URL stored without modification

#### Scenario: Non-HTTPS URL is rejected
- **WHEN** the `asset_base_url` is set to an HTTP URL (e.g., `http://cdn.example.com/models`)
- **THEN** the configuration SHALL be rejected with an error indicating HTTPS is required

#### Scenario: URL with path traversal is rejected
- **WHEN** the `asset_base_url` contains `..` path segments (e.g., `https://cdn.example.com/../../etc/`)
- **THEN** the configuration SHALL be rejected with an error indicating invalid URL

#### Scenario: Empty URL is accepted (disables custom base)
- **WHEN** the `asset_base_url` is empty or consists only of whitespace
- **THEN** no validation error SHALL be raised; the field SHALL be treated as unset

#### Scenario: Excessively long URL is rejected
- **WHEN** the `asset_base_url` exceeds 2048 characters
- **THEN** the configuration SHALL be rejected with an error indicating URL is too long

### Requirement: Model asset downloads SHALL verify Content-Type
When the embedded provider downloads model assets (ONNX model, tokenizer, config files) via HTTP, the system SHALL verify that the response Content-Type header matches an expected set of allowed types before writing the downloaded data to disk.

#### Scenario: Allowed Content-Type is accepted
- **WHEN** a model asset download returns Content-Type `application/octet-stream`
- **THEN** the download SHALL proceed and the file SHALL be written to disk

#### Scenario: Gzip Content-Type is accepted
- **WHEN** a model asset download returns Content-Type `application/gzip` or `application/x-gzip`
- **THEN** the download SHALL proceed normally

#### Scenario: Empty Content-Type is accepted
- **WHEN** a model asset download returns no Content-Type header (empty string)
- **THEN** the download SHALL proceed normally (to accommodate CDNs that omit the header)

#### Scenario: Unexpected Content-Type is rejected
- **WHEN** a model asset download returns Content-Type `text/html` (indicating a compromised or misconfigured server)
- **THEN** the download SHALL be aborted and the temporary file SHALL be removed without writing to the final path

#### Scenario: Zero-length response body is rejected
- **WHEN** a model asset download completes successfully but the response body is empty (0 bytes)
- **THEN** the download SHALL be treated as a failure and the asset SHALL NOT be considered downloaded

