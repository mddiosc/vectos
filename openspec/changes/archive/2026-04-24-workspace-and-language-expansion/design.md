## Context

Vectos currently scopes a project by the current working directory name and indexes files directly from the filesystem. That is insufficient for monorepos where one logical project may span multiple roots resolved from workspace metadata.

## Goals / Non-Goals

**Goals:**
- Detect Nx projects and their component roots.
- Add explicit logical-project selection for CLI and MCP in Nx workspaces.
- Expand support beyond current code files into Java and a bounded set of infra/config files.

**Non-Goals:**
- Full semantic understanding of every supported language in the first pass.
- Full dependency graph analysis across all monorepo toolchains.
- Generic manual multi-root path-group definitions in the first implementation phase.

## Decisions

- Nx will be the first workspace implementation target.
- The first phase will require explicit project selection when multiple Nx projects are present.
- CLI and MCP flows will expose project selection explicitly rather than inferring a logical project silently.
- Language support will be staged: classification first, smarter chunking later.
- The first phase file-type set is fixed to: `.java`, `Dockerfile`, `docker-compose*.yml`, `*.yml`, `*.yaml`, `BUILD`, `BUILD.bazel`, `WORKSPACE`, `MODULE.bazel`, and `*.bzl`.

## Risks / Trade-offs

- Monorepo detection can become tool-specific quickly.
- Requiring explicit Nx project selection adds UX work in CLI and MCP, but avoids ambiguous scoping.
- Infra/config files have very different structure than application source files.
- Broader language support increases maintenance pressure for chunking quality.

## Migration Plan

1. Add Nx workspace resolution model.
2. Implement explicit Nx project selection for CLI and MCP.
3. Resolve project roots from the selected Nx project.
4. Expand the bounded file-type set and chunking heuristics.
