## Context

Vectos already abstracts embeddings behind an interface, which makes this a good time to formalize provider strategy rather than keeping a single remote implementation wired into the CLI and MCP server.

However, the primary requirement is not merely to support multiple providers. Vectos must be able to generate embeddings locally without requiring the user to run or configure a separate embeddings service. The architecture therefore needs to prioritize a true embedded runtime, with remote providers treated as optional extensions.

## Goals / Non-Goals

**Goals:**
- Add an embedded local provider as the preferred default.
- Keep support for remote URL-based providers using an OpenAI-compatible embeddings API.
- Add provider selection and fallback policy.
- Make standalone local indexing/search work without an externally managed embeddings endpoint.
- Track index metadata so provider/model changes can invalidate semantic search until reindex.

**Non-Goals:**
- Supporting every vendor-specific embeddings API directly in the first iteration.
- Building a custom model training or conversion pipeline.
- Building a general-purpose local LLM runtime beyond what is needed for embeddings.

## Decisions

- Default provider type will be `embedded`.
- The embedded provider will run in-process inside Vectos rather than requiring an external sidecar or model server.
- The first embedded implementation will target an ONNX-based embeddings runtime with a compact local model suited for code search.
- Remote providers will target an OpenAI-compatible embeddings contract.
- Provider selection will be configuration-driven, not hardcoded in the CLI.
- Fallback order will be explicit rather than implicit.
- Every project index will persist enough embedding metadata to detect when the current provider/model is incompatible with stored vectors.

## Risks / Trade-offs

- Embedded inference increases packaging complexity.
- ONNX runtime and model assets increase installation footprint and cross-platform packaging work.
- Remote fallback improves resilience but adds configuration complexity.
- Different providers may produce embeddings with incompatible dimensions, so an index may need to be rebuilt when provider configuration changes.
- An embedded ONNX path is a narrower and more deliberate choice than a general llama.cpp runtime, but is a better fit for compact embeddings models.

## Migration Plan

1. Introduce provider config, provider factory, and project index metadata.
2. Move the current remote client behind the new provider system.
3. Add the embedded ONNX provider as the default standalone path.
4. Add provider validation, fallback logic, and reindex detection.
