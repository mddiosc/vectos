## Why

Vectos is already useful for single-project repositories, but it needs better project scoping and broader language coverage to be practical in modern mixed-language monorepos.

## What Changes

Add workspace-aware indexing with Nx as the first-class monorepo target for the first implementation phase, plus broader source/config file support with an explicitly limited file-type set.

- **New Capability: `workspace-awareness`**: Detect project boundaries in monorepos, starting with Nx.
- **New Capability: `language-expansion`**: Expand indexing and chunking support across a bounded set of additional source and infrastructure files.

## Capabilities

### New Capabilities
- `workspace-awareness`: Supports indexing a logical project inside an Nx monorepo using Nx workspace metadata and explicit project selection.
- `language-expansion`: Adds first-pass support for Java and a fixed set of infrastructure/config file types: Dockerfile, `docker-compose*.yml`, `*.yml`, `*.yaml`, `BUILD`, `BUILD.bazel`, `WORKSPACE`, `MODULE.bazel`, and `*.bzl`.

## Impact

- New Nx workspace resolution logic.
- New indexing rules for a logical project selected explicitly from an Nx workspace.
- New file classification and baseline chunking heuristics for the added file-type set.
