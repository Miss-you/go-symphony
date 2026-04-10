## Context

`T05` froze the provider-neutral runtime vocabulary and `T06` proved the orchestrator can schedule against normalized `domain.WorkItem` values, but the Go repo still has no concrete tracker-core boundary. `internal/tracker` and `internal/trackers/memory` remain placeholder packages, while the approved design already expects a read-only tracker contract in the core and a memory path that can support local/test work without Linear.

The Elixir reference implementation exposes five tracker callbacks, but only three are true runtime reads:

- candidate listing for normal dispatch polling
- state-based listing for startup terminal-workspace cleanup
- refresh by ID for reconcile, retry revalidation, and worker continuation checks

The Go design intentionally narrows the core boundary relative to Elixir by moving writes out of the core. `T10` must preserve the real read semantics without freezing write APIs or prematurely wiring unfinished runtime packages together.

## Goals / Non-Goals

**Goals:**

- freeze a provider-neutral, read-only `TrackerReader` contract in `internal/tracker`
- keep the contract aligned with the Symphony spec and the approved Go design
- land a deterministic in-memory implementation in `internal/trackers/memory`
- make returned `domain.WorkItem` values safe for callers by eliminating shared slice/pointer aliasing
- lock the boundary with package-scoped tests rather than deferring correctness to later integration work

**Non-Goals:**

- adding tracker write methods to the core
- implementing Linear GraphQL behavior, pagination, routing, or error mapping
- changing `internal/orchestrator` in this task
- wiring startup cleanup into `internal/workspace`
- adding memory-only mutation or event hooks just because the Elixir compatibility boundary has them

## Decisions

### 1. Freeze exactly three read methods in `TrackerReader`

The core tracker contract will export:

- `ListCandidates(context.Context) ([]domain.WorkItem, error)`
- `ListByStates(context.Context, []string) ([]domain.WorkItem, error)`
- `RefreshByIDs(context.Context, []string) ([]domain.WorkItem, error)`

This is the narrowest interface that still matches:

- the Symphony tracker integration contract
- the Elixir orchestrator and worker continuation flow
- the approved Go design note that core keeps tracker reads only

Alternative considered:

- freeze only the two methods the current Go orchestrator private seams already resemble. Rejected because the Symphony spec already proves state-based listing is part of restart recovery, and omitting it now would freeze a knowingly incomplete core contract.

### 2. Keep `T10` scoped to tracker packages only

`T10` will not edit `internal/orchestrator`, even though `T06` deferred tracker-interface freeze to this task.

Reason:

- the formal `T10` gate only covers `internal/tracker` and `internal/trackers/memory`
- changing orchestrator wiring here would broaden the task into a runtime-package change without matching verification
- later tasks already own runtime adoption and broader integration

Alternative considered:

- swap the private orchestrator callbacks to `TrackerReader` now. Rejected because it would couple `T10` to unowned runtime verification and blur the task-board scope boundary.

### 3. Make memory-reader isolation explicit and deep

The memory adapter will:

- keep a private copy of the seeded `[]domain.WorkItem`
- return deep copies on every read
- clone `Labels`, `BlockedBy`, `Priority`, `Routable`, `CreatedAt`, and `UpdatedAt`

This keeps tests deterministic and prevents callers from mutating adapter-internal state accidentally through slices or pointers.

Alternative considered:

- use shallow copies because `domain.WorkItem` is "mostly values". Rejected because slices and pointer-backed fields would still alias stored data and create subtle test pollution.

### 4. Keep the memory adapter read-only in `T10`

The Elixir memory adapter also emits comment/state-update notifications, but those belong to a broader compatibility boundary that the Go core intentionally narrows.

`T10` therefore lands only the read behavior. If later tests or compatibility-shell work need memory-local mutation helpers, they can be introduced as concrete adapter helpers without widening `TrackerReader`.

Alternative considered:

- preserve Elixir memory adapter writes now for parity completeness. Rejected because that would reintroduce writes into the core-adjacent boundary before `T12` owns provider-specific write behavior.

## Risks / Trade-offs

- `[ListByStates looks broader than the current Go runtime usage]` → Keep it because the Symphony spec already proves restart cleanup depends on it, but document that Go runtime adoption is deferred.
- `[Later tasks may misuse `TrackerReader` as a policy surface]` → Keep method names and requirement text read-only and provider-neutral; dispatch and cleanup policy stay outside the interface.
- `[Memory adapter copy isolation drifts over time]` → Add tests that mutate returned slices and pointers and assert the adapter's stored seed data remains unchanged.
- `[Core contract silently widens later]` → Lock the exported method set and semantics with package-scoped contract tests in this change.

## Migration Plan

This change is internal-only:

1. add the `TrackerReader` interface under `internal/tracker`
2. add the memory-backed implementation under `internal/trackers/memory`
3. add package tests that freeze the contract and adapter semantics
4. leave runtime adoption and provider-specific readers to later tasks

Rollback is straightforward: revert the tracker and memory package changes and return both packages to placeholders, at the cost of blocking later tracker/integration work again.

## Open Questions

- None for `T10`. The remaining uncertainty is intentionally deferred to later integration and provider-specific tasks rather than left ambiguous inside this change.
