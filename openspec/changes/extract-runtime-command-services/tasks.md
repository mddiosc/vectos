## 1. Runtime Command Extraction

- [ ] 1.1 Extract indexing runtime logic and closely related helpers out of `cmd/vectos/main.go`
- [ ] 1.2 Extract search and status runtime logic into dedicated command-oriented files
- [ ] 1.3 Reorganize shared runtime helpers so scope, storage, and path-related logic are easier to find

## 2. Validation

- [ ] 2.1 Spot-check current `index`, `search`, and `status` behavior after the refactor
- [ ] 2.2 Run `go test ./...`
- [ ] 2.3 Run `go build ./...`
