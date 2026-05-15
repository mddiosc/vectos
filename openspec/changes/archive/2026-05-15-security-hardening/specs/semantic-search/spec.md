## ADDED Requirements

### Requirement: Text search SHALL escape LIKE wildcards in user input
When performing a text-based fallback search using SQL LIKE, the system SHALL escape SQL LIKE wildcard metacharacters (`%` and `_`) in user-provided search terms to prevent unintended wildcard injection and resource-exhaustion attacks.

#### Scenario: Normal search term passes through
- **WHEN** a text search is performed with query `auth middleware`
- **THEN** the search SHALL construct a LIKE pattern `%auth middleware%` and return matching results normally

#### Scenario: Wildcard characters in query are escaped
- **WHEN** a text search is performed with query `100% coverage`
- **THEN** the `%` character SHALL be escaped to `\%` before constructing the LIKE clause, so the search matches literal `100%` text rather than treating `%` as a wildcard

#### Scenario: Underscore in query is escaped
- **WHEN** a text search is performed with query `my_func`
- **THEN** the `_` character SHALL be escaped to `\_` before constructing the LIKE clause, so the search matches literal `my_func` rather than treating `_` as a single-character wildcard

#### Scenario: Denial-of-service pattern is neutralized
- **WHEN** a text search is performed with a query consisting entirely of repeated `%` characters (e.g., `%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%`)
- **THEN** all `%` characters SHALL be escaped to `\%`, preventing the LIKE engine from expanding the pattern combinatorially

#### Scenario: Escape character itself is escaped
- **WHEN** a text search is performed with query containing a backslash (e.g., `path\to\file`)
- **THEN** the backslash character SHALL be escaped to `\\` before constructing the LIKE clause, preventing escape-chain bypass
