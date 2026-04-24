## 1. File Support

- [x] 1.1 Add detection for `.json`, `.sh`, `.md`, `.toml`, `.ini`, `.xml`, `.properties`, `Makefile`, and `.gitignore`
- [x] 1.2 Add baseline chunking for the new common project file types
- [x] 1.3 Exclude secret-prone env files from this first phase while allowing future safe examples later

## 2. Categorization

- [x] 2.1 Expand file categorization to include `scripts`, `docs`, and `dependency_metadata`
- [x] 2.2 Classify common dependency and project metadata files into the right categories
- [x] 2.3 Preserve the new category context in indexing and search output

## 3. Documentation and Validation

- [x] 3.1 Document the new supported file types and categories
- [x] 3.2 Validate indexing and search against a mixed project containing the new file types
