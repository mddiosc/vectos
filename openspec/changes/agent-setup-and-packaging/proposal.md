## Why

Vectos is now usable, but adoption friction is still too high. Setup should be adapter-based across supported agent clients, and installation should feel like a real CLI product rather than a local project binary.

## What Changes

Add adapter-based multi-agent setup automation and a source-based installation path suitable for installation like a real CLI.

- **New Capability: `agent-setup`**: Configure supported agent clients through reusable per-agent adapters.
- **New Capability: `distribution-packaging`**: Provide a real installable CLI workflow centered on source-based installation and global binary usage.

## Capabilities

### New Capabilities
- `agent-setup`: Automates setup for supported agents with validated config targets, while keeping unsupported or unstable targets out of the first implementation phase.
- `distribution-packaging`: Provides a global binary UX and installation documentation for building and installing `vectos` from source.

## Impact

- New reusable setup adapter layer for agent clients.
- Source-based installation workflow for a global binary.
- Install UX shift from `./vectos` to global `vectos` command usage.
