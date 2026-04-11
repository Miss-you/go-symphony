# T14 Test Strategy

## Purpose

T14 turns the already-frozen runtime pieces into a complete run loop. The tests need to prove behavior, not just compilation:

1. startup builds the runtime from the current `config.Store` snapshot and performs cleanup before dispatch begins
2. the memory path stays fully local and never depends on a live Linear client
3. the Linear path wires a fake GraphQL/client through the workflow bundle and reaches the same run loop shape
4. workers refresh state after each turn, respect `agent.max_turns`, and normalize runtime events consistently
5. workspace cleanup, invalidation, and shutdown stay idempotent and stateful in the expected places

The evidence below maps each requirement to the smallest proof that can actually support it.

## Proof Matrix

| Behavior or risk | Check | What it proves |
| --- | --- | --- |
| `config.Store` is read before worker construction and the current snapshot drives startup | `go test ./internal/cli/... ./cmd/symphony/...` | The launcher uses the active settings/workflow snapshot rather than stale cached state, and bootstrap can be exercised without hand-wiring a second config source. |
| Startup cleanup runs before the first dispatch | `go test ./internal/cli/... ./internal/workspace/...` | The process removes leftover workspaces for terminal items and waits for cleanup to finish before orchestrator start, so the runtime does not begin from a dirty workspace set. |
| The memory run path is fully local and has no network dependency | `go test ./internal/cli/... ./internal/orchestrator/... ./internal/trackers/memory/... ./internal/codex/...` | The no-network bundle can drive a complete run with an injected unsupported-tool handler and without instantiating the Linear workflow or HTTP client. |
| The Linear run path uses the workflow bundle plus fake GraphQL/client seams | `go test ./internal/workflow/... ./internal/toolbridge/... ./internal/trackers/linear/... ./internal/codex/... ./internal/cli/...` | The workflow-selected bundle can inject `linear_graphql` and the bridge client shape into Codex startup, while tests keep the transport fake and deterministic. |
| Post-turn refresh happens before the worker decides whether to continue | `go test ./internal/cli/... ./internal/orchestrator/... ./internal/tracker/...` | The worker rereads the current item after each completed turn, so continuation decisions are based on fresh tracker state instead of stale in-memory state. |
| `agent.max_turns` ends a worker run without turning it into a failure | `go test ./internal/cli/... ./internal/orchestrator/... ./internal/codex/...` | A max-turn cap produces a normal run-completed exit, leaves claim/retry control to the orchestrator, and does not misclassify the exit as session failure or terminal cleanup. |
| Event normalization stays stable across success, failure, tool, approval, and malformed Codex protocol messages | `go test ./internal/codex/... ./internal/cli/...` | The runtime emits the expected normalized `RunEvent` set, with one source of truth for turn completion and consistent mapping for session start, tool calls, unsupported tools, and unknown payloads. |
| Terminal cleanup and non-terminal invalidation remain distinct | `go test ./internal/workspace/... ./internal/orchestrator/... ./internal/cli/...` | Terminal states trigger workspace removal, while active or retryable states only invalidate or retain the workspace according to the existing retry path. |
| Shutdown is idempotent and closes the active session exactly once | `go test ./internal/cli/... ./internal/codex/... ./internal/orchestrator/...` | Repeated stop paths do not double-close the session, double-cancel the run, or double-remove the workspace, which is the minimum proof that process teardown is safe to call more than once. |
| Runner/workspace bootstrap remains compatible with the current service seam | `go test ./internal/runner/... ./internal/workspace/... ./internal/orchestrator/...` | The launcher can still create the execution host, workspace manager, and orchestrator without introducing a new scheduler layer or widening package ownership. |
| The repo still builds, lints, and keeps the top-level e2e entrypoint healthy | `go test ./...`, `make build`, `make lint`, `make test-e2e` | The new integration wiring does not break unrelated packages, the normal binary build still works, static checks still pass, and the repo-level e2e command still behaves as a usable gate. |

## Package Test Coverage

The main task gate is `go test ./internal/...`. That broad package sweep should include these focused proofs:

1. `internal/cli`
   - proves startup reads `config.Store` before worker construction
   - proves startup cleanup completes before orchestrator start
   - proves shutdown is idempotent and waits for active worker/session teardown
   - proves the launcher can switch between memory and Linear bundle construction from the current snapshot
2. `internal/codex`
   - proves event normalization is stable for session start, approvals, user input, tool calls, unsupported tools, malformed messages, unknown messages, turn completion, and turn failure paths
   - proves `RunTurn` remains the single source of turn-count and usage totals
3. `internal/orchestrator`
   - proves the worker lifecycle can be driven by run events
   - proves retry, invalidation, and cleanup decisions remain in orchestrator ownership
   - proves continuation scheduling after max-turn exit uses the existing retry machinery instead of worker-side retry logic
4. `internal/workspace`
   - proves terminal cleanup removes workspace state only for terminal items
   - proves non-terminal invalidation preserves the workspace and only drops stale assumptions
   - proves cleanup remains idempotent when called through process shutdown
5. `internal/trackers/memory`
   - proves the injected no-network path can feed a complete run without requiring any Linear client or HTTP transport
6. `internal/trackers/linear` and `internal/toolbridge`
   - prove the Linear fake-client path still exposes the expected reader and `linear_graphql` seams for workflow-selected startup
7. `internal/workflow`
   - proves the selected bundle can be injected into Codex startup and stays compatible with the Linear path

These tests matter because T14 is not adding new business rules. It is stitching existing runtime ownership together, so the right evidence is that each boundary still does the job its owner already promised to do.

## Memory No-Network Path

The memory path needs one explicit proof: the runtime can complete a run with a bundle that never touches a live Linear transport.

Recommended checks:

- construct the no-network memory bundle in `internal/cli` tests with no dynamic tools and an unsupported-tool handler
- drive a worker run against `internal/trackers/memory` and a fake Codex session
- assert that the run completes, emits the expected normalized events, and never requires a Linear GraphQL client or workflow selection
- add a negative assertion that the memory path does not instantiate the Linear workflow bundle, Linear bridge, or Linear HTTP client

This proves the local verification path is real, not just a stubbed branch that still depends on network-adjacent wiring.

## Linear Path With Fake GraphQL And Workflow Bundle

The Linear path should be proven with fake transport seams, not live HTTP.

Recommended checks:

- use `internal/workflow` to select the Linear bundle from the current settings snapshot
- inject a fake GraphQL/client into the bridge and verify `linear_graphql` is advertised through the Codex startup path
- run the worker loop against `internal/trackers/linear` fixtures so candidate and refresh reads feed the same runtime flow as the memory path
- assert the bundle wiring reaches `internal/codex` without an adapter layer that rewrites tool specs or handler identity

This proves the product path remains Linear-specific where it should, while still exercising the same end-to-end orchestration logic as the memory path.

## Turn Loop Semantics

The turn loop needs four behavioral proofs:

1. post-turn refresh happens after every successful turn and before the continuation decision
2. a still-active item with remaining turns continues
3. a still-active item at `agent.max_turns` exits normally instead of failing
4. an inactive or missing item exits normally and lets orchestrator ownership decide the next retry or release step

The smallest useful evidence is a set of worker tests in `internal/cli` or `internal/orchestrator` that inject a fake tracker reader and a fake Codex session:

- first turn succeeds
- refresh returns active, then inactive, then missing in separate cases
- the worker emits `RunEventTurnCompleted` once per successful `RunTurn`
- the worker emits `RunEventRunCompleted` on the inactive, missing, and max-turn exit cases
- max-turn exit does not emit failure metadata or release the claim directly

These cases prove the runtime is actually following the continuation contract described in `final_impl.md`, not just incrementing counters.

## Event Normalization

`internal/codex` needs behavior tests that pin the event mapping matrix rather than a loose “events were seen” assertion.

The tests should prove:

- session start, approval, and user-input events normalize to `RunEventCodexEventReceived`
- supported tool success and failure normalize to the same event family with tool outcome metadata
- unsupported tools and malformed/unknown messages do not escape as raw protocol objects
- `RunEventTurnCompleted` is emitted from the `RunTurn` result path exactly once, with the normalized usage totals attached there
- turn failure, cancellation, and timeout cases normalize to the failure event family instead of being mistaken for a clean exit
- `turn_failed` and `turn_cancelled` preserve their category in the failure event message so later observability can distinguish them

This matters because T14 uses events as the only worker-to-runtime communication channel. If the normalization drifts, the orchestrator can no longer make reliable decisions from the run stream.

## Startup Cleanup, Terminal Cleanup, And Shutdown

The process-level proof should focus on lifecycle edges:

- startup cleanup runs before the first orchestrator dispatch
- terminal-state cleanup removes workspaces only when the reader reports terminal items
- non-terminal invalidation leaves the workspace in place
- shutdown is idempotent even when the launcher sees repeated cancel/stop signals
- `config.Store` closes cleanly and does not leak a stale snapshot into the next run

The right tests here are integration-style tests around `internal/cli`, `internal/workspace`, and `internal/orchestrator` with fakes that capture call order. That call order is the proof: if cleanup happens after dispatch starts, or if shutdown double-closes a session, the test should fail immediately.

## Build, Lint, And E2E Gates

These broader gates are still required because T14 changes a shared startup seam:

1. `go test ./...`
   - proves the runtime wiring compiles and the package changes do not break unrelated code
   - catches contract drift between `cli`, `orchestrator`, `codex`, `workspace`, `runner`, and tracker packages
2. `make build`
   - proves the normal binary build path still works
   - useful as an additional end-to-end compile signal after the package gates pass
3. `make lint`
   - proves the integration code still satisfies repository style and static-analysis checks
   - especially relevant for lifecycle code, event mapping, and cleanup paths where small mistakes usually surface as lint or vet findings
4. `make test-e2e`
   - proves the repository still exposes a runnable top-level e2e gate
   - if the command is not yet meaningful for this slice, that limitation should be recorded explicitly in `workspace/T14/todo.md` rather than hidden

These are broad confidence gates, not substitutes for the targeted proofs above.

## Verification Order

Run verification in this order:

1. package-scoped tests under `go test ./internal/...`
2. focused integration tests for startup, shutdown, memory no-network flow, Linear fake-client flow, turn refresh, max-turn, and event normalization
3. `go test ./...`
4. `make build`
5. `make lint`
6. `make test-e2e`

The package and integration tests prove the runtime behavior. The repo-level gates prove the change still fits the tree as a whole.
