# T08 Residual Notes

## Final Compare

Compared against `workspace/T08/original_impl.md`, `workspace/T08/final_impl.md`, the approved design, and the implemented code.

Current result:

- `internal/runner` owns local command execution, SSH command construction, SSH config handling, host/port parsing, result normalization, timeout surfacing, and stateless worker-host selection.
- `internal/workspace` keeps path safety, create/reuse/remove policy, hook ordering, and cleanup fan-out while delegating command execution to runner.
- `internal/orchestrator` keeps mutable running/claim/retry state and derives host loads from private running entries before calling runner host selection.
- Codex app-server protocol and session lifecycle remain deferred to T09.

No unrecorded high-severity parity risk remains for T08.

## Residual / Deferred Work

- T09 must consume the runner executor for Codex app-server process/session lifecycle.
- T14 must exercise the full end-to-end run path across tracker, workspace, runner, Codex, toolbridge, and workflow.
- Real SSH network behavior is intentionally covered by argv construction and injected process tests in T08; live SSH integration belongs to later end-to-end wiring.

## Verification Evidence

Fresh post-review verification passed:

- `go test ./internal/runner/... ./internal/workspace/... ./internal/orchestrator/...`
- `go test ./...`
- `make build`
- `make lint`
- `make test-e2e`
