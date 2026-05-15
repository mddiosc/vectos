## Why

Security audit revealed five vulnerabilities across search, download, indexing, and HTTP serving layers. These issues could be exploited by a compromised configuration, a malicious MCP client with localhost access, or an attacker controlling the asset base URL — leading to denial of service, path traversal, or arbitrary binary execution. Fixing them now hardens the system before any external exposure.

## What Changes

- **SQL LIKE wildcard escaping**: Escape `%` and `_` characters in user-provided search terms before constructing LIKE clauses to prevent wildcard-injection DoS via patterns like `%*10000`.
- **Asset base URL validation**: Validate the `asset_base_url` config field on load — reject non-HTTPS schemes, paths containing `..`, and URLs exceeding a reasonable length.
- **Content-Type check on model downloads**: Verify the `Content-Type` header matches expected MIME types (`application/octet-stream`, `application/x-gzip`, `application/gzip`) before writing downloaded model assets to disk.
- **Sensitive file/path blocklist**: Expand the existing `ShouldSkipDir` blocklist to refuse indexing of sensitive items: `.env` files, SSH keys, credential files, and direct `.git` object traversal (already partially covered but needs a file-level filter for dot-env and key files).
- **Rate limiting on /reindex**: Add a token-bucket rate limiter (1 req/sec, burst 5) to the `/reindex` HTTP endpoint to prevent resource exhaustion from repeated expensive reindex operations.

## Capabilities

### New Capabilities
- `rate-limiting`: Token-bucket rate limiting for the `/reindex` HTTP endpoint, with configurable rate and burst, preventing an attacker from triggering repeated expensive reindex operations.

### Modified Capabilities
- `semantic-search`: Text-search fallback path (SQL LIKE queries) SHALL escape user-provided search terms to prevent wildcard injection.
- `embedding-provider`: Embedded provider's auto-download flow SHALL validate `asset_base_url` at configuration time and check Content-Type headers during download.
- `code-indexing`: File-walking functions SHALL reject known sensitive file patterns (`.env`, SSH private keys, credential files) in addition to existing directory skips.
- `serve-http-server`: The `/reindex` endpoint SHALL enforce a configurable rate limit to prevent abuse.

## Impact

- **Storage layer** (`internal/storage/sqlite.go`): New `escapeLikeTerm` helper, applied in `SearchText`.
- **Config validation** (`internal/config/embedding.go`): New `ValidateAssetBaseURL` function, called during config merge.
- **Embedded provider** (`internal/embeddings/embedded.go`): Content-Type header check in `downloadAsset`.
- **Content/indexer** (`internal/content/language.go`, `paths.go`): Extended `ShouldSkipDir` with file-level sensitive-file filter.
- **HTTP server** (`internal/server/server.go`): Token-bucket rate limiter in `handleReindex`.
- New dependency: `golang.org/x/time/rate` (standard extended library).
