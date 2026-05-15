## 1. Discovery & Verification

- [x] 1.1 Use `vectos_search_code` with query "internal/mcp" to confirm no active references
- [x] 1.2 Grep for `"internal/mcp"` import paths across the entire codebase
- [x] 1.3 Grep for `inferGoPurpose` to locate all occurrences (definition + call sites)
- [x] 1.4 Read `inferGoPurpose` function body to confirm it is a no-op stub
- [x] 1.5 Run `staticcheck ./...` to identify additional unreferenced exports and dead code

## 2. Remove internal/mcp/

- [x] 2.1 Delete the entire `internal/mcp/` directory
- [x] 2.2 Run `go vet ./...` to verify no broken imports
- [x] 2.3 Run `go build ./...` to confirm the project compiles

## 3. Remove inferGoPurpose

- [x] 3.1 Remove the `inferGoPurpose` function definition
- [x] 3.2 Remove any test code or helpers that only tested `inferGoPurpose`
- [x] 3.3 Run `go vet ./...` and `go build ./...` to verify no breakage

## 4. Additional Dead Code Scan

- [x] 4.1 Review `staticcheck` output for other dead code (unreferenced exported functions, unused types, orphaned test helpers)
- [x] 4.2 For each candidate: verify it is truly unreferenced (grep + vectos_search_code)
- [x] 4.3 Remove confirmed dead code, excluding any ambiguous/WIP candidates
- [x] 4.4 Run `go vet ./...` and `go build ./...` after each removal batch

## 5. Final Validation

- [x] 5.1 Run `go vet ./...` — must pass with zero warnings
- [x] 5.2 Run `go build ./...` — must compile cleanly
- [x] 5.3 Run `staticcheck ./...` — must pass (no new warnings from removed code)
- [x] 5.4 Run `go test ./...` — all existing tests must pass (no behavioral regressions)
