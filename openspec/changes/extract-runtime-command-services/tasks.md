## 1. Runtime Command Extraction

- [x] 1.1 Extract indexing runtime logic and closely related helpers out of `cmd/vectos/main.go`
- [x] 1.2 Extract search and status runtime logic into dedicated command-oriented files
- [x] 1.3 Reorganize shared runtime helpers so scope, storage, and path-related logic are easier to find

## 2. Validation

- [x] 2.1 Spot-check current `index`, `search`, and `status` behavior after the refactor
- [x] 2.2 Run `go test ./...`
- [x] 2.3 Run `go build ./...`
