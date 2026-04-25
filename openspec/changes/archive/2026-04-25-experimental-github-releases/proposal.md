## Why

Vectos already supports source-based installation, but the next milestone needs a safer way to distribute experimental internal builds through GitHub without pretending that the project is fully release-hardened. To do that well, Vectos needs explicit release versioning, a changelog process, and a narrow packaging contract for downloadable assets.

## What Changes

- Add a release version reporting capability so Vectos exposes build version, commit, and build date consistently in the CLI and MCP metadata.
- Add a release changelog capability for maintaining a human-readable history of experimental internal releases.
- Extend `distribution-packaging` so Vectos can publish experimental GitHub release assets for a narrow validated platform set instead of supporting only source-based global installation.
- Keep the first release scope intentionally small: SemVer `0.x`, experimental/internal messaging, and initial targets limited to `darwin/arm64` and `linux/amd64`.

## Capabilities

### New Capabilities
- `version-reporting`: expose a single release version contract for CLI and MCP metadata.
- `release-changelog`: maintain a changelog format suitable for experimental/internal releases.

### Modified Capabilities
- `distribution-packaging`: extend packaging from source-based installation only to experimental GitHub release assets with documented installation guidance and explicit scope constraints.

## Impact

- CLI surface and build metadata injection.
- MCP server metadata.
- Release documentation and changelog maintenance.
- GitHub release workflow, release asset generation, and checksum publication.
- Packaging scope and install UX for downloadable experimental builds.
