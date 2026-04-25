## Context

Vectos currently documents only a source-based installation flow. There is no changelog file, no release automation, and no build metadata injection beyond a hardcoded MCP version string. The next milestone is not a stable public release; it is an experimental/internal GitHub distribution that proves Vectos can be downloaded and used without cloning the repository first.

The repository also has native-release constraints: `go-sqlite3` introduces CGO considerations, and the embedded provider downloads ONNX Runtime shared libraries by platform. That makes a broad first release risky, so the design should optimize for a narrow, validated release contract rather than maximum platform coverage.

## Goals / Non-Goals

**Goals:**
- Introduce a single version source that can be injected at build time and exposed by both the CLI and MCP server.
- Define a simple changelog process suitable for experimental/internal releases.
- Enable experimental GitHub release assets for a narrow initial platform matrix.
- Document the supported download-and-install flow for release assets while keeping source install as a fallback.
- Make the release posture explicit: SemVer `0.x`, experimental/internal, limited platform support.

**Non-Goals:**
- Shipping Homebrew support in this phase.
- Supporting Windows in the first release milestone.
- Fully automating release notes or changelog generation from commits.
- Guaranteeing stable CLI, MCP, or packaging contracts beyond the documented experimental scope.

## Decisions

- Use SemVer `0.x` tags for the first release phase.
  Rationale: Vectos is still evolving quickly, and `0.x` communicates that interfaces and packaging may still change.
- Introduce a dedicated version-reporting capability rather than leaving version strings scattered in the codebase.
  Rationale: release metadata should be sourced consistently for CLI output, MCP metadata, and GitHub release assets.
- Keep changelog maintenance manual in the first phase.
  Rationale: the release cadence is still low and internal; manual curation is simpler and yields higher-quality release notes at this stage.
- Scope the first downloadable release to `darwin/arm64` and `linux/amd64`.
  Rationale: these targets minimize native packaging risk while covering the most likely early adopters.
- Treat GitHub releases as experimental/internal artifacts, not a stable public distribution contract.
  Rationale: this keeps messaging aligned with the current maturity of Vectos and avoids overcommitting to support and compatibility expectations.

## Risks / Trade-offs

- Native build dependencies can make cross-platform releases fragile. -> Start with a minimal validated target matrix and expand only after hands-on verification.
- Manual changelog maintenance can drift if not enforced during releases. -> Require changelog updates as part of the release workflow and tasks.
- Release assets may create the impression of stability that the project does not yet have. -> Add explicit experimental/internal messaging to documentation and release notes.
- Source install and release install can diverge over time. -> Keep both flows documented and validate the release asset path explicitly in the first milestone.

## Migration Plan

1. Add a version-reporting path in the application and remove hardcoded release strings.
2. Introduce `CHANGELOG.md` and document the release note structure.
3. Add release packaging configuration and GitHub workflow for the initial two platforms.
4. Update README to document experimental/internal download-based installation.
5. Validate the release flow end to end before treating it as the preferred experimental distribution path.

## Open Questions

- Whether the first tagged experimental release should be `v0.1.0` or a later `v0.x.0` remains open until implementation starts.
- Whether `linux/arm64` is close enough to include in a follow-up release depends on validation results after the first milestone.
