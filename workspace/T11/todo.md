# T11 Residual Notes

## Blocking Residuals

- None currently known.
- Final comparison is recorded in `workspace/T11/final_compare.md`; it found no unrecorded high-severity parity or boundary risk.

## Deferred By Design

- T12 owns Linear write behavior, including `linear_graphql`, comment creation, and issue state mutation.
- Later runtime integration tasks own constructing the Linear reader from full runtime config and wiring it into the orchestrator.

## Verification Notes

- `make test-e2e` was run as a broad confidence gate even though T11 is package-scoped and not wired into an end-to-end Linear runtime path yet.
- The T11-specific proof remains `go test -count=1 ./internal/trackers/linear/...`.
