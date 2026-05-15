## 1. HNSW Loading for Docs Search

- [x] 1.1 Modify `runSearch` in `cmd/vectos/commands_search.go` to call `store.LoadVectorIndex()` for documentation stores
- [x] 1.2 Verify the HNSW loading works for docs stores (already applies to both stores via `runSearch`)
- [x] 1.3 Test: docs search latency drops from ~1.85s to ~0.25s with HNSW loaded

## 2. Configurable Index Exclusions — Config Loading

- [x] 2.1 Create `internal/config/project.go` with `IndexConfig` struct and `LoadIndexConfig` function
- [x] 2.2 Add `IndexConfig` support to global config parsing in `LoadIndexConfig`
- [x] 2.3 Implement `LoadIndexConfig(globalConfigPath, projectDir string) IndexConfig` that merges global + project patterns
- [ ] 2.4 Add unit test: project config file is parsed correctly
- [ ] 2.5 Add unit test: project patterns are appended to global patterns (not replaced)
- [ ] 2.6 Add unit test: missing config files return empty exclusions without error

## 3. Configurable Index Exclusions — .gitignore Integration

- [x] 3.1 Implement `ReadGitignorePatterns(projectDir string) ([]string, error)` that reads `.gitignore` and returns usable patterns
- [x] 3.2 Handle gitignore negation patterns (`!pattern`) — skip them in initial implementation
- [x] 3.3 Handle gitignore directory-only patterns (trailing `/`) — preserved as-is for filepath.Match
- [ ] 3.4 Add unit test: `.gitignore` patterns are correctly parsed
- [ ] 3.5 Add unit test: empty or missing `.gitignore` returns empty slice

## 4. Configurable Index Exclusions — Integration

- [x] 4.1 Modify path accumulation with `excludePatterns` field and `shouldExclude` method
- [x] 4.2 Implement `shouldExclude(absPath string) bool` using `filepath.Match`
- [x] 4.3 Pass merged exclusion patterns to `CollectIndexablePathsWithExclusions` in `runIndex` and `reindexProject`
- [x] 4.4 Ensure hardcoded exclusions (`ShouldSkipFile`, `ShouldSkipDir`) still apply
- [ ] 4.5 Add unit test: file matching exclusion pattern is skipped
- [ ] 4.6 Add unit test: file NOT matching exclusion pattern is indexed

## 5. Path-Based Keyword Scoring Boosts

- [x] 5.1 Add `isDocFilePath(filePath string) bool` helper for `docs/` prefix or `README.md` filename
- [x] 5.2 Apply 1.5× multiplier in `computeKeywordScore` when `isDocFilePath` returns true
- [ ] 5.3 Add unit test: blog post scores lower than equivalent docs/ chunk

## 6. Documentation

- [x] 6.1 Update `README.md` with jina-embeddings-v3 default model, RRF fusion, TS structural tags, config exclusions
- [x] 6.2 Update `docs/indexing.md` with `vectos.config.json` format, global config `index` section, `.gitignore` integration
- [ ] 6.3 Create `docs/configuration.md` with full config reference (embeddings, index exclusions, HNSW tuning)
- [x] 6.4 Add config example snippets to documentation

## 7. Integration & Regression

- [ ] 7.1 Run full reindex of mywebsite-2 docs with global config exclusions
- [ ] 7.2 Run full reindex of mywebsite-2 code with project config exclusions
- [ ] 7.3 Verify docs search latency and precision improve
- [x] 7.4 Run existing test suite (`go test ./...`) to verify no regressions
- [x] 7.5 Verify code search is unaffected by config changes
