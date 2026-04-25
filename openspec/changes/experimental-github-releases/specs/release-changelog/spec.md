## ADDED Requirements

### Requirement: Vectos SHALL maintain a human-readable release changelog
Vectos SHALL maintain a changelog file that records the user-visible contents and limitations of each experimental/internal release.

#### Scenario: Record an experimental release
- **WHEN** a new experimental/internal release is prepared
- **THEN** the changelog SHALL include an entry for that release version with categorized notes and known limitations

#### Scenario: Keep changelog format predictable
- **WHEN** a user or maintainer reads the changelog
- **THEN** each release entry SHALL use a consistent section structure so release notes can be curated without guessing the format
