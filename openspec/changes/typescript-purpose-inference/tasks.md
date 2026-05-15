## 1. Regex Patterns & Detection Functions

- [ ] 1.1 Add `tsInterfacePattern` regex to `internal/indexer/chunker.go` matching `^(export\s+)?interface\s+[A-Z][\w$]*`
- [ ] 1.2 Add `tsTypeAliasPattern` regex matching `^(export\s+)?type\s+[A-Z][\w$]*\s*=`
- [ ] 1.3 Add `tsEnumPattern` regex matching `^(export\s+)?(const\s+)?enum\s+[A-Z][\w$]*`
- [ ] 1.4 Add `tsAsyncPattern` regex matching `^(export\s+)?async\s+(function\s+|\(.*\)\s*=>)`
- [ ] 1.5 Implement `isInterfaceChunk(language, chunkContent string) bool` helper
- [ ] 1.6 Implement `isTypeAliasChunk(language, chunkContent string) bool` helper
- [ ] 1.7 Implement `isEnumChunk(language, chunkContent string) bool` helper
- [ ] 1.8 Implement `isAsyncChunk(language, chunkContent string) bool` helper

## 2. Purpose Inference Integration

- [ ] 2.1 Integrate new detection functions into `inferNonGoPurpose()` — add checks for interface, type alias, enum, and async before the generic fallback checks
- [ ] 2.2 Ensure tags are additive: a chunk matching multiple patterns gets all applicable tags joined with "; "
- [ ] 2.3 Update `isStructuredBoundary()` to include interface and enum as structural boundaries for TypeScript (alongside existing component/hook/function/class boundaries)
- [ ] 2.4 Verify `extractSignature()` correctly extracts interface/enum/type alias declarations as signatures for non-Go languages (existing `isStructuredBoundary` check already handles this once boundaries are added)

## 3. Tests

- [ ] 3.1 Add unit test: interface chunk receives "type definition" tag
- [ ] 3.2 Add unit test: type alias chunk receives "type definition" tag
- [ ] 3.3 Add unit test: enum chunk receives "enumeration" tag
- [ ] 3.4 Add unit test: async function chunk receives "async function" tag
- [ ] 3.5 Add unit test: chunk with both component and async gets both "react component" and "async function"
- [ ] 3.6 Add unit test: regular function without async does NOT get "async function" tag
- [ ] 3.7 Add unit test: `export interface` gets both "type definition" and "exported api"
- [ ] 3.8 Add unit test: interface/enum are recognized as structural boundaries in `isStructuredBoundary()`
- [ ] 3.9 Add table-driven test for all new patterns (valid and invalid inputs)
- [ ] 3.10 Run existing test suite (`go test ./internal/indexer/...`) to verify no regressions
