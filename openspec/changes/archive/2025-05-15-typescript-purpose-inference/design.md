## Context

The chunker (`internal/indexer/chunker.go`) already performs structured chunking for JS/TS files via `chunkBraceStructuredFileImpl`, splitting at function, component, hook, class, and test boundaries. Each chunk gets a `purpose` tag via `inferNonGoPurpose` (line 445) that feeds into `buildSemanticContent` for embedding enrichment.

Current detected patterns in `inferNonGoPurpose`:
- React components
- Custom hooks (`use[A-Z]`)
- Test blocks (`describe`, `it`, `test`)
- Exported APIs (`export `)
- Network/API access (`fetch`, `axios`)
- Functions/callables (`function`, `=>`)
- Classes
- Return statements

**Missing patterns**: `interface`, `type` alias, `enum`, `async function`, generic types. These fall through to "code block" — a significant gap given TypeScript's heavy use of structural typing.

## Goals / Non-Goals

**Goals:**
- Detect `interface X { }` declarations → tag "type definition"
- Detect `type X = ...` aliases → tag "type definition"
- Detect `enum X { }` declarations → tag "enumeration"
- Detect `async function X()` and `async () =>` → tag "async function"
- Integrate seamlessly with existing `inferNonGoPurpose` logic

**Non-Goals:**
- Full TypeScript AST parsing
- Tracking generic type parameters across files
- Type narrowing or inference analysis
- Changes to the chunking boundary logic (interfaces already split correctly at `isStructuredBoundary`)

## Decisions

### Decision 1: Regex-based detection, same as existing patterns

Add new regex patterns following the existing style (line 13-19 in chunker.go):

```go
var tsInterfacePattern = regexp.MustCompile(`^(export\s+)?interface\s+[A-Z][\w$]*`)
var tsTypeAliasPattern = regexp.MustCompile(`^(export\s+)?type\s+[A-Z][\w$]*\s*=`)
var tsEnumPattern = regexp.MustCompile(`^(export\s+)?(const\s+)?enum\s+[A-Z][\w$]*`)
var tsAsyncPattern = regexp.MustCompile(`^(export\s+)?async\s+(function\s+|\(.*\)\s*=>)`)
```

**Alternative considered**: Tree-sitter or AST-based detection. Rejected because it adds a heavy dependency and the regex approach already handles 90%+ of real-world patterns correctly.

### Decision 2: Group `interface` and `type` alias under "type definition"

Both represent TypeScript structural type definitions. Using a single tag keeps the embedding space concise and makes queries like "type definitions" match both.

### Decision 3: Add detection functions alongside existing ones (`isHookChunk`, `isReactComponentChunk`)

New helpers: `isInterfaceChunk`, `isTypeAliasChunk`, `isEnumChunk`, `isAsyncChunk`. Each follows the same pattern: iterate lines, match regex, return bool.

### Decision 4: Tags are additive to existing tags

A chunk that is both a React component AND uses `async function` gets both tags: "react component; async function". This enriches the embedding without losing information.

## Risks / Trade-offs

- **[False positives]** Regex may match `interface` in comments or string literals → mitigated by matching only at line start after trimming whitespace, same as existing patterns
- **[`type` keyword ambiguity]** `type X = ...` vs `typeof x` → the regex requires `type` followed by uppercase identifier then `=`, which distinguishes type aliases from other `type` usage
