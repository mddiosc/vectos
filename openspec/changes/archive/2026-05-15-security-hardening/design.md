## Context

Vectos is a Go binary with an HTTP server (`vectos serve`) listening on `127.0.0.1`. It supports SQLite-backed keyword search with LIKE fallback, embedded ONNX-based model downloads from a configurable URL, file-walking for indexing, and a `/reindex` endpoint used by MCP clients. A security audit identified five issues across these layers. All are addressable with small, focused changes — no architectural refactoring needed.

## Goals / Non-Goals

**Goals:**
- Prevent wildcard-injection DoS in SQL LIKE text search
- Validate `asset_base_url` to prevent path traversal and unexpected hosts
- Verify Content-Type on model asset downloads
- Block indexing of sensitive files (`.env`, SSH keys, credentials)
- Rate-limit `/reindex` endpoint to prevent CPU exhaustion

**Non-Goals:**
- Network-level firewall or IP-based rate limiting (the server is already localhost-only)
- Input validation for every HTTP endpoint (only /reindex needs it)
- General-purpose download content scanning
- Filesystem permissions enforcement
- TLS for localhost server

## Decisions

### 1. LIKE escape: prefix-escape function, not regex replacement

**Decision**: Add a simple `escapeLikeTerm(s string) string` function that escapes `%` → `\%` and `_` → `\_`, and an additional `\` → `\\` to prevent escape-chain bypass. Apply it to `query` in `SearchText` before concatenating `%` wildcards.

**Rationale**: Simple, no allocation-heavy regex, comprehensive (prevents `\%` abuse too). SQLite's default ESCAPE is `\`, so this is compatible.

**Alternatives considered**:
- SQLite `GLOB` instead of LIKE — would break current escape semantics
- Client-side length limit only — doesn't prevent `%`-only patterns that are short but expensive

### 2. Asset base URL validation: config-time, not download-time only

**Decision**: Add `ValidateAssetBaseURL(raw string) error` in `internal/config/embedding.go`, called during `mergeEmbeddedConfig`. Validation rules:
- Must parse as valid URL (`net/url.Parse`)
- Scheme must be `https` (reject `http`, `file`, etc.)
- Host must be non-empty
- Raw URL must not contain `..` (prevents path traversal in path construction)
- Raw URL string length ≤ 2048

**Rationale**: Fail fast at config load. The `downloadAsset` function constructs URLs as `baseURL + "/" + remotePath` — a malicious base URL like `https://evil.com/../../etc/` combined with relative paths could point outside the intended asset space.

**Alternatives considered**:
- Validate only at download time — too late, harder to diagnose
- Use `strings.HasPrefix` only — doesn't catch all traversal variants
- Parse and re-join paths — more robust but the `..` check is sufficient for current URL construction pattern

### 3. Content-Type check: allow-list of expected types

**Decision**: In `downloadAsset`, after receiving the HTTP response, check `Content-Type` before writing. Allowed types: `application/octet-stream`, `application/gzip`, `application/x-gzip`, `application/x-tar`. Also accept if Content-Type header is empty (some CDNs omit it for binary). Check size is non-zero before renaming.

**Rationale**: ONNX model files and tokenizer configs are binary or JSON — Content-Type mismatch signals a misconfigured or compromised server. Empty Content-Type is common enough to accept. The check adds minimal overhead (one header read).

**Alternatives considered**:
- Magic bytes detection — overkill for this threat model, adds complexity
- Full SHA-256 integrity check — requires maintaining a hash manifest, out of scope

### 4. Sensitive file blocklist: extend ShouldSkipDir with file-level filter

**Decision**: Add a `ShouldSkipFile(name string) bool` function in `internal/content/language.go` that blocks:
- Files named `.env`, `.env.local`, `.env.production`, `.env.development`
- Filenames matching `*_rsa`, `*_ecdsa`, `*_ed25519` (SSH private keys), `id_rsa`, `id_ecdsa`, `id_ed25519`
- Filenames matching `*.pem`, `*.key`, `*.pfx`, `*.p12`, `credentials.json`, `service-account.json`
- Files inside `.git` in `walkDir` already skipped by `ShouldSkipDir` but ensure `.git/config`, `.git/HEAD` etc. are covered

Call `ShouldSkipFile(info.Name())` in `walkDir`'s file handler before adding to indexable paths.

**Rationale**: `.env` and credential files are the most commonly accidentally-indexed sensitive files. Blocking at the walking layer means they never reach storage. Existing `ShouldSkipDir` already covers `.git` directory-level, but adding explicit file filter ensures no edge case leak.

**Alternatives considered**:
- Content-based detection (regex for API keys) — too expensive, false positives
- Opt-in allowlist — too restrictive for general-purpose indexing

### 5. Rate limiting: `golang.org/x/time/rate` token bucket

**Decision**: Add a `rate.Limiter` field to the `Server` struct, initialized in `NewServer` with rate 1 and burst 5. In `handleReindex`, call `limiter.Allow()` before processing — if not allowed, return HTTP 429 Too Many Requests.

**Rationale**: `x/time/rate` is the de-facto standard Go rate limiter, used by Kubernetes and many production systems. Token bucket with burst allows legitimate burst of requests (e.g., 5 rapid incremental reindexes) while capping sustained rate. 1/sec with burst 5 is conservative for a localhost-only server.

**Alternatives considered**:
- Custom sliding window — reinventing the wheel
- `time.Ticker`-based simple limiter — no burst support, worse UX for legitimate use
- Queue-based backpressure — too complex for this use case

## Risks / Trade-offs

- **[Risk] LIKE escape changes text search behavior**: Queries containing `%` or `_` will no longer match as wildcards in text search. → **Mitigation**: Text search already has semantic search as primary path; LIKE is fallback. Users searching for literal `%` in code are extremely rare. Note in changelog.
- **[Risk] Content-Type check may reject downloads from CDNs with unusual headers** → **Mitigation**: Accept empty Content-Type. Only reject when header is present AND doesn't match allowed list. If this causes issues, the allowed list is easy to extend.
- **[Risk] Sensitive file filter may block legitimate files named `.env` (e.g., `.env.example`)** → **Mitigation**: `ShouldSkipFile` is explicitly matched; `.env.example` would NOT be blocked. Only exact `.env` and well-known patterns are blocked.
- **[Risk] Rate limiter may reject legitimate rapid reindex requests** → **Mitigation**: Burst 5 allows short bursts. Legitimate MCP clients shouldn't need >5 reindexes in rapid succession. If needed, rate/burst can be made configurable.

## Open Questions

- Should rate limit parameters be configurable via environment variable? (Decision: No for now — keep it simple, revisit if legit users hit the limit.)
- Should content-type validation apply to ONNX runtime library downloads too? (Decision: Yes, same `downloadAsset`-style check in `downloadRuntimeLibrary`.)
