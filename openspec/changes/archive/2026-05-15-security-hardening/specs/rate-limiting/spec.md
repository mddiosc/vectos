## ADDED Requirements

### Requirement: /reindex endpoint SHALL enforce rate limiting
The `/reindex` endpoint SHALL enforce a token-bucket rate limit to prevent resource exhaustion from excessive reindex requests. The default configuration SHALL allow 1 request per second with a burst capacity of 5 requests.

#### Scenario: Request within rate limit
- **WHEN** a POST request is made to `/reindex` and the rate limiter has available tokens
- **THEN** the request SHALL be processed normally and the rate limiter SHALL consume one token

#### Scenario: Request exceeds rate limit
- **WHEN** a POST request is made to `/reindex` and the rate limiter has exhausted its burst capacity
- **THEN** the server SHALL return HTTP 429 Too Many Requests with body `{"status": "error", "message": "rate limit exceeded"}`

#### Scenario: Burst allows short spike of requests
- **WHEN** up to 5 rapid POST requests are made to `/reindex` in quick succession
- **THEN** all requests SHALL be accepted and processed (serialized as per existing behavior)

#### Scenario: Sustained rate caps at 1/sec
- **WHEN** more than 5 requests arrive within a 5-second window
- **THEN** requests beyond the burst capacity SHALL receive HTTP 429 until tokens regenerate

#### Scenario: Rate limit does not apply to /health endpoint
- **WHEN** a GET request is made to `/health`
- **THEN** the request SHALL NOT be subject to rate limiting
