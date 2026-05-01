## MODIFIED Requirements

### Requirement: Project code can be indexed from files and directories
The system SHALL accept a file path or project directory and index supported source files into searchable code chunks.

#### Scenario: Index only documentation files
- **WHEN** a user or agent requests indexing with `docs: true`
- **THEN** the system SHALL index only files where `category == "docs"` (markdown, mdx, rst, asciidoc, latex, plain text) into the documentation database

#### Scenario: Documentation language types supported for indexing
- **WHEN** the system indexes documentation files
- **THEN** it SHALL recognize and process: markdown (`.md`), MDX (`.mdx`), reStructuredText (`.rst`), AsciiDoc (`.adoc`), LaTeX (`.tex`), and plain text (`.txt`) files