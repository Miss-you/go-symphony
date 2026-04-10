# T10 Final Implementation v1

## Task Goal

`T10` freezes the provider-neutral, read-only tracker contract for the Go core and lands a memory-backed adapter that can drive local and test flows without Linear.

This task is the point where `internal/tracker` stops being a placeholder and becomes the durable read boundary that later `linear`, `workflow`, and end-to-end wiring work must depend on.

## Scope

`T10` should land all of the following:

- the first real `internal/tracker.TrackerReader` interface
- a memory-backed implementation in `internal/trackers/memory`
- package-scoped tests that freeze the read contract and memory semantics

`T10` must not land any of the following:

- tracker write methods in the core
- Linear GraphQL queries, pagination, or error mapping
- workflow selection or toolbridge behavior
- a provider-agnostic default workflow
- edits to `internal/orchestrator` or other runtime packages whose formal gate is not owned by `T10`
- exported orchestrator integration APIs for the unfinished `workspace`, `runner`, `codex`, or full runtime assembly tasks

## Review Framing

This first draft is intentionally compatibility-first and narrow:

- preserve the real Symphony read surface the runtime already depends on
- keep the Go core on `WorkItem` terms, not `issue` or `linear` terms
- avoid freezing more than `T10` actually owns
- leave provider-specific normalization details to `T11`

## Core Design Decision

The core tracker boundary should freeze exactly three read operations, because Symphony's language-agnostic contract and the Elixir runtime both require all three:

1. candidate listing for dispatch polling
2. state-based listing for startup terminal-workspace cleanup
3. refresh by ID for reconcile, retry revalidation, and worker continuation checks

That leads to one small provider-neutral interface in `internal/tracker`:

```go
type TrackerReader interface {
	ListCandidates(context.Context) ([]domain.WorkItem, error)
	ListByStates(context.Context, []string) ([]domain.WorkItem, error)
	RefreshByIDs(context.Context, []string) ([]domain.WorkItem, error)
}
```

Use `context.Context` on every method so later network-backed readers can honor timeouts and cancellation without revisiting the interface.

Use `domain.WorkItem` everywhere so the core never depends on provider-specific payload types.

Do not add:

- comment creation
- issue state mutation
- generic metadata bags
- provider-specific query option structs
- startup cleanup or dispatch policy on the tracker itself

Those are either compatibility-shell concerns or orchestrator concerns, not tracker-core concerns.

## Why All Three Methods Belong

### `ListCandidates`

This is the poll-loop entrypoint already implied by `internal/orchestrator/service.go` and `state.go`.

It should return the normalized active-state work items visible to the current provider configuration. It does not need to pre-apply orchestrator policy such as blocker gating, claim dedupe, or concurrency.

### `ListByStates`

This belongs in the interface even though the current Go orchestrator does not call it yet.

Reason:

- the Symphony spec explicitly requires `fetch_issues_by_states(state_names)` for startup terminal cleanup
- the Elixir orchestrator calls it during restart recovery to find terminal issues whose workspaces should be removed
- omitting it in `T10` would freeze a core contract that is already known to be incomplete

`ListByStates` is still provider-neutral because it expresses a generic read over normalized state names, not any Linear-specific query concept.

### `RefreshByIDs`

This is required by both the current Go scheduler logic and the reference implementation:

- dispatch revalidation
- running-item reconcile
- retry revalidation
- worker-side continuation checks

It is therefore part of the core runtime contract, not an adapter convenience.

## Contract Semantics

`T10` should freeze these semantics explicitly:

- `ListCandidates` returns zero or more normalized `domain.WorkItem` values and may return blocked or unroutable items; the orchestrator still owns dispatch policy.
- `ListByStates` compares states case-insensitively after trimming surrounding whitespace.
- `RefreshByIDs` accepts tracker-internal item IDs, not human identifiers.
- `RefreshByIDs` returns visible matches in the same order as the requested IDs.
- missing IDs are omitted from the returned slice rather than treated as an error.
- empty `states` or empty `ids` input returns an empty slice and `nil` error.
- readers return normalized `domain.WorkItem` values, including `BlockedBy`, `Routable`, and prompt-facing fields already frozen by `T05`.

These semantics are the narrowest set that still preserves the current Symphony behavior.

## Memory Adapter Scope

`internal/trackers/memory` should implement `tracker.TrackerReader` as a deterministic in-memory source of `domain.WorkItem` values.

The concrete memory reader should:

- be constructed from a seeded `[]domain.WorkItem`
- keep its own private copy of the seed data
- return deep copies from each read method so callers cannot mutate internal state accidentally
- support exact ID lookup for `RefreshByIDs`
- support normalized state filtering for `ListByStates`
- return the full seeded item set from `ListCandidates`

`T10` should keep the memory adapter read-only.

The deep-copy rule needs to be explicit because `domain.WorkItem` is not a flat value. Each returned item must clone:

- `Labels`
- `BlockedBy`
- `Priority`
- `Routable`
- `CreatedAt`
- `UpdatedAt`

Strings can be reused safely, but slices and pointer-backed values must not alias the adapter's stored seed data.

Do not add memory-only write or event APIs in this task. The Elixir memory adapter emits comment/state-update notifications, but those are part of a broader compatibility boundary that the Go core is intentionally not copying. If later tests need adapter-local mutation helpers, they can be introduced as concrete memory-package helpers without widening `TrackerReader`.

## Runtime Adoption Boundary

`T06` deliberately kept tracker access as private function seams because `T10` still owned the tracker freeze. `T10` should finish the tracker-side contract, but it should not change orchestrator code under a tracker-only verification gate.

That means the correct boundary for this task is:

- freeze `tracker.TrackerReader` in `internal/tracker`
- land `internal/trackers/memory` as the first concrete implementation of that contract
- leave `internal/orchestrator` on its existing package-private seams for now
- defer the actual swap from ad hoc callbacks to `tracker.TrackerReader` until a later task that already owns and verifies runtime wiring

`ListByStates` still belongs in the interface now, because the Symphony spec and Elixir restart flow already prove startup terminal cleanup is part of the real tracker read contract. The adoption of that method by Go runtime code is simply deferred to the task that owns workspace cleanup and end-to-end runtime assembly.

This keeps the interface stable without forcing `T10` to modify packages outside its formal gate.

## Boundary Rules

- `internal/tracker` owns only the provider-neutral read contract.
- `internal/trackers/memory` and later `internal/trackers/linear` own concrete read implementations.
- `internal/orchestrator` owns polling, reconcile, retry, claim state, and dispatch policy.
- `internal/workspace` owns cleanup behavior once `T07` lands; `ListByStates` exists now so that later cleanup wiring does not force another tracker-interface change.
- provider-specific writes remain outside the core and must not reappear through `TrackerReader`.

## TDD Direction

`T10` should be implemented test-first with the package gate already recorded in the task board:

`go test ./internal/tracker/... ./internal/trackers/memory/...`

The minimum red/green set should prove:

1. `TrackerReader` contains exactly the approved three read methods and no write methods.
2. the memory reader satisfies `tracker.TrackerReader` at compile time.
3. `ListCandidates` returns seeded items without leaking shared mutable slices or pointer-backed values.
4. `ListByStates` trims and case-folds state names, filters correctly, and returns empty on empty input.
5. `RefreshByIDs` returns matches in request order, skips missing IDs, and returns empty on empty input.
6. returned `domain.WorkItem` values preserve the already-frozen runtime fields that later orchestrator logic depends on.
7. mutating returned `Labels`, `BlockedBy`, `Priority`, `Routable`, `CreatedAt`, or `UpdatedAt` values does not mutate the adapter's stored seed data.

Do not broaden the formal task gate beyond the tracker and memory packages unless the implementation truly touches other packages.

## Deferred To Later Tasks

- `T11` owns Linear-specific read normalization, pagination, assignee routing, and error classification.
- `T07` and `T14` own real startup terminal-workspace cleanup wiring and full runtime integration.
- `T12` owns write behavior such as `linear_graphql` and other provider-specific mutations.
- end-to-end memory runtime closure beyond the reader contract can grow later without widening `TrackerReader`.

## Bottom Line

The right `T10` move is to freeze a three-method, read-only `TrackerReader` and back it with a deterministic memory implementation.

That keeps the Go core aligned with Symphony's real runtime contract, preserves restart-cleanup and reconcile requirements, and avoids locking provider-specific writes or speculative runtime assembly into the wrong layer.
