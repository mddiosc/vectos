## Context

Vectos currently indexes source files plus a bounded set of infra/config formats, but many practical repositories rely heavily on shell scripts, JSON/TOML/INI-style config, Markdown documentation, XML/properties files, and root-level project metadata like `Makefile` or `.gitignore`. These files are often exactly where setup, tooling, routing, dependency, or deployment behavior is described.

## Goals / Non-Goals

**Goals:**
- Add a bounded set of common project file types that improve real-world repository understanding.
- Keep chunking lightweight and heuristic-based.
- Expand categories to make search results easier to interpret.

**Non-Goals:**
- Deep parsing or semantic validation of every supported config format.
- Indexing secrets or private runtime env files by default.
- Turning lockfiles or huge generated config files into parser-heavy first-class artifacts.

## Decisions

- Add support first for: `.json`, `.sh`, `.md`, `.toml`, `.ini`, `.xml`, `.properties`, `Makefile`, and `.gitignore`.
- Treat common config/docs/script files with line-based chunking unless a trivial boundary heuristic is obvious.
- Expand categories to: `source`, `infra_config`, `scripts`, `docs`, and `dependency_metadata`.
- Keep `.env`-style secret-bearing files out of this first phase unless they are explicit sample/example variants.
- Treat dependency-heavy metadata files such as `package.json`, `Cargo.toml`, `go.mod`, and similar files as `dependency_metadata` rather than generic config.

## Risks / Trade-offs

- [Risk] Some supported formats may include very noisy machine-generated content. → Mitigation: keep the first scope bounded and avoid obviously generated files.
- [Risk] Category heuristics may be imperfect. → Mitigation: keep category assignment simple and deterministic.
- [Risk] JSON and Markdown can become large and broad. → Mitigation: continue using chunk-size limits and line-based chunking.

## Migration Plan

1. Extend file detection.
2. Add category heuristics for the new formats.
3. Add baseline chunking behavior for the new files.
4. Expose category/language context in results and docs.

## Open Questions

- Whether `.env.example` / `.env.sample` should be included in the same phase or a later safe-config follow-up.
