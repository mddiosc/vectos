## 1. SQL LIKE Wildcard Escape

- [x] 1.1 Add `escapeLikeTerm(s string) string` helper in `internal/storage/sqlite.go` that escapes `\` → `\\`, `%` → `\%`, and `_` → `\_`
- [x] 1.2 Apply `escapeLikeTerm(query)` to user-provided `query` in `SearchText` before wrapping with `%` wildcards
- [x] 1.3 Add unit tests for `escapeLikeTerm` covering normal terms, wildcard-only queries, mixed content, and escape-chain bypass prevention

## 2. Asset Base URL Validation

- [x] 2.1 Add `ValidateAssetBaseURL(raw string) error` in `internal/config/embedding.go` that validates URL scheme (HTTPS only), path traversal (`..`), non-empty host, and max length (2048 chars)
- [x] 2.2 Call `ValidateAssetBaseURL` from `mergeEmbeddedConfig` when `asset_base_url` is non-empty, returning configuration error on failure
- [x] 2.3 Add unit tests for `ValidateAssetBaseURL` covering valid HTTPS, HTTP rejection, path traversal rejection, empty URL passthrough, and over-length rejection

## 3. Content-Type Check on Model Downloads

- [x] 3.1 In `downloadAsset` (`internal/embeddings/embedded.go`), add Content-Type header validation against allowed list (`application/octet-stream`, `application/gzip`, `application/x-gzip`, `application/x-tar`) after receiving HTTP response, accepting empty Content-Type as valid
- [x] 3.2 Add zero-length response body check before renaming temp file — treat empty body as download failure
- [x] 3.3 Apply same Content-Type validation in `downloadRuntimeLibrary` (same file)
- [x] 3.4 Add unit tests for Content-Type validation covering allowed types, empty header passthrough, rejection of unexpected types (e.g., `text/html`), and zero-length body rejection

## 4. Sensitive File Blocklist

- [x] 4.1 Add `ShouldSkipFile(name string) bool` function in `internal/content/language.go` that blocks exact matches for `.env`, `.env.local`, `.env.production`, `.env.development`; SSH private key names (`id_rsa`, `id_ecdsa`, `id_ed25519` and `*_rsa`, `*_ecdsa`, `*_ed25519`); and credential/certificate files (`*.pem`, `*.key`, `*.pfx`, `*.p12`, `credentials.json`, `service-account.json`)
- [x] 4.2 Call `ShouldSkipFile(info.Name())` in `walkDir`'s file handler (`internal/content/paths.go`) before adding paths to indexable list, appending skipped files with reason "sensitive file"
- [x] 4.3 Add unit tests for `ShouldSkipFile` covering .env variants, SSH keys, certificate files, credential files, and non-sensitive files (e.g., `.env.example`, standard source files)

## 5. Rate Limiting on /reindex

- [x] 5.1 Add `golang.org/x/time/rate` dependency via `go get`
- [x] 5.2 Add `reindexLimiter *rate.Limiter` field to `Server` struct in `internal/server/server.go`
- [x] 5.3 Initialize `reindexLimiter` in `NewServer` with rate 1 and burst 5 (configurable via environment variable as future enhancement)
- [x] 5.4 In `handleReindex`, check `limiter.Allow()` before processing; return HTTP 429 with `{"status": "error", "message": "rate limit exceeded"}` body when denied
- [x] 5.5 Add unit tests for rate limiting covering within-limit requests, burst capacity, sustained rate capping, and 429 response format

## 6. Integration & Verification

- [x] 6.1 Run full test suite (`go test ./...`) and fix any failures
- [ ] 6.2 Manual smoke test: verify text search with wildcard characters, validate bad asset_base_url is rejected at startup, confirm sensitive files are skipped during indexing, and confirm rapid /reindex requests return 429 after burst exhausted
