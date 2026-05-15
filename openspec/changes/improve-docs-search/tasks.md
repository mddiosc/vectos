## 1. HNSW Loading for Docs Search

- [ ] 1.1 Modify `runSearch` in `cmd/vectos/commands_search.go` to call `store.LoadVectorIndex()` for documentation stores (current guard skips it when `docsOnly=true`)
- [ ] 1.2 Verify the HNSW loading warning message appears when index is missing (same behavior as code search)
- [ ] 1.3 Test: docs search latency drops from ~1.85s to ~0.25s with HNSW loaded

## 2. Docs Directory Exclusions

- [ ] 2.1 Add `skippedDocsDirs` map to `internal/content/language.go` with `.github/prompts/` entry (separate from `skippedDirs` to avoid affecting code indexing)
- [ ] 2.2 Add `.agents/skills/` to `skippedDocsDirs` 
- [ ] 2.3 Implement `ShouldSkipDocsDir(name string) bool` function
- [ ] 2.4 Integrate `ShouldSkipDocsDir` into the documentation indexing file walker in `internal/content/paths.go`
- [ ] 2.5 Verify existing `skippedDirs` (`.git`, `node_modules`, etc.) still work for docs indexing
- [ ] 2.6 Test: reindex docs → `.github/prompts/` and `.agents/skills/` files are no longer indexed
- [ ] 2.7 Add unit test: `ShouldSkipDocsDir` returns true for `.github/prompts` and `.agents/skills/`

## 3. Path-Based Keyword Scoring Boosts

- [ ] 3.1 Add `isDocFilePath(filePath string) bool` helper that checks for `docs/` prefix or `README.md` filename in `internal/storage/sqlite.go`
- [ ] 3.2 Apply 1.5× multiplier in `computeKeywordScore` when `isDocFilePath` returns true
- [ ] 3.3 Test: a blog post chunk scores lower than an equivalent `docs/` chunk for the same keyword match
- [ ] 3.4 Add unit test: `isDocFilePath` correctly identifies `docs/en/TESTING.md` and `README.md`, rejects `src/content/blog/post.md`

## 4. Integration & Regression

- [ ] 4.1 Run full reindex of mywebsite-2 docs (`vectos index --docs`)
- [ ] 4.2 Verify docs search queries from benchmark return improved results (higher precision, lower latency)
- [ ] 4.3 Run existing test suite (`go test ./...`) to verify no regressions
- [ ] 4.4 Verify code search is unaffected (same queries, same results)
