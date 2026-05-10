## 1. Setup guidance flow

- [x] 1.1 Add `setup.Options` in `internal/setup/setup.go` with `Uninstall`, `SkipGuidance`, and `AssumeYes`
- [x] 1.2 Change `Run` to accept `Options` and update the `main.go` call site
- [x] 1.3 Add `--yes` / `-y` and `--no-guidance` to `setupCmd` in `cmd/vectos/cli_flags.go`
- [x] 1.4 Update `cmd/vectos/cli_dispatch.go` to pass the new options through
- [x] 1.5 Remove the TTY prompt gate from `internal/setup/guidance.go` and make guidance upsert unconditional unless skipped
- [x] 1.6 Update `opencode.go`, `claude.go`, and `codex.go` to honor `SkipGuidance`
- [x] 1.7 Refresh `cmd/vectos/cli_help.go` setup help text

## 2. Guidance content

- [x] 2.1 Update `internal/setup/guidance_content.go` to describe the current retrieval flow
- [x] 2.2 Add Nx lib coverage wording: all internal libs are included by default, only `type: "e2e"` is excluded, `VECTOS_NX_INCLUDE_E2E=1` overrides
- [x] 2.3 Add/adjust `internal/setup` tests for append, replace, and skip behavior

## 3. Nx scope resolution

- [x] 3.1 Remove substring-based project exclusion from `internal/workspace/workspace.go`
- [x] 3.2 Keep default e2e exclusion based on Nx graph `type`
- [x] 3.3 Add `VECTOS_NX_INCLUDE_E2E` override handling
- [x] 3.4 Add `Scope.Warnings` and populate it when Nx graph resolution fails
- [x] 3.5 Update Nx workspace tests for included docs-like libs, e2e exclusion, and env override

## 4. Index output

- [x] 4.1 Print detected internal libs in `cmd/vectos/commands_index.go` for every Nx index run
- [x] 4.2 Print resolver warnings so users can see when Nx graph coverage was incomplete
- [x] 4.3 Add a small regression test for the printed lib list

## 5. OpenSpec / release hygiene

- [x] 5.1 Verify the change status becomes apply-ready after tasks are written
