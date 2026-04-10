# T06 Residual Notes

## Verification Notes

- `make test-e2e` passed on 2026-04-11 00:53 CST, but at `T06` it is still primarily a repo command-contract and compile check under the `e2e` build tag.
- Meaningful end-to-end runtime proof for real tracker/workspace/runner/Codex orchestration remains deferred to `T07`, `T08`, `T09`, `T10`, and especially `T14`, because those integrations do not exist yet.

## Boundary Notes

- `internal/orchestrator` intentionally keeps its collaborator seams package-private in `T06`. If a later task needs a public constructor or cross-package interface, that promotion should happen only when the real integration path proves the correct shape.

## Final Compare Notes

- Re-checked `T06` against `workspace/T06/original_impl.md` and the current Elixir orchestrator source before closure.
- Retry delivery now matches the required source-faithful behavior: retry entries are both projected and actually scheduled for redispatch through orchestrator-owned timers.
- Candidate dispatch now performs host admission exactly once per attempt, matching the source model where worker-host selection happens inside dispatch rather than as a separate pre-check.
- No additional high-severity parity gap was found within the approved `T06` scope after the review fixes above.
