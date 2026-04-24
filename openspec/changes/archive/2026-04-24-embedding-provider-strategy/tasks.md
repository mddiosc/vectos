## 1. Provider Abstraction

- [x] 1.1 Add provider configuration model for embedded and remote embeddings
- [x] 1.2 Add provider factory/resolution logic
- [x] 1.3 Move current remote implementation behind the provider factory
- [x] 1.4 Persist provider/model metadata with each project index

## 2. Embedded Provider

- [x] 2.1 Integrate an in-process ONNX-based local embeddings runtime
- [x] 2.2 Add local model resolution/download/cache workflow for the embedded provider
- [x] 2.3 Add default embedded provider selection
- [x] 2.4 Add provider health validation at startup and on demand

## 3. Fallback and Migration

- [x] 3.1 Add explicit fallback order between providers
- [x] 3.2 Detect provider mismatch that requires reindexing
- [x] 3.3 Document provider configuration and migration behavior

## 4. Standalone Validation

- [x] 4.1 Verify indexing works with embedded provider and no external endpoint configured
- [x] 4.2 Verify semantic search works with embedded provider and existing local index metadata
