## 1. CLI Entry And Dispatch

- [x] 1.1 Extract subcommand help text and flag-set creation out of `cmd/vectos/main.go`
- [x] 1.2 Move command dispatch flow into dedicated CLI wiring helpers while keeping `main.go` as a thin entrypoint
- [x] 1.3 Preserve current argument normalization and help behavior for existing commands

## 2. Validation

- [x] 2.1 Spot-check current help and parse flows for `index`, `search`, `benchmark`, `status`, `mcp`, `setup`, and `version`
- [x] 2.2 Run `go test ./...`
- [x] 2.3 Run `go build ./...`
