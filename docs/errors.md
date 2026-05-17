# Error Guide

This page lists the main user-facing indexing and embedding errors that Vectos now tries to surface more clearly.

## Filesystem Errors

Typical messages:

- `cannot read file /path/to/file: it does not exist`
- `cannot read file /path/to/file: permission denied`
- `cannot access path /path/to/project: it does not exist`

What to do:

- confirm the path still exists
- confirm the current user can read the file or directory
- retry after fixing permissions or restoring the file

## Remote Embedding Provider Errors

Typical messages:

- `embedding API request to ... timed out after 30s; check the remote provider or increase embeddings.remote.timeout_seconds`
- `embedding API rate limited the request (429 Too Many Requests); retry later or lower request concurrency`
- `embedding API rejected the request (401 Unauthorized); check embeddings.remote.base_url and upstream authentication`
- `embedding API endpoint not found at ...; check embeddings.remote.base_url`
- `embedding API is temporarily unavailable (502 Bad Gateway)`

What to do:

- verify `embeddings.remote.base_url`
- confirm the upstream service is healthy
- if you use a proxy or authenticated gateway, verify its auth setup
- increase `embeddings.remote.timeout_seconds` for slow providers
- retry later if the provider is rate limiting or unavailable

## Embedded Model Setup Errors

Typical messages already surfaced by the embedded provider include:

- missing model directory
- missing model assets when auto-download is disabled
- failed model asset download
- failed tokenizer or ONNX session initialization

What to do:

- run `vectos doctor`
- check `~/.vectos/models/`
- confirm downloads are allowed from the configured asset source
- retry `vectos index .` after fixing the local model files
