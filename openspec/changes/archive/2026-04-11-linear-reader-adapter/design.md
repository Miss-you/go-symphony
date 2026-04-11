## Context

The Go core already freezes a provider-neutral `TrackerReader` and a `domain.WorkItem` model, but `internal/trackers/linear` is still a placeholder. T11 needs a real Linear reader adapter that preserves Symphony's current read-side behavior without moving provider-specific writes or runtime wiring into the core.

The source implementation shows three distinct read paths that matter here: candidate polling, state-based cleanup reads, and refresh-by-ID revalidation. Those paths share some GraphQL plumbing but not the same semantics, especially around routing and state cleanup.

## Goals / Non-Goals

**Goals:**
- Implement a read-only Linear adapter in `internal/trackers/linear`.
- Preserve candidate pagination, state-based reads, refresh-by-ID ordering, assignee routing, and error taxonomy.
- Normalize Linear payloads into `domain.WorkItem` without widening the core tracker contract.
- Keep the implementation testable with fakes and package-scoped tests.

**Non-Goals:**
- Linear write behavior, including comment creation and issue state mutation.
- `linear_graphql` toolbridge behavior.
- Orchestrator, workflow, or HTTP/dashboard wiring.
- New provider-neutral abstractions beyond the frozen `TrackerReader`.

## Decisions

1. Use a thin `Reader` plus `Client` split.
   - `Reader` owns contract semantics and normalization.
   - `Client` only performs GraphQL execution and returns decoded payloads or transport/status failures.
   - Alternative considered: a single package-level function set. Rejected because the reader needs a stable unit boundary for tests and for routing decisions.

2. Treat candidate, state-based, and refresh-by-ID reads as separate behaviors.
   - Candidate reads are project-scoped and active-state-scoped.
   - State-based reads are cleanup-oriented and must not reuse routing semantics.
   - Refresh-by-ID reads preserve caller order and skip missing IDs.
   - Alternative considered: one generic read helper with flags. Rejected because it blurs the parity-sensitive contract differences and makes tests less precise.

3. Map Linear assignee semantics into `domain.WorkItem.Routable`.
   - `Routable` is the adapter signal for later dispatch logic.
   - `ListByStates` leaves routing unset because cleanup reads do not use assignee gating.
   - Alternative considered: adding a provider-specific routing field. Rejected because the core already has the needed runtime field.

4. Keep error classification explicit.
   - Missing token, missing project slug, missing viewer identity, transport failure, non-200 status, GraphQL errors, malformed payloads, and missing cursors each remain distinct.
   - Alternative considered: collapse everything into a generic Linear error. Rejected because it hides parity regressions and makes debugging harder.

## Risks / Trade-offs

- [Risk] Assignee routing might be treated like a query filter. → Mitigation: keep routing in normalization and tests, not in candidate selection.
- [Risk] State-based reads could accidentally inherit candidate routing semantics. → Mitigation: make `ListByStates` a separate contract path with empty-input no-op and no assignee routing.
- [Risk] Refresh-by-ID ordering could regress if the Linear response order is used directly. → Mitigation: reorder by requested IDs in the adapter and test missing-ID handling.
- [Risk] Error taxonomy could collapse during client implementation. → Mitigation: keep sentinel or typed errors narrow and test each bucket separately.

## Migration Plan

1. Land the adapter package and tests behind the existing `TrackerReader` contract.
2. Verify the package-scoped gate and then broader repo tests.
3. Leave runtime wiring to later tasks that already own orchestrator integration.

## Open Questions

- None blocking. The contract is narrow enough to implement without introducing new core abstractions.
