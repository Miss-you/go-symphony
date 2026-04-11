# T14 Final Implementation

## Review Gate

`workspace/T14/final_impl_v1.md` passed the required rubric review after one correction round.

Round-one review results:

- `review_1.md`: 74 / 100, not accepted
- `review_2.md`: 73 / 100, not accepted

The rejected version left four blocking points underspecified:

- memory closure could still instantiate a live Linear HTTP-backed bundle
- `agent.max_turns` did not define the final state transition
- Codex-to-runtime event normalization was too vague to test
- bootstrap, `config.Store`, and `cmd/symphony` entrypoint behavior were not pinned clearly enough

Round-two review results:

- `review_1_round2.md`: 88 / 100, no high-severity issues
- `review_2_round2.md`: 93 / 100, no high-severity issues
- average: 90.5 / 100

Acceptance decision:

- average score exceeds the `>= 80` threshold
- no round-two reviewer reported a high-severity issue
- no required changes remain before spec creation

The accepted implementation plan is the revised `workspace/T14/final_impl_v1.md`. The key binding decisions are summarized below so implementation and spec work do not depend on review prose alone.

## Goal

Connect the already-built runtime pieces into an end-to-end Symphony run loop without changing their ownership boundaries.

T14 must wire:

- `tracker.TrackerReader` candidate and refresh reads
- orchestrator scheduling, retry, stall, reconcile, and snapshot state
- workspace create/reuse/hooks/cleanup
- runner-backed local/SSH host selection already owned by orchestrator and runner
- Codex app-server sessions, turns, dynamic tools, and event sink
- workflow-selected Linear dynamic-tool wiring for the Linear path
- a no-network memory closure path for tests/local verification

The deliverable is composition, not a generic runtime framework.

## Non-Goals

T14 does not add:

- a universal tracker write API
- a universal workpad abstraction
- a provider-agnostic default workflow
- HTTP API, terminal dashboard, or web dashboard behavior
- provider-specific fields in `internal/domain`
- Linear GraphQL behavior inside `internal/orchestrator`
- full CLI parity beyond the minimal executable bootstrap needed to start and stop the runtime

## Package Boundaries

Use `internal/cli` as the process assembly boundary and keep `cmd/symphony/main.go` thin.

`internal/cli` owns:

- `config.Store` lifecycle and last-known-good config/workflow snapshots
- tracker reader selection or injection
- Linear workflow selection through `internal/workflow`
- no-network memory bundle injection for tests/local closure
- workspace manager construction
- orchestrator construction and shutdown
- worker handles, cancellation, Codex session close, and cleanup intent execution

`internal/orchestrator` owns:

- polling
- dispatch gates
- claims
- running entries
- retry entries
- stale retry protection
- stall recovery
- snapshots
- all mutable scheduling truth

`internal/orchestrator` must not import `internal/workflow`, `internal/toolbridge/linear`, or provider-specific tracker packages.

## Runtime Flow

Startup must:

1. Create `config.Store`.
2. Read the current workflow and settings snapshot.
3. Build the tracker reader.
4. Build the runtime bundle.
   - Linear uses `workflow.Select`.
   - Memory uses an explicit injected no-network bundle with no dynamic tools and an unsupported-tool handler.
5. Create `workspace.Manager`.
6. Run startup terminal cleanup with `reader.ListByStates` and `workspace.RemoveIssueWorkspaces`.
7. Start orchestrator only after startup cleanup completes.
8. Block until context cancellation, then close orchestrator, active workers, Codex sessions, and the config store idempotently.

`startRun` should return quickly, start a worker goroutine, and report only through `domain.RunEvent`.

`stopRun` should cancel and close the active session idempotently, and should remove the workspace only when orchestrator passes `CleanupWorkspace=true`.

## Turn Loop Semantics

Each worker run uses one Codex session and up to `settings.Agent.MaxTurns` turns.

After every completed turn, the worker must call `TrackerReader.RefreshByIDs` for the current item ID before deciding whether to continue.

If the refreshed item is still active and the turn count is below `agent.max_turns`, the worker runs another turn with Symphony continuation guidance.

If the refreshed item is inactive or missing, the worker emits normal `RunEventRunCompleted` and exits. The orchestrator continuation retry revalidates and releases or cleans up according to existing retry rules.

If the refreshed item is still active but `agent.max_turns` has been reached, the worker also emits normal `RunEventRunCompleted` and exits. This is not a failure, terminal state, or claim release. It returns control to orchestrator, which schedules continuation retry attempt `1`.

## Event Normalization

The worker emits:

- `RunEventRunnerHostSelected` for selected local/SSH host
- `RunEventWorkspaceCreated` for workspace creation or reuse
- `RunEventWorkspacePathDiscovered` once the workspace path is known
- `RunEventCodexEventReceived` for session start, approvals, user-input answers, tool call success/failure, unsupported tools, malformed messages, and unknown messages
- `RunEventTurnCompleted` exactly once per successful `RunTurn` result, carrying normalized usage totals
- `RunEventRunFailed` for session bootstrap, workspace hook, prompt render, refresh, turn failure/cancellation/timeout, and other worker failures
- `RunEventRunCompleted` for normal worker exit after inactive/missing item or max-turn cap

Do not emit `RunEventTurnCompleted` from both the Codex event sink and the `RunTurn` result. The `RunTurn` result is the single source for turn count and token totals.

## Test Strategy Summary

Implementation must follow TDD and include behavior-level tests for:

- exported orchestrator service seam
- startup cleanup before first dispatch
- memory no-network end-to-end run
- post-turn refresh and max-turn normal completion
- Linear-backed run with `linear_graphql` advertised through `thread/start`
- Codex event normalization
- continuation retry and failure retry metadata
- terminal cleanup versus non-terminal invalidation
- bootstrap/main delegation and idempotent shutdown
- `config.Store` lifecycle and current snapshot use before worker creation

## Verification Plan

Primary targeted gate:

```bash
go test ./internal/orchestrator/... ./internal/cli/... ./cmd/symphony/...
```

Broader gates:

```bash
go test ./internal/...
go test ./...
make build
make lint
make test-e2e
openspec validate --type change end-to-end-run-integration
```

If `make test-e2e` is not yet meaningful after T14, record the exact reason in `workspace/T14/todo.md`.

## Risks and Deferred Work

Risks:

- runtime glue can become a second scheduler if worker handles start making retry decisions
- prompt rendering can drift from Symphony if it grows beyond the needed compatibility variables
- hot reload may not update every already-initialized scheduler field until later full CLI parity
- Codex protocol changes may require later event mapping additions

Deferred work remains T15-T18: HTTP API, terminal dashboard, web dashboard, and full CLI parity.
