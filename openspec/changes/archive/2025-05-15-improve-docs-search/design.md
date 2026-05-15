## Context

The docs search pipeline (`executeSearchDocs` in `cmd/vectos/search_exec.go`) uses the same RRF fusion engine as code search (vector + keyword), but the initialization path lacks HNSW loading. Additionally, the documentation index includes blog posts, agent skill manifests, and GitHub prompts that dominate keyword scoring.

There is currently no user-facing configuration for index exclusions — all exclusion rules are hardcoded in `internal/content/language.go`. Users need per-project and global control over what gets indexed.

### Current state

- Code search: HNSW loaded, linear scan for <1000 chunks, avg 0.25s
- Docs search: HNSW never loaded, always linear scan, avg 1.85s
- Exclusions: only hardcoded (`skippedDirs`, `sensitiveFilenames`) — no user config
- Config: only `~/.vectos/config.json` for embedding provider settings

## Goals / Non-Goals

**Goals:**
- Load HNSW vector index for documentation search stores
- Apply the same <1000-chunk linear-scan threshold used in code search
- Add configurable exclusion patterns via project-level `vectos.config.json` and global `~/.vectos/config.json`
- Automatically respect `.gitignore` patterns during indexing
- Support separate exclusion lists for `docs` and `code` indexes
- Merge strategy: global defaults + project overrides (cumulative)
- Add path-based keyword scoring boosts for files under `docs/` and project READMEs
- Document all recent features

**Non-Goals:**
- Changing the chunking strategy for documentation files
- Separate embedding model for docs
- Regex-based exclusion patterns (glob only, matching gitignore syntax)

## Decisions

### Decision 1: Three-tier exclusion system (gitignore → global config → project config)

**Tier 1 — `.gitignore` (automatic):** Files matching gitignore patterns are always excluded. Implemented by reading `.gitignore` at indexing time and converting patterns to exclusion checks. Uses `filepath.Match` for simple patterns; directories are handled specially (trailing `/` in gitignore = directory-only match).

**Tier 2 — Global config (`~/.vectos/config.json`):** New `index.docs.exclude` and `index.code.exclude` arrays. These provide organization-wide defaults (e.g., always exclude `.github/prompts/` from docs).

**Tier 3 — Project config (`vectos.config.json`):** Placed in project root. Same structure as global `index` section. Patterns are appended to global patterns (not replaced). Allows per-project overrides (e.g., exclude `src/content/blog/` for this specific project).

**Alternative considered**: Only project-level config. Rejected because teams want organization-wide defaults without adding a config file to every project.

### Decision 2: Separate `docs` and `code` exclusion lists

Each has its own `exclude` array. This allows different policies: blog posts should be excluded from docs index but remain in code index (they're semantically relevant code content). Build artifacts should be excluded from both.

### Decision 3: Config structure

Global (`~/.vectos/config.json`):
```json
{
  "embeddings": { ... },
  "index": {
    "docs": { "exclude": [".github/prompts/**", ".agents/**"] },
    "code": { "exclude": [] }
  }
}
```

Project (`vectos.config.json`):
```json
{
  "index": {
    "docs": { "exclude": ["src/content/blog/**"] },
    "code": { "exclude": ["**/__mocks__/**"] }
  }
}
```

### Decision 4: Merge strategy is cumulative

Hardcoded exclusions always apply. Global config adds more. Project config adds even more. No replacement — all three layers are active simultaneously. If a user wants to "un-exclude" something from a global default, they must remove it from the global config (project config cannot undo global exclusions).

**Alternative considered**: Project config replaces global. Rejected because it makes global defaults unreliable — a project could accidentally expose sensitive files by not including a security exclusion.

### Decision 5: Glob syntax uses gitignore-compatible patterns

Same syntax as `.gitignore`: `**` for recursive directories, `*` for single-level wildcards, `!` for negation (future). This is familiar to developers and consistent with git.

## Risks / Trade-offs

- **[.gitignore may exclude legitimate docs]** Some projects gitignore generated docs that should still be searchable → mitigated by project config allowing override intent (future: `!pattern` negation)
- **[Config file proliferation]** Adding yet another config file → mitigated by making it optional; zero-config works fine for most projects
- **[Glob matching performance]** Checking patterns per file adds overhead → mitigated by compiling patterns once at indexing start; matching is O(patterns × filename), negligible vs embedding generation
