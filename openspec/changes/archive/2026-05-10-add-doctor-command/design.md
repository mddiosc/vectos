## Context

Users need a single, read-only command that explains why Vectos is unhealthy without requiring them to run several commands and manually compare results. The existing `status` command is close, but it is scoped to index state; `doctor` should broaden that into an install-and-runtime diagnosis flow.

## Goals / Non-Goals

**Goals**
- Provide one command that checks the most common local failure modes.
- Keep the command read-only and safe to run anytime.
- Surface actionable remediation hints instead of raw internal data only.
- Reuse existing provider/index logic rather than duplicating it.

**Non-Goals**
- Change indexing behavior.
- Add networked telemetry or remote health checks.
- Auto-fix any issue.
- Replace `status`; `doctor` is a superset diagnostic entry point.

## Decisions

### D1: `doctor` is read-only
The command SHALL not mutate indexes, config files, guidance blocks, or installed binaries.

Rationale: diagnostics should be safe to run repeatedly and from support instructions.

### D2: `doctor` reuses existing status checks
The command SHALL reuse embedding-provider inspection and index/reindex consistency checks already available to `status`.

Rationale: this avoids drift between commands and keeps the failure messages aligned.

### D3: `doctor` adds install/runtime checks
The command SHALL add checks for version/build info, home/config directory access, model/cache availability, and general runtime readiness before printing index details.

Rationale: the common failure cases are usually installation or environment related, not only index related.

### D4: Exit codes distinguish healthy vs unhealthy states
The command SHALL exit 0 when all critical checks pass and non-zero when any critical check fails.

Rationale: users and scripts need a quick machine-readable success signal.

### D5: Output is sectioned and actionable
The command SHALL print a stable human-readable report with sections such as install, environment, provider, and index, plus short remediation hints for failing checks.

Rationale: the command is primarily for humans debugging local setup.

## Risks / Trade-offs

- Broader checks may produce more console output than `status`, but that is the point of the command.
- Reusing existing logic may expose older wording in some diagnostics; that is acceptable for the first version.
- A stricter exit code can surface hidden issues that were previously only warnings.
