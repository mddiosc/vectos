## ADDED Requirements

### Requirement: Documentation files SHALL be indexable into a separate database
The system SHALL support indexing documentation files (markdown, mdx, rst, asciidoc, latex, plain text) into a dedicated database file separate from the source code index.

#### Scenario: Index documentation with --docs flag
- **WHEN** a user or agent calls `index_project` with `docs: true`
- **THEN** the system SHALL index only files with `category == "docs"` and store them in `<name>-docs.db`

#### Scenario: Documentation index created automatically
- **WHEN** `index_project` is called with `docs: true` and the documentation database does not exist
- **THEN** the system SHALL create `<name>-docs.db` and populate it with documentation chunks

#### Scenario: Documentation language types supported
- **WHEN** the system indexes documentation files
- **THEN** it SHALL support: markdown (`.md`), MDX (`.mdx`), reStructuredText (`.rst`), AsciiDoc (`.adoc`), LaTeX (`.tex`), and plain text (`.txt`)

#### Scenario: Incremental documentation refresh
- **WHEN** `index_project` is called with `docs: true` and `changed` parameter
- **THEN** the system SHALL update only the changed documentation files, deleting prior chunks for those paths before saving new ones

#### Scenario: Documentation index coexists with source index
- **WHEN** both `index_project` and `index_project docs: true` are called for the same project
- **THEN** the source index SHALL remain in `<name>.db` and the documentation index in `<name>-docs.db`, independent of each other