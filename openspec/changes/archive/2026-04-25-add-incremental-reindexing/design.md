## Context

Vectos should remain a useful standalone tool. Incremental reindexing must work from the CLI and MCP without assuming any external orchestrator or mandatory git-hook integration.

## Goals

- Reindex only files that changed when enough information is available
- Remove stale chunks for deleted or no-longer-indexable files
- Keep full-project indexing as a simple fallback

## Non-Goals

- Mandatory background daemons
- Mandatory git hooks
- Complex filesystem watchers in the first iteration

## Approach

Introduce an incremental indexing mode that can accept or derive a changed-file set.

Possible sources of changed-file information:

- explicit file paths passed by the user or MCP client
- a repo-aware diff against recent git state when available
- future hook automation layered on top of the standalone feature

The core indexing logic should:

1. resolve the project scope
2. determine the changed file set
3. delete old chunks for those files
4. regenerate chunks and embeddings for files that are still indexable
5. delete stale chunks for files that were removed or that no longer match current indexing policy

## Design Decisions

### Full indexing remains the fallback

The system should not become harder to use just to gain incremental behavior. `vectos index .` must remain valid and predictable.

### Cleanup is part of incremental correctness

Incremental indexing is only correct if it also removes chunks for removed or newly excluded files.

### Hook automation is layered, not required

Git hooks may later call the incremental path, but the capability itself belongs to Vectos standalone.

## Risks

- Incomplete changed-file detection leading to stale indexes
- Over-complicating the CLI before the underlying indexing contract is solid
- Divergence between CLI and MCP indexing semantics

## Validation

- Run `go build ./...`
- Validate incremental updates on modified, deleted, and newly ignored files
- Confirm that full indexing still works unchanged
