## Context

`T05` froze the provider-neutral runtime vocabulary in `internal/domain`, but `internal/orchestrator` is still an empty package. The approved design already expects one runtime owner for polling, claims, running items, retries, stall recovery, and snapshot generation, while later tasks still own tracker-interface freeze (`T10`), workspace lifecycle (`T07`), runner semantics (`T08`), and Codex protocol parsing (`T09`).

The Elixir reference implementation proves the required behavior, but it mixes those behaviors with GenServer refs, task supervision, Linear-normalized issue structs, and concrete workspace cleanup calls. `T06` must preserve the behavior while keeping the Go core provider-neutral and without freezing the wrong cross-package abstractions too early.

## Goals / Non-Goals

**Goals:**

- land the first real provider-neutral scheduler core in `internal/orchestrator`
- keep all mutable scheduling state private to the orchestrator
- preserve the proven polling, dispatch, retry, reconcile, stall, and snapshot semantics from the reference implementation
- accept worker lifecycle input only through `domain.RunEvent`
- project runtime state only through `domain.Snapshot`
- keep collaborator seams package-private until later tasks prove the right tracker/workspace/runner/Codex interfaces
- make the behavior precise enough that package-scoped TDD can lock it in

**Non-Goals:**

- freezing `internal/tracker` in this task
- implementing workspace hooks or actual workspace deletion
- implementing local versus SSH host-selection policy or per-host capacity rules
- parsing raw Codex protocol payloads in `internal/orchestrator`
- creating HTTP, dashboard, or web DTOs
- widening `internal/domain` with new helper types just to make the scheduler easier to write

## Decisions

### 1. Use one private orchestrator state model and keep collaborator seams package-private

`internal/orchestrator` will own one private runtime state model containing:

- normalized active/terminal state sets
- poll cadence and stale-tick guards
- claimed item ids
- running entries
- retry entries
- aggregate Codex totals
- latest rate-limit view

The package may use timers and goroutines internally, but no mutable state escapes the package.

Later integrations will plug in through package-private seams for:

- listing candidate items
- refreshing items by id
- starting a run
- stopping a run with cleanup intent
- optional host admission hints
- clock/timer control

Alternative considered:

- export tracker/workspace/runner/Codex interfaces now so the package looks “complete”. Rejected because `T07` to `T10` still own those boundaries, and freezing them in `T06` would likely lock in the wrong abstractions.

### 2. Preserve source-faithful scheduling order and reconcile-before-dispatch

Each poll cycle will:

1. enter `checking=true`
2. reconcile stalled runs
3. refresh and reconcile currently running items
4. consume due retries
5. fetch, sort, revalidate, and dispatch new candidates
6. schedule the next poll and clear `checking`

Candidate ordering stays source-faithful:

- priority ascending for `1..4`
- `CreatedAt` oldest first
- identifier / id tie-breaker

Dispatch gating stays source-faithful:

- active and non-terminal state required
- `Routable != false`
- blocked `Todo` items wait for blockers to become terminal
- claimed/running items do not redispatch
- global and per-state concurrency limits must permit dispatch

Alternative considered:

- dispatch first and reconcile later for simpler code. Rejected because the reference behavior refreshes running truth before new dispatch so missing, terminal, unroutable, and stalled work cannot linger and distort capacity.

### 3. Freeze retry lineage, claim retention, and stale-delivery guards explicitly

Retry behavior will distinguish continuation from failure while keeping one explicit attempt lineage:

- normal completion always schedules continuation retry attempt `1` with ~`1s` delay
- the claim remains held until retry revalidation decides to redispatch or release
- once a continuation retry redispatches into a new run, any later failure advances from that run's carried attempt lineage
- failure retry delay is `min(10s * 2^(attempt-1), maxRetryBackoff)` for `attempt >= 1`
- due retries that cannot redispatch because refresh fails or slots are unavailable are rescheduled as the next failure attempt
- retry entries carry a private sequence token so stale scheduled deliveries are ignored

Alternative considered:

- treat continuation and failure as unrelated counters or drop the claim immediately on failure. Rejected because both would diverge from the reference implementation and would either allow duplicate dispatch or create ambiguous retry state.

### 4. Keep snapshot and aggregate counters precise but projection-only

`domain.Snapshot` remains the only outward runtime projection.

Projection rules:

- `Running` sorts by `ItemIdentifier`, then `ItemID`, then `StartedAt`
- `Retrying` sorts by `DueAt`, then `ItemIdentifier`, then `ItemID`
- aggregate `CodexTotals` are lifetime cumulative since orchestrator start
- repeated worker events that report cumulative per-run totals add only the delta from that run's last reported totals
- `RateLimits` is the latest non-nil observed rate-limit view, not a merged aggregate

Alternative considered:

- expose private maps directly or keep aggregate counters underspecified. Rejected because later observability tasks need stable projection semantics and deterministic ordering, but must not gain a second mutable runtime store.

## Risks / Trade-offs

- `[T06 exports the wrong collaborator API too early]` → Keep seams package-private and let later tasks promote only what real integrations prove necessary.
- `[Retry semantics drift from the reference implementation]` → Lock claim retention, attempt progression, stale-token handling, and continuation versus failure behavior in package tests.
- `[Aggregate token counters double-count cumulative worker reports]` → Store the last reported cumulative totals per run and add only deltas to orchestrator-wide counters.
- `[Startup/checking semantics are silently simplified away]` → Add a service-level test that proves immediate startup polling and refresh coalescing behavior instead of relying only on manual tick helpers.

## Migration Plan

This change is internal-only:

1. land the orchestrator package with package-scoped behavior tests
2. keep collaborator seams private until real tracker/workspace/runner/Codex integrations arrive
3. let later tasks replace seams with concrete implementations without moving scheduling ownership out of `internal/orchestrator`

Rollback is straightforward: revert the `internal/orchestrator` package changes and return the package to a placeholder, at the cost of blocking downstream runtime tasks again.

## Open Questions

- None for this change. The remaining boundaries are intentionally deferred to `T07` through `T10`, not left ambiguous inside `T06`.
