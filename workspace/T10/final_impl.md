# T10 Final Implementation

## Review Gate

`final_impl_v1.md` passed rubric review after one correction round.

Round-two review results:

- `review_1_round2.md`: 95 / 100, no high-severity issues
- `review_2_round2.md`: 90 / 100, no high-severity issues
- average: 92.5 / 100

Key review corrections accepted into this final plan:

- remove `internal/orchestrator` wiring changes from `T10` so the task scope matches the recorded verification gate
- make memory-reader deep-copy semantics explicit for slice and pointer-backed `domain.WorkItem` fields
- keep `ListByStates` in the frozen interface for Symphony contract fidelity, while explicitly deferring runtime adoption to later tasks

Acceptance decision:

- average score exceeds the `>= 80` threshold
- no reviewer reported a remaining high-severity issue
- remaining notes are documentation-boundary reminders and are incorporated below

## Task Goal

`T10` freezes the provider-neutral, read-only tracker contract for the Go core and lands a memory-backed adapter that can drive local and test flows without Linear.

This task is where `internal/tracker` stops being a placeholder and becomes a durable core boundary that later runtime and provider packages must consume.

## Final Scope

`T10` lands:

- the first real `internal/tracker.TrackerReader` interface
- a deterministic memory-backed implementation in `internal/trackers/memory`
- package-scoped tests that freeze the tracker read contract and memory semantics

`T10` does not land:

- tracker write methods in the core
- Linear GraphQL queries, pagination, assignee routing, or error mapping
- workflow selection or toolbridge behavior
- a provider-agnostic default workflow
- edits to `internal/orchestrator` or other runtime packages whose formal gate is not owned by `T10`
- exported runtime-assembly APIs for unfinished `workspace`, `runner`, `codex`, or end-to-end integration tasks

## Final Tracker Boundary

The core tracker contract should freeze exactly three read operations, because Symphony's language-agnostic spec and the Elixir runtime both rely on all three:

1. candidate listing for dispatch polling
2. state-based listing for startup terminal-workspace cleanup
3. refresh by ID for reconcile, retry revalidation, and worker continuation checks

The final provider-neutral interface in `internal/tracker` should be:

```go
type TrackerReader interface {
	ListCandidates(context.Context) ([]domain.WorkItem, error)
	ListByStates(context.Context, []string) ([]domain.WorkItem, error)
	RefreshByIDs(context.Context, []string) ([]domain.WorkItem, error)
}
```

Boundary rules:

- every method takes `context.Context`
- every method returns normalized `domain.WorkItem` values
- the core must not add comment creation, state mutation, provider-specific query options, or generic metadata bags
- startup cleanup and dispatch policy remain outside the tracker interface

## Contract Semantics

`T10` freezes these semantics explicitly:

- `ListCandidates` returns zero or more normalized `domain.WorkItem` values and may include blocked or unroutable items; dispatch policy remains orchestrator-owned
- `ListByStates` trims and case-folds requested state names before comparison
- `RefreshByIDs` accepts tracker-internal item IDs, not human identifiers
- `RefreshByIDs` returns visible matches in the same order as the requested IDs
- missing IDs are omitted rather than treated as an error
- empty `states` or `ids` input returns an empty slice with `nil` error
- returned items preserve the runtime-relevant `WorkItem` fields already frozen by `T05`, including `BlockedBy`, `Routable`, prompt-facing fields, and timestamps

## Memory Adapter Design

`internal/trackers/memory` implements `tracker.TrackerReader` as a deterministic in-memory source of `domain.WorkItem` values.

The concrete reader should:

- be constructed from a seeded `[]domain.WorkItem`
- keep its own private copy of the seed data
- return deep copies on every read so callers cannot mutate adapter-internal state accidentally
- support exact ID lookup for `RefreshByIDs`
- support normalized state filtering for `ListByStates`
- return the full seeded item set from `ListCandidates`

The deep-copy rule is mandatory because `domain.WorkItem` is not flat. Each returned item must clone:

- `Labels`
- `BlockedBy`
- `Priority`
- `Routable`
- `CreatedAt`
- `UpdatedAt`

Strings may be reused safely, but slices and pointer-backed values must not alias the adapter's stored seed data.

`T10` keeps the memory adapter read-only. Do not add memory-only write or event APIs in this task. The Elixir memory adapter exposes comment/state-update notifications, but those belong to the broader compatibility boundary and must not widen the Go core read interface.

## Runtime Adoption Boundary

`T06` deliberately kept tracker access as private function seams because `T10` still owned the tracker freeze. `T10` should finish the tracker-side contract, but it should not change orchestrator code under a tracker-only verification gate.

That means:

- `internal/tracker` freezes the core read contract
- `internal/trackers/memory` becomes the first concrete implementation
- `internal/orchestrator` stays on its existing private seams in this task
- the first real runtime adoption of `TrackerReader` is deferred to a later task that already owns and verifies runtime wiring

`ListByStates` still belongs in the interface now because the Symphony spec and Elixir restart flow already prove startup terminal cleanup is part of the real tracker read contract. What is deferred is only Go runtime consumption of that method, not the interface boundary itself.

## TDD And Verification

The formal package gate remains the one recorded in the task board:

`go test ./internal/tracker/... ./internal/trackers/memory/...`

The minimum red/green set should prove:

1. `TrackerReader` contains exactly the approved three read methods and no write methods
2. the memory reader satisfies `tracker.TrackerReader` at compile time
3. `ListCandidates` returns seeded items without leaking shared mutable slices or pointer-backed values
4. `ListByStates` trims and case-folds state names, filters correctly, and returns empty on empty input
5. `RefreshByIDs` returns matches in request order, skips missing IDs, and returns empty on empty input
6. returned `domain.WorkItem` values preserve the already-frozen runtime fields later runtime logic depends on
7. mutating returned `Labels`, `BlockedBy`, `Priority`, `Routable`, `CreatedAt`, or `UpdatedAt` values does not mutate the adapter's stored seed data

Broader verification can still run later in the task, but the formal `T10` gate should not expand unless the implementation actually touches packages outside `internal/tracker` and `internal/trackers/memory`.

## Deferred To Later Tasks

- `T11` owns Linear-specific read normalization, pagination, assignee routing, and error classification
- `T07` and `T14` own real startup terminal-workspace cleanup wiring and full runtime integration
- `T12` owns provider-specific write behavior such as `linear_graphql`
- broader end-to-end memory runtime closure can grow later without widening `TrackerReader`

## Bottom Line

The correct `T10` outcome is a three-method, read-only `TrackerReader` plus a deterministic memory implementation with explicit deep-copy guarantees.

That keeps the Go core aligned with Symphony's real read contract, preserves restart-cleanup and reconcile requirements, and avoids locking provider-specific writes or unfinished runtime assembly into the wrong layer.
