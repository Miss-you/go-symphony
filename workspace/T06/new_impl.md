# T06 New Implementation Input

## Current Go Baseline

The current Go repo already gives `T06` two stable inputs:

1. Approved architecture and task sequencing in `docs/plans/2026-04-10-go-symphony-design.md` and `docs/plans/2026-04-10-go-symphony-design-task.md`
2. A frozen provider-neutral domain contract from `workspace/T05/final_impl.md` and `internal/domain/types.go`

What does **not** exist yet:

- `internal/orchestrator` logic beyond `doc.go`
- a frozen `internal/tracker` read interface (`T10`)
- workspace lifecycle implementation (`T07`)
- runner host/execution abstraction (`T08`)
- Codex protocol integration (`T09`)

That means `T06` must land the scheduler core without pretending the later packages already exist.

## Constraints The Go Design Already Freezes

From the approved design and `T05` artifacts:

- the orchestrator is the only owner of mutable runtime state
- workers report facts through `domain.RunEvent`
- `observability` stays projection-only and must read `domain.Snapshot`
- core packages remain provider-neutral
- no universal tracker write API, workpad abstraction, or second runtime state store

`internal/domain` already froze:

- `WorkItem`
- `ActiveRun`
- `RetryEntry`
- `PollingState`
- `Snapshot`
- `RunEvent`
- `RunEventKind`
- `CodexTotals`
- `RateLimits`

`internal/config` already provides the scheduler inputs `T06` needs:

- `Settings.Polling.IntervalMS`
- `Settings.Provider.ActiveStates`
- `Settings.Provider.TerminalStates`
- `Settings.Agent.MaxConcurrentAgents`
- `Settings.Agent.MaxConcurrentAgentsByState`
- `Settings.Agent.MaxRetryBackoffMS`
- `Settings.Codex.StallTimeoutMS`

## What T06 Should Implement Now

The Go implementation should land a narrow orchestrator service with private mutable state and package-scoped dependency seams.

Private state should cover:

- normalized active/terminal state sets
- poll interval, next poll instant, and checking flag
- claimed item ids
- currently running items with attempt/session/event/totals metadata
- retry queue entries with attempt, due time, last error, worker/workspace context
- aggregate Codex totals and latest rate-limit view

The public package surface should stay intentionally small. `T06` should not freeze more cross-package API than necessary before `T07` to `T10` land.

## What T06 Should Defer Deliberately

To avoid overdesign and boundary drift, `T06` should **not** finalize:

- the provider-neutral tracker read interface shape for other packages
- workspace creation/removal hooks
- local versus SSH execution host selection semantics
- Codex app-server protocol parsing rules
- compatibility-shell DTOs for API/dashboard/web

Those are later tasks. `T06` only needs the minimal package-private hooks necessary to test poll/reconcile/dispatch/retry decisions in isolation.

## Proposed Go Shape

### Service Ownership

Implement an orchestrator-owned service that:

- accepts config-derived scheduler settings
- stores all mutable scheduling state privately
- exposes snapshot projection from that private state
- accepts worker lifecycle input only as `domain.RunEvent`
- performs poll/reconcile/dispatch/retry transitions through orchestrator methods only

### Package-Private Test Seams

Because `internal/tracker`, `internal/workspace`, `internal/runner`, and `internal/codex` are not ready yet, `T06` should use package-private seams for tests and future wiring:

- candidate fetch
- running-item refresh by id
- run start
- run stop / cleanup intent
- clock and timer control

Keeping those seams package-private avoids prematurely freezing the wrong cross-package interfaces before `T07` to `T10`.

### Snapshot Discipline

The snapshot should be built only from private orchestrator state and already-frozen `domain` types.

That means:

- no process refs or timer handles in `domain.Snapshot`
- stable ordering for `Running` and `Retrying` slices so tests are deterministic
- `PollingState` derived from orchestrator state, not a second runtime cache

## TDD Targets That Matter

The highest-value red/green loops for `T06` are:

1. dispatch gating and sort order
2. reconcile transitions for active, terminal, missing, and unroutable items
3. continuation retry versus failure retry semantics
4. retry backoff capping and retry reschedule behavior
5. stall detection based on last worker activity
6. snapshot projection for running/retrying/polling/aggregate totals
7. refresh coalescing and next-poll scheduling behavior

If those tests pass, `T06` proves the scheduler core rather than just proving that an empty package compiles.

## Main Design Risk

The main risk is trying to solve `T07`, `T08`, `T09`, and `T10` inside `T06`.

The safe implementation line is:

- preserve the orchestration semantics now
- keep dependency seams private for now
- let later tasks replace stubs with real tracker/workspace/runner/Codex integrations
