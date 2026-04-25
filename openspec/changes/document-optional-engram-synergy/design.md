## Context

The user explicitly wants Vectos to remain a standalone product. Any Engram integration must be additive and optional.

## Goals

- Make the combined memory + code workflow explicit and recommended
- Preserve Vectos standalone value and messaging
- Avoid introducing hard runtime or storage coupling between Vectos and Engram

## Non-Goals

- Sharing a mandatory database or runtime process with Engram
- Requiring Engram for Vectos indexing or search
- Reworking Vectos into a general memory product

## Approach

Treat Engram synergy as an optional layer above Vectos standalone:

1. document the combined workflow
2. refine agent guidance so Vectos tools and memory tools complement each other
3. optionally add small helper affordances later, but only if they preserve standalone behavior

## Design Decisions

### Standalone-first messaging

Docs and specs should make it clear that Vectos solves semantic code context on its own.

### Optional synergy contract

When both tools are present, the recommended flow should be:

- recover prior memory
- retrieve relevant code with Vectos
- read only the focused files needed for the task

### No shared persistence requirement

The first implementation should stay at the workflow/documentation layer unless a future helper can be added without coupling the systems.

## Risks

- Accidental product messaging that makes Vectos sound incomplete without Engram
- Over-designing cross-tool integration before standalone retrieval is mature enough

## Validation

- Review docs and guidance for standalone-first clarity
- Confirm that any new guidance still makes sense when Engram is absent
