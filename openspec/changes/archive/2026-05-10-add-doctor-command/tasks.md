## 0. Branch Setup

- [x] 0.1 Create a dedicated feature branch from `main` for this change
- [x] 0.2 Keep the branch up to date with `main` before implementation and review

## 1. Command Wiring

- [x] 1.1 Add `vectos doctor` to CLI dispatch and shared command grouping
- [x] 1.2 Add doctor-specific help text and global command listing
- [x] 1.3 Reuse existing status/provider/index checks where possible

## 2. Diagnostic Output

- [x] 2.1 Implement read-only install/runtime diagnostics
- [x] 2.2 Report provider health and index consistency with actionable hints
- [x] 2.3 Ensure the command exits non-zero on critical failure

## 3. Docs and Tests

- [x] 3.1 Add tests for healthy and unhealthy doctor runs
- [x] 3.2 Add help coverage for global and doctor-specific help
- [x] 3.3 Update troubleshooting docs to point users to `vectos doctor`
