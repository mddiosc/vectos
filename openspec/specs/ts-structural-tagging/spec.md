## Purpose
Define how the chunker detects and tags TypeScript-specific structural constructs (interfaces, type aliases, enums, async functions) to enrich code embeddings with structural role metadata.

## Requirements

### Requirement: TypeScript interface declarations SHALL be tagged as type definitions
When the chunker processes a TypeScript or TSX chunk containing an `interface` declaration, the system SHALL detect it and tag the chunk with "type definition" in its purpose metadata.

#### Scenario: Interface chunk receives type definition tag
- **WHEN** a chunk contains `interface UserProps { name: string; email: string }`
- **THEN** the chunk's purpose SHALL include "type definition"

#### Scenario: Exported interface also receives type definition tag
- **WHEN** a chunk contains `export interface ApiResponse<T> { data: T; error?: string }`
- **THEN** the chunk's purpose SHALL include both "type definition" and "exported api"

### Requirement: TypeScript type aliases SHALL be tagged as type definitions
When the chunker processes a TypeScript chunk containing a `type` alias declaration, the system SHALL detect it and tag the chunk with "type definition" in its purpose metadata.

#### Scenario: Type alias chunk receives type definition tag
- **WHEN** a chunk contains `type UserRole = "admin" | "editor" | "viewer"`
- **THEN** the chunk's purpose SHALL include "type definition"

#### Scenario: Generic type alias receives type definition tag
- **WHEN** a chunk contains `type ApiResponse<T> = { data: T; error?: string }`
- **THEN** the chunk's purpose SHALL include "type definition"

### Requirement: TypeScript enum declarations SHALL be tagged as enumerations
When the chunker processes a TypeScript chunk containing an `enum` declaration, the system SHALL detect it and tag the chunk with "enumeration" in its purpose metadata.

#### Scenario: Enum chunk receives enumeration tag
- **WHEN** a chunk contains `enum Status { Active, Inactive, Pending }`
- **THEN** the chunk's purpose SHALL include "enumeration"

#### Scenario: Const enum receives enumeration tag
- **WHEN** a chunk contains `const enum Direction { Up, Down, Left, Right }`
- **THEN** the chunk's purpose SHALL include "enumeration"

### Requirement: Async function declarations SHALL be tagged as async functions
When the chunker processes a JavaScript or TypeScript chunk containing an `async function` declaration or `async` arrow function, the system SHALL detect it and tag the chunk with "async function" in its purpose metadata.

#### Scenario: Async function chunk receives async tag
- **WHEN** a chunk contains `async function fetchUserData(id: string): Promise<User>`
- **THEN** the chunk's purpose SHALL include "async function"

#### Scenario: Async arrow function receives async tag
- **WHEN** a chunk contains `const fetchData = async (url: string) => { ... }`
- **THEN** the chunk's purpose SHALL include "async function"

#### Scenario: Chunk with both component and async gets both tags
- **WHEN** a chunk contains `export default async function UserProfile({ id }: Props)`
- **THEN** the chunk's purpose SHALL include both "react component" and "async function"

### Requirement: Existing purpose tags SHALL NOT be affected
The new detection SHALL be additive — chunks that already receive purpose tags (react component, custom hook, test block, etc.) SHALL continue to receive those tags. Chunks without any structural role SHALL still receive the "code block" fallback tag.

#### Scenario: Regular function still gets function tag
- **WHEN** a chunk contains a non-async, non-component function like `function formatDate(date: Date): string`
- **THEN** the chunk SHALL receive "function or callable block" as before, without the new tags
