## ADDED Requirements

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
