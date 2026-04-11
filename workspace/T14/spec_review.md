# T14 Spec Review

## Verdict

Not approved yet. The spec and implementation plan are mostly aligned, but the task board still understates the verification gate, which is a source-of-truth mismatch for T14.

## High-Severity Issues

1. **T14 task board verification is narrower than the accepted implementation/test plan.**
   - `docs/plans/2026-04-10-go-symphony-design-task.md:47` still lists only `Gate: go test ./internal/...` for T14.
   - `workspace/T14/final_impl.md:149-168` and `workspace/T14/test_strategy.md:17-29` require broader proof: targeted package tests, repo-wide `go test ./...`, `make build`, `make lint`, `make test-e2e`, and explicit handling if e2e remains non-meaningful.
   - This is a real task-board/spec mismatch, not just wording drift. T14 can be closed without the required verification if the board remains narrower than the accepted spec/test strategy.

## Medium / Low Issues

1. **Event normalization still leaves the failure-path matrix slightly under-specified.**
   - `openspec/changes/end-to-end-run-integration/specs/end-to-end-run-integration/spec.md:73-83` and `workspace/T14/final_impl.md:127-132` cover failure/cancellation/timeout in the aggregate, but they do not pin `turn_failed` and `turn_cancelled` as explicit categories.
   - This is probably implementable as written, but the test plan would be stronger if those cases were named directly.

2. **The memory no-network proof should include a negative assertion about Linear bundle instantiation, not only “no network traffic.”**
   - `workspace/T14/test_strategy.md:21` proves the local memory path is runnable, but it does not explicitly assert that the Linear workflow/bridge is never constructed on that path.
   - Given the earlier review history, this is worth tightening so the local closure path cannot regress into “works with a mocked transport” behavior.

## Required Fixes

1. Update the T14 row in `docs/plans/2026-04-10-go-symphony-design-task.md` so the verification gate and done-when text match the accepted T14 plan and test strategy.
2. Add one explicit spec/test assertion for `turn_failed` and `turn_cancelled` normalization.
3. Add a negative assertion for the memory path proving the Linear workflow/client is not instantiated, not just that the run completes locally.
