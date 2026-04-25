## 1. Version Metadata

- [x] 1.1 Add a single version metadata source for release version, commit, and build date
- [x] 1.2 Expose version metadata through a user-facing CLI command
- [x] 1.3 Replace hardcoded MCP version strings with injected build metadata

## 2. Changelog

- [x] 2.1 Add `CHANGELOG.md` with a consistent experimental/internal release format
- [x] 2.2 Seed the changelog with the first release-ready entry structure and known limitations guidance

## 3. Experimental GitHub Packaging

- [x] 3.1 Add release packaging configuration for `darwin/arm64` and `linux/amd64`
- [x] 3.2 Add GitHub workflow automation to build release assets and publish checksums
- [x] 3.3 Document the experimental/internal release posture in packaging and release metadata

## 4. Installation Docs

- [x] 4.1 Update README with download-and-install instructions for experimental GitHub releases
- [x] 4.2 Keep source-based installation documented as a fallback path

## 5. Validation

- [x] 5.1 Validate that the built binary reports consistent version metadata in CLI and MCP paths
- [x] 5.2 Validate the experimental release flow for `darwin/arm64`
- [x] 5.3 Validate the experimental release flow for `linux/amd64`
