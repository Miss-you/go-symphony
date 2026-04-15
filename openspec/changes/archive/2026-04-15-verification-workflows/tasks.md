## 1. Read Filter

- [x] 1.1 Add TDD coverage for filtering `TrackerReader` candidate, state, and refresh results by work item ID or identifier.
- [x] 1.2 Implement the provider-neutral read-only filter in `internal/tracker`.

## 2. Verification Command

- [x] 2.1 Add TDD coverage for `symphony-verify linear` argument parsing, provider validation, report rendering, and fake-reader success without network.
- [x] 2.2 Add TDD coverage for `symphony-verify run` guardrails acknowledgement, required `--only-issue`, timeout parsing, and runtime dependency injection.
- [x] 2.3 Implement `cmd/symphony-verify` with `linear` and `run` subcommands.
- [x] 2.4 Add a boundary test proving the `linear` probe path does not import runtime, workspace, orchestrator, or Codex packages.

## 3. Documentation

- [x] 3.1 Add `docs/verification-workflows.md` with staged Linear probe and single-issue runtime smoke instructions.
- [x] 3.2 Document live-run risks, including real Codex execution and possible Linear/workspace mutation.

## 4. Verification

- [x] 4.1 Run targeted tests for `internal/tracker` and `cmd/symphony-verify`.
- [x] 4.2 Run broad build/test/e2e gates and OpenSpec validation.
- [x] 4.3 Record final verification evidence and live-provider manual-smoke limits in `workspace/T19/`.
