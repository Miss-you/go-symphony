# T10 New Implementation

## Scope

T10 should freeze the first provider-neutral tracker read contract in Go and provide an in-memory tracker path that can support local closure and tests without Linear.

This task is intentionally narrower than the current Elixir boundary. The approved Go design says the core package `tracker` defines `TrackerReader`, while provider-specific read normalization stays in `internal/trackers/linear` and provider writes stay out of the core. The task board only requires the core tracker surface to be read-only and a memory-backed path that can drive integration tests without Linear.

## Current Go Evidence

The Go side is not yet implementing tracker behavior, but the surrounding shape is already visible:

- `internal/tracker/doc.go` is only a placeholder package comment today, so T10 is the first place where the read contract can be made real.
- `internal/trackers/memory/doc.go` is also a placeholder, which means the memory adapter is still unshaped and can be kept minimal.
- `internal/orchestrator/service.go` already uses package-private dependency seams rather than a public tracker interface. The service stores `serviceDeps` with `listCandidates` and `refreshItems` callbacks, schedules polls internally, and drives the poll cycle from `handlePollCycle` through `state.reconcileStalled`, `state.reconcileRunning`, and `state.processCandidates`.
- `internal/orchestrator/state.go` already assumes a read model of `domain.WorkItem` and depends on refresh-by-id semantics for candidate revalidation.
- `internal/domain/types.go` already froze the runtime item shape as `domain.WorkItem`, which is the payload the tracker reader needs to return.
- `internal/config/settings.go` already supports `ProviderKindMemory`, so there is already a typed configuration hook for a non-Linear tracker path.

The important consequence is that T10 does not need to invent the orchestrator contract. It needs to formalize the read seam that `internal/orchestrator` is already using privately.

## Reference Implementation

The Elixir reference still exposes a broader tracker boundary than T10 should freeze in core Go:

- `SymphonyElixir.Tracker` defines callbacks for `fetch_candidate_issues`, `fetch_issues_by_states`, `fetch_issue_states_by_ids`, `create_comment`, and `update_issue_state`.
- `SymphonyElixir.Tracker.Memory` implements all of those callbacks, but its read behavior is the part that matters here.
- `SymphonyElixir.Orchestrator` uses `Tracker.fetch_candidate_issues/0` for dispatch discovery and `Tracker.fetch_issue_states_by_ids/1` for running-item reconcile and revalidation.
- Tests in `elixir/test/symphony_elixir/extensions_test.exs` confirm that the memory adapter filters configured issues, normalizes state matching case-insensitively, ignores non-issue entries, and emits test-visible events for the write-like methods.

The reference boundary is therefore broader than the Go core should be. T10 should preserve the read semantics the orchestrator needs, not copy the full Elixir adapter surface into `internal/tracker`.

## Constraints

The approved design and task board impose several hard limits:

- Core packages must stay provider-neutral.
- `tracker` must define a read interface only.
- `internal/trackers/linear` owns Linear-specific normalization later, not the core tracker package.
- No universal tracker write API should appear in V1.
- No provider-agnostic default workflow should appear in V1.
- The orchestrator remains the single owner of mutable runtime state, so tracker code should be a data source, not a second policy engine.
- T10 depends on `T05` and `T06`, which means the contract has to match the already-frozen `domain.WorkItem` and the orchestrator’s private `listCandidates` / `refreshItems` seams.

The task board also constrains success to two concrete outcomes: the core tracker surface is read-only, and the memory path can support tests without Linear.

## Proposed Go-Native Shape

The strongest candidate for `internal/tracker` is a small interface centered on the two reads the orchestrator already uses:

- list dispatch candidates
- refresh items by ID for reconcile and revalidation

That shape fits the current `internal/orchestrator` seams without adding future obligations. It is also close enough to the Elixir behavior to preserve the real contract, while avoiding the broader Elixir write callbacks that do not belong in the Go core.

The memory adapter should probably be a simple in-memory reader over a seeded slice or map of `domain.WorkItem` values, with deterministic filtering and normalization behavior. Based on the Elixir adapter, the most important semantics to preserve are:

- return only configured `domain.WorkItem` values
- ignore non-item or malformed entries
- normalize state comparisons case-insensitively when filtering
- support ID-based refresh for reconcile and dispatch revalidation

One open question is whether the memory adapter should also expose test-only event hooks similar to Elixir’s comment/state-update messages. That behavior exists in the reference implementation, but it is not part of the core read interface and should not force a write API into `TrackerReader`. If it is kept at all, it should stay as a local test helper on the memory adapter, not as a core contract.

## Open Questions

- Should `TrackerReader` expose exactly two methods, or should it include a third helper for state-based filtering because the reference Elixir boundary has `fetch_issues_by_states/1`?
- Should the memory adapter own its own seeded data store, or should tests inject a reader fixture directly?
- Should refresh return missing IDs as an empty result, or should the reader surface partial misses explicitly?
- Should the memory adapter preserve Elixir-style state normalization rules exactly, including trimming and case folding?
- Should write-like test hooks exist only on the memory adapter, or should T10 omit them entirely and let later tests stub behavior another way?

My read of the current repository evidence is that the first two questions are the real ones. The third and fourth are about parity fidelity, and the fifth is mostly an overdesign trap.

## Implementation Risks

- If the tracker interface is broadened now, T11 will inherit a core abstraction that is already contaminated by provider-specific behavior.
- If the memory adapter is too narrow, the orchestrator tests later will need fake tracker plumbing instead of a real local/test data source.
- If filtering and refresh semantics are underspecified, reconcile behavior will drift between the Go core and the Elixir reference.
- If the memory adapter returns shared mutable slices, tests can become order-sensitive once orchestrator code starts reconciling and sorting the returned items.
- If T10 tries to preserve the whole Elixir tracker surface, it will violate the approved boundary that keeps writes outside the core.

## Bottom Line

T10 should formalize the read seam the Go orchestrator is already using privately, with a memory adapter that behaves like a deterministic local fixture source. The contract should be small, read-only, and compatible with the frozen `domain.WorkItem` model, while leaving write behavior and Linear-specific normalization to later compatibility-shell tasks.
