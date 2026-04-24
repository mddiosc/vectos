## Why

Vectos already covers source files and some infrastructure formats, but it still misses many of the files that developers rely on daily to understand project behavior, tooling, scripts, and configuration. Adding support for common config, script, and documentation files would make semantic search much more useful in real repositories.

## What Changes

- Add support for common project files such as `.json`, `.sh`, `.md`, `.toml`, `.ini`, `.xml`, `.properties`, `Makefile`, and `.gitignore`.
- Expand lightweight file classification beyond `source` and `infra_config` to include `scripts`, `docs`, and `dependency_metadata`.
- Add baseline chunking heuristics appropriate for these file types without requiring deep parsers.
- Ensure search and indexing output preserve the new file-type/category context.

## Capabilities

### New Capabilities
- `common-project-files`: Supports indexing common config, script, and documentation files used in everyday repositories.
- `content-categorization`: Expands content categorization to distinguish source, infra/config, scripts, docs, and dependency metadata.

### Modified Capabilities
- `language-expansion`: Extend the supported file-type set and reporting behavior to include common non-source project files.

## Impact

- Changes to file detection and chunking heuristics.
- Changes to chunk metadata/category assignment.
- Changes to README/docs and search/status output expectations.
