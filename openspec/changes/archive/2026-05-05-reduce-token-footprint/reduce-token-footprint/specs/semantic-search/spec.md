## MODIFIED Requirements

### Requirement: Embedding input SHALL be semantically enriched
The system SHALL generate embeddings from enriched chunk content that includes structural metadata in addition to raw code.

#### Scenario: Enrich a Go function chunk
- **WHEN** the system creates an embedding for a Go function chunk
- **THEN** it SHALL include contextual metadata such as file name, function signature, and inferred purpose along with the raw code — without explicit language or category labels

#### Scenario: Enrich a TypeScript or React structural chunk
- **WHEN** the system creates an embedding for a TypeScript or React chunk with a recognized structural role
- **THEN** it SHALL include contextual metadata such as file name, detected signature, and chunk role when available in addition to the raw code — without explicit language or category labels

#### Scenario: Enrichment omits redundant metadata
- **WHEN** the system builds the enrichment string for a code chunk
- **THEN** it SHALL NOT include explicit `Language:` or `Category:` prefix lines, since the embedding model can infer language from code structure and the category is derivable from language at query time