## ADDED Requirements

### Requirement: File walking SHALL reject known sensitive file patterns
During directory walking for indexable file collection, the system SHALL reject files matching known sensitive-file patterns. These files SHALL be skipped and recorded in the skipped-files list.

#### Scenario: .env file is skipped
- **WHEN** the indexer walks a directory containing a file named `.env`
- **THEN** the file SHALL be skipped and added to the skipped-files list with reason "sensitive file"

#### Scenario: .env variant files are skipped
- **WHEN** the indexer encounters files named `.env.local`, `.env.production`, or `.env.development`
- **THEN** those files SHALL be skipped and recorded as sensitive

#### Scenario: SSH private key files are skipped
- **WHEN** the indexer encounters files named `id_rsa`, `id_ecdsa`, `id_ed25519`, or filenames ending with `_rsa`, `_ecdsa`, `_ed25519`
- **THEN** those files SHALL be skipped and recorded as sensitive

#### Scenario: Certificate and credential files are skipped
- **WHEN** the indexer encounters files with extensions `.pem`, `.key`, `.pfx`, `.p12`, or named `credentials.json` or `service-account.json`
- **THEN** those files SHALL be skipped and recorded as sensitive

#### Scenario: Normal source files are still indexed
- **WHEN** the indexer encounters files with standard source extensions (`.go`, `.ts`, `.py`, etc.) that do not match sensitive patterns
- **THEN** those files SHALL be indexed normally without interference from the sensitive-file filter

#### Scenario: .env.example and similar non-sensitive variants are indexed
- **WHEN** the indexer encounters a file named `.env.example` or `.env.sample`
- **THEN** the file SHALL be indexed normally (the filter SHALL use exact-name matching, not prefix matching)
