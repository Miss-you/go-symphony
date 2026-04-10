# T06 Final Implementation V1

## Task Goal

`T06` lands the first real `internal/orchestrator` implementation: one provider-neutral runtime owner for poll cadence, dispatch eligibility, retry bookkeeping, reconcile, stall recovery, and snapshot projection.

This task is the point where the Go port stops treating orchestration as design prose and starts enforcing it in code.

## Scope

`T06` must implement:

- single-owner mutable runtime state inside `internal/orchestrator`
- poll state transitions, including immediate initial poll and refresh coalescing
- deterministic candidate ordering and dispatch gating based on the already-frozen `domain.WorkItem` contract
- retry scheduling for continuation and failure paths
- running-item reconcile against refreshed item state
- stale-run detection from worker activity timestamps
- projection of private runtime state into `domain.Snapshot`
- worker-to-orchestrator updates via `domain.RunEvent` only

`T06` does not finalize:

- the long-term exported tracker interface; `T10` still owns that freeze
- workspace creation/removal behavior; `T07` owns those semantics
- local versus SSH execution behavior and host-capacity policy; `T08` owns that integration
- Codex app-server protocol parsing; `T09` owns that normalization
- compatibility-shell API/dashboard/web DTOs; later tasks project from `domain.Snapshot`

## Core Design Decision

The Go orchestrator should be a narrow service with private state and reducer-style transition helpers. The service may use timers and goroutines internally, but the only durable rule is that mutable scheduling state never escapes the package and is never mutated directly by worker code.

The implementation should follow this ownership split:

- `internal/config` provides typed scheduler inputs
- `internal/domain` provides the stable data vocabulary
- `internal/orchestrator` owns all mutable runtime decisions
- later packages provide real tracker/workspace/runner/Codex integrations without moving ownership away from the orchestrator

## Private Runtime State

Keep the following state private to `internal/orchestrator`:

- normalized active-state and terminal-state sets
- poll loop state:
  - `interval`
  - `nextPollAt`
  - `checking`
  - internal refresh/tick token or equivalent stale-message guard
- claimed item registry keyed by item id
- running item registry keyed by item id
- retry registry keyed by item id
- aggregate `domain.CodexTotals` as lifetime cumulative usage since orchestrator start
- latest `*domain.RateLimits` as the most recent non-nil rate-limit view observed from any worker event

Each running entry should retain only the private facts needed to drive policy and later project `domain.ActiveRun`:

- current `domain.WorkItem`
- current retry lineage attempt number carried into the active run
- started time
- last activity time
- last event kind and message
- session id
- turn count
- worker host
- workspace path
- per-run `domain.CodexTotals`
- last reported per-run cumulative totals used to derive aggregate deltas without double-counting repeated cumulative updates

Each retry entry should retain only the private facts needed to drive policy and later project `domain.RetryEntry`:

- item id and identifier
- retry attempt number
- due time
- last error
- worker host
- workspace path
- stale-delivery guard token / sequence

Do not mirror Elixir timer refs, PIDs, or monitor refs into `internal/domain`.

## Scheduler Rules To Preserve

### Polling

`T06` should preserve the Elixir-visible polling behavior in Go-native form:

- startup schedules an immediate poll
- entering a poll marks `checking=true` and clears the next due instant
- finishing a poll schedules `nextPollAt = now + interval` and clears `checking`
- manual refresh requests coalesce if a poll is already active or already due
- stale tick deliveries are ignored

This is required because the snapshot contract needs to show both “checking now” and “next poll at” states.

### Candidate Ordering

Candidate ordering must stay deterministic:

1. lower numeric priority first
2. older `CreatedAt` first
3. identifier as stable tie-breaker

Missing or invalid priority sorts after explicit priority `1..4`. Missing timestamps sort after real timestamps.

### Dispatch Gating

An item is dispatchable only if all of these hold:

- it has the same minimum identity fields the current runtime requires for safe dispatch: `ID`, `Identifier`, `Title`, and `State`
- it is in an active state and not in a terminal state
- `Routable != false`
- a `Todo` item is not blocked by any non-terminal blocker
- it is not already claimed
- it is not already running
- a global concurrency slot is free
- the per-state concurrency limit is not exhausted

`T06` should revalidate a candidate immediately before starting a run, using the latest refreshed item view rather than the stale list result.

## Retry Design

The orchestrator owns retry policy. Workers do not directly mutate the retry queue.

Preserve two distinct retry paths:

- continuation retry after a normal completion:
  - completion always schedules retry attempt `1`
  - retry delay is fixed at about `1s`
  - this explicit `attempt=1` continuation retry is the start of a new retry lineage seeded by normal completion
  - the item stays claimed until retry revalidation decides whether it is still active, visible, and routable
  - the later refresh decides whether the item still needs another run
  - if revalidation succeeds but dispatch is blocked by capacity or concurrency, reschedule as failure attempt `2` with the normal failure backoff path rather than repeating the short continuation delay indefinitely
- failure retry after crash/stall/restart-worthy error:
  - the item also stays claimed until retry revalidation resolves the item to redispatch or release
  - if the failed run had no prior retry lineage attempt, schedule attempt `1`
  - if the failed run already came from retry attempt `N > 0`, schedule attempt `N+1`
  - if a due retry cannot redispatch because refresh failed or slots are unavailable, reschedule as attempt `currentAttempt+1`
  - delay for failure attempt `N` is `min(10s * 2^(N-1), settings.Agent.MaxRetryBackoffMS)` for `N >= 1`
  - capped by `settings.Agent.MaxRetryBackoffMS`

Retry entries must retain worker host and workspace path so later compatibility surfaces can still show where the previous run lived.
Continuation and failure share the same attempt lineage once a continuation retry redispatches the item into a new run. A later failure from that run advances from the run's carried attempt number rather than resetting to `1`.

`T06` should also preserve stale-delivery protection. If an older scheduled retry fires after a newer retry replaced it, the old delivery must be ignored.

## Reconcile And Stall Recovery

Every poll cycle should start with reconcile before new dispatch.

Reconcile rules:

- if a running item is still active and still routable, refresh the stored item and keep the run
- if it moved to a terminal state, stop the run and drop claim/run state
- if it left active states, stop the run and drop claim/run state
- if it is no longer routable, stop the run and drop claim/run state
- if it disappears from the refresh response, stop the run and drop claim/run state

Stall recovery is separate from state-based reconcile:

- compute inactivity from the last worker activity time, or the run start time if no later activity exists
- if inactivity exceeds `settings.Codex.StallTimeoutMS`, stop the run and schedule a failure-style retry

The decision to stop/cleanup can flow through package-private hooks in `T06`; concrete workspace cleanup behavior is still finalized in `T07`.

## Worker Event Handling

`domain.RunEvent` is the only worker-to-orchestrator mutation input.

`T06` should accept events and update private runtime state as follows:

- workspace/path/runner-host events update the stored runtime context
- Codex events update last activity time, session id, per-run totals, and latest rate limits
- turn-completed increments turn count and stamps the latest event metadata
- run-completed removes the running entry, keeps the claim, and schedules continuation retry attempt `1`
- run-failed removes the running entry, keeps the claim, and schedules the next failure retry attempt from the run's carried attempt lineage
- worker-emitted `retry_scheduled` may update last-event metadata only; it must not let workers write or replace private retry entries directly

The orchestrator may emit its own internal retry bookkeeping, but workers must never insert or rewrite retry entries directly.
When worker events report cumulative per-run token totals repeatedly, the orchestrator's aggregate `domain.CodexTotals` must add only the delta from that run's previously reported cumulative totals so lifetime counters never double-count.

## Dependency Strategy

Because `T07` through `T10` are not finished, `T06` should use package-private seams for:

- listing candidate items
- refreshing items by id
- starting a run
- stopping a run
- optional host preference / capacity selection
- clock/timer control

These seams are allowed because they are local test and wiring hooks, not frozen cross-package contracts. Do not export a speculative tracker/runner/workspace/Codex abstraction just to make `T06` look “clean”.
`T06` should not export collaborator interfaces, dependency bundles, or constructor shapes that later tasks would be forced to preserve before the real integrations exist.

For host handling specifically, `T06` should preserve worker-host metadata on running and retry entries, and it may accept an optional package-private host-selection hook so later `T08` can plug in SSH/local capacity policy without moving orchestration ownership out of `internal/orchestrator`.
That host behavior is limited to carried metadata and admission hints in `T06`; actual host-selection policy and per-host capacity semantics are still deferred to `T08`.

## Package Surface

Keep the package surface intentionally small in `T06`.

It is acceptable for the first landed implementation to keep most wiring helpers package-private, because no other package depends on `internal/orchestrator` yet. The durable contract at this stage is behavioral:

- orchestrator owns mutable state
- workers report via `domain.RunEvent`
- observers read `domain.Snapshot`

Do not widen the public API until later tasks prove a real need.

## Snapshot Contract In Go

The orchestrator snapshot must be built only from private state and already-frozen `domain` types.

`domain.Snapshot` should contain:

- `Running []domain.ActiveRun`
- `Retrying []domain.RetryEntry`
- `Polling domain.PollingState`
- aggregate `CodexTotals`
- latest `RateLimits`

Projection rules:

- sort `Running` by `ItemIdentifier`, then `ItemID`, then `StartedAt`
- sort `Retrying` by `DueAt`, then `ItemIdentifier`, then `ItemID`
- derive `Polling.NextPollAt` from orchestrator state only
- never expose internal timer handles, goroutine refs, or dependency objects

## TDD Plan

The minimum red/green cycle set for `T06` is:

1. sorting and dispatch gating
2. refresh coalescing and poll countdown/checking transitions
3. reconcile keeps active items and drops invalid running items
4. normal completion schedules short continuation retry
5. failure completion schedules exponential backoff with cap
6. stale retry deliveries are ignored
7. stall detection schedules failure retry
8. failure retry keeps claims until retry revalidation resolves them
9. retry lineage advances exactly as specified for first failure, repeated failure, and rescheduled due retry
10. startup checking state, refresh coalescing, and stale tick suppression are all covered by service-level tests
11. terminal versus non-terminal invalidation preserves the correct cleanup intent
12. snapshot projection matches private running/retrying/polling state with the exact stable sort keys
13. worker events update running metadata, cumulative aggregate totals, latest rate limits, and `retry_scheduled` metadata without leaking retry-queue ownership

These tests should run at `go test ./internal/orchestrator/...` and prove policy, not just compilation.

## Main Risks

### Risk: Freezing later-package boundaries too early

If `T06` exports a tracker/runner/workspace/Codex interface now, it will likely be the wrong shape once `T07` through `T10` land.

### Risk: Making observability a second runtime owner

If snapshot data is cached or mutated outside the orchestrator, later API/dashboard work will drift from the real scheduler state.

### Risk: Treating completion as terminal success

The current Symphony behavior uses completion as a continuation checkpoint, not proof that the item is forever done.

### Risk: Reintroducing provider-specific runtime state

`internal/orchestrator` must operate on `domain.WorkItem` and `domain.Blocker`, not Linear-shaped issue payloads or provider config.

## Acceptance Direction

`T06` is ready to move forward only if:

- `go test ./internal/orchestrator/...` proves the exact sort, gating, retry lineage, claim-retention, reconcile, stall, and snapshot rules above
- the tests prove aggregate `CodexTotals` are lifetime cumulative and `RateLimits` are the latest non-nil observed view
- workers mutate orchestrator state only by sending `domain.RunEvent`
- the implementation exposes exactly one `domain.Snapshot` projection surface and no second runtime truth store
- the landed package stays provider-neutral and does not freeze later `T07` to `T10` interfaces early
