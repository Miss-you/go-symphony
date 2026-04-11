# T11 Test Strategy

## Purpose

T11 has one job: prove that `internal/trackers/linear` is a real read adapter that preserves Symphony's Linear read contract without widening the Go core.

This strategy ties each verification gate to a specific contract risk:

- candidate polling must stay project-scoped, paginated, ordered, and unrouted at query time
- `ListByStates` must remain a distinct cleanup-oriented read with empty-input no-op behavior and no assignee routing
- refresh-by-ID must preserve caller order and ignore missing IDs
- `domain.WorkItem` normalization must preserve the runtime fields later packages depend on
- routing must map into `Routable` for candidate and refresh reads, including `me`
- Linear-specific failures must remain distinguishable

## What Already Proves the Frozen Core

These existing tests are the baseline contract that T11 must not break:

- [`internal/tracker/tracker_test.go`](../../internal/tracker/tracker_test.go) freezes the read-only `TrackerReader` interface shape.
- [`internal/trackers/memory/reader_test.go`](../../internal/trackers/memory/reader_test.go) proves read semantics that the core already depends on: deep copies, state normalization, request-order refresh, and empty-input behavior.

T11 should keep those tests green while adding reader-specific coverage for the Linear adapter.

## T11 Package-Scoped Unit Tests

The main proof lives in `go test ./internal/trackers/linear/...`.

That package test suite should prove:

1. `Reader` satisfies `tracker.TrackerReader` at compile time.
2. Candidate reads page through Linear with `project.slugId`, configured active states, and stable page ordering.
3. Candidate reads return all visible items and set routing metadata instead of filtering unroutable items out.
4. `ListByStates` is project-scoped, returns an empty slice for empty normalized input, and does not resolve or apply assignee routing.
5. `RefreshByIDs` batches at 50 IDs, preserves caller-visible order, and drops missing IDs without error.
6. Normalization populates `domain.WorkItem` fields correctly, including labels, blockers, timestamps, priority, and `AssigneeID`.
7. Routing maps into `Routable` correctly for no assignee, blank assignee, exact match, mismatch, and `me`.
8. Missing cursor, missing credentials, GraphQL payload errors, transport/status failures, malformed payloads, and missing viewer identity remain distinct error classes.
9. Context cancellation and deadlines propagate through the client layer as context-derived failures.

Suggested shape for those tests:

- a fake GraphQL client that records queries and variables
- table-driven normalization fixtures
- routing tests that cover both candidate and refresh reads
- explicit empty-input tests for `ListByStates` and `RefreshByIDs`
- a pagination test that simulates more than one page and a broken `endCursor`

## What The Package Tests Prove

`go test ./internal/trackers/linear/...` is the gate that proves the reader is implementable and parity-safe.

It should demonstrate:

- the adapter compiles against the frozen `TrackerReader`
- query construction matches the required Linear contract
- `ListByStates` cannot accidentally inherit assignee-routing behavior
- `Routable` is not a soft hint, but a contract carried by candidate and refresh reads
- error handling is stable enough that later runtime code can distinguish misconfiguration from transient Linear failures

## Broader Repo Verification

After the package gate passes, run broader verification:

- `go test ./...`
- `make build`
- `make lint`

Those gates prove T11 did not regress the frozen core tracker contract, the memory adapter, or unrelated packages that already depend on `domain.WorkItem` and `TrackerReader`.

The wider suite is important even though the implementation is localized, because Linear normalization touches shared runtime fields:

- `domain.WorkItem.BlockedBy`
- `domain.WorkItem.Labels`
- `domain.WorkItem.Priority`
- `domain.WorkItem.Routable`
- `domain.WorkItem.CreatedAt`
- `domain.WorkItem.UpdatedAt`

## E2E Applicability

`make test-e2e` is not a primary proof for T11.

Reason:

- T11 does not wire the reader into orchestrator/runtime paths yet.
- The task is package-scoped adapter work, not end-to-end run integration.

If the repository's current state allows e2e to run cheaply, it can still be executed as a broad confidence gate, but it does not prove T11-specific behavior. The meaningful T11 evidence remains the package tests plus full repo compile/test/lint.

## Residual Risk Handling

If package tests or broader verification reveal a gap, the follow-up should stay scoped:

- fix the failing reader behavior or test
- re-run the affected package gate
- re-run the broader verification gates

Any leftover non-blocking note should be recorded in `workspace/T11/todo.md` before the task advances beyond verification.
