## 1. Regression baseline

- [x] 1.1 Run `go test ./cmd/vectos/... -v -run TestResolveChangedPath` and confirm all existing tests pass

## 2. Fix `resolveChangedPath` to try all scope roots

- [x] 2.1 Modify `resolveChangedPath` in `cmd/vectos/runtime_paths.go` to build the base list as `[WorkspaceRoot, PrimaryRoot, ...scope.Roots]` with deduplication, then iterate all bases with the existing `isWithinRoots || fileExists` check
- [x] 2.2 Add test `TestResolveChangedPathResolvesDependencyRootRelativePaths` in `cmd/vectos/runtime_paths_test.go` — creates a real file at `libs/lib-ui/src/button.tsx` inside a temp workspace, calls `resolveChangedPath(scope, "src/button.tsx")` where `libs/lib-ui` is in `scope.Roots` but not `PrimaryRoot`, expects the lib path returned
- [x] 2.3 Run `go test ./cmd/vectos/... -count=1` — all tests pass including new one

## 3. Improve managed guidance

- [x] 3.1 Add `## Incremental Reindex (\`changed\`)` section to `managedGuidance` in `internal/setup/guidance_content.go` covering: absolute/workspace-relative paths, renames include old+new, docs refresh with `docs: true`, full reindex after config changes, Nx shared-lib downstream refresh
- [x] 3.2 Run `go test ./internal/setup/... -count=1` — passes

## 4. Final verification

- [x] 4.1 Run `go test ./... -count=1` — 39+ tests pass, no regressions
- [x] 4.2 Run `go build ./cmd/vectos` — binary compiles
