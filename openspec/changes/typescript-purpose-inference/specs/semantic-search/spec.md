## MODIFIED Requirements

### Requirement: Embedding input SHALL be semantically enriched
The system SHALL generate embeddings from enriched chunk content that includes structural metadata in addition to raw code. The enrichment SHALL include TypeScript-specific structural tags (type definition, enumeration, async function) when the chunk content contains those constructs.

#### Scenario: Enrich a Go function chunk
- **WHEN** the system creates an embedding for a Go function chunk
- **THEN** it SHALL include contextual metadata such as file name, language, function signature, and inferred purpose along with the raw code

#### Scenario: Enrich a TypeScript interface chunk
- **WHEN** the system creates an embedding for a TypeScript chunk containing an interface declaration
- **THEN** it SHALL include the "type definition" tag in the chunk purpose and include it in the embedding input text

#### Scenario: Enrich a TypeScript enum chunk
- **WHEN** the system creates an embedding for a TypeScript chunk containing an enum declaration
- **THEN** it SHALL include the "enumeration" tag in the chunk purpose and include it in the embedding input text

#### Scenario: Enrich a TypeScript or React structural chunk
- **WHEN** the system creates an embedding for a TypeScript or React chunk with a recognized structural role (component, hook, type definition, enumeration, async function, etc.)
- **THEN** it SHALL include contextual metadata such as file name, language, chunk role, and detected signature when available in addition to the raw code
