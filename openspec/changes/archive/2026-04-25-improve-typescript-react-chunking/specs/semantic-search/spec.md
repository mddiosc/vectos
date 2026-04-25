## MODIFIED Requirements

### Requirement: Embedding input SHALL be semantically enriched
The system SHALL generate embeddings from enriched chunk content that includes structural metadata in addition to raw code.

#### Scenario: Enrich a TypeScript or React structural chunk
- **WHEN** the system creates an embedding for a TypeScript or React chunk with a recognized structural role
- **THEN** it SHALL include contextual metadata such as file name, language, chunk role, and detected signature when available in addition to the raw code
