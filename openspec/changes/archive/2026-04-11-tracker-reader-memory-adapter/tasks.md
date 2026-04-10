## 1. Tracker Contract

- [x] 1.1 Add the provider-neutral `TrackerReader` interface under `internal/tracker` with exactly the approved read-only method set and package docs that keep writes out of the core boundary.
- [x] 1.2 Add package-scoped contract tests that lock the exported tracker interface shape and prevent silent widening toward tracker writes or provider-specific query APIs.

## 2. Memory Reader Implementation

- [x] 2.1 Implement `internal/trackers/memory` as a deterministic in-memory `TrackerReader` over seeded `domain.WorkItem` values.
- [x] 2.2 Implement and test normalized state filtering, request-ordered refresh-by-id behavior, empty-input handling, and deep-copy isolation for slices and pointer-backed `WorkItem` fields.

## 3. Verification And Task Sync

- [x] 3.1 Run `go test ./internal/tracker/... ./internal/trackers/memory/...` and fix the tracker packages until the contract suite passes.
- [x] 3.2 Run broader verification (`go test ./...`, `make build`, `make lint`, and `make test-e2e` or an explicit applicability note) and record the evidence in `workspace/T10/` and the task board.
