# T10 Test Strategy

## Goal

`T10 TrackerReader + Memory Adapter` must prove that the Go core now has a stable, read-only tracker contract and a deterministic in-memory implementation that later runtime tasks can safely build on. The proof is not "an interface and package exist"; it is that the frozen read surface matches the approved design, the memory reader preserves the required read semantics, and callers cannot accidentally mutate adapter-internal `domain.WorkItem` state through shared slices or pointers.

## What Must Be Proven

1. `internal/tracker` exports exactly the approved three-method `TrackerReader` read contract and does not silently widen toward tracker writes or provider-specific query APIs.
2. `internal/trackers/memory` satisfies `tracker.TrackerReader` at compile time and preserves deterministic candidate, state-filtered, and refresh-by-id reads.
3. State filtering is normalization-safe: trimmed and case-folded inputs still match seeded item states.
4. Refresh-by-id behavior is runtime-safe: it preserves visible request order, omits missing IDs, and treats empty input as a successful no-op.
5. Returned `domain.WorkItem` values preserve the runtime fields later scheduler logic depends on, including blockers, routing eligibility, and prompt-visible fields.
6. Memory-reader results are caller-isolated: mutating returned `Labels`, `BlockedBy`, `Priority`, `Routable`, `CreatedAt`, or `UpdatedAt` values does not mutate the adapter's stored seed data.
7. The wider repository still compiles and passes its canonical build/lint gates after the tracker packages stop being placeholders.

## Verification Matrix

### 1. Package-scoped tracker contract and memory behavior tests

Command:

```bash
go test ./internal/tracker/... ./internal/trackers/memory/...
```

Why this matters:

- This is the formal task-board gate for `T10`.
- `T10` is a contract-freezing task, so package tests must prove interface shape and adapter semantics directly rather than only compiling the packages.
- These tests should cover:
  - `TrackerReader` method-set locking
  - compile-time satisfaction of `tracker.TrackerReader` by the memory reader
  - candidate listing over seeded items
  - normalized `ListByStates` matching
  - request-ordered `RefreshByIDs`
  - empty-input handling for state and ID queries
  - deep-copy isolation for slice and pointer-backed `domain.WorkItem` fields

### 2. Repository-wide compile safety

Command:

```bash
go test ./...
```

Why this matters:

- `T10` replaces placeholder tracker packages with real contract code.
- A green package-level tracker suite is not enough if the new interface names, imports, or package wiring leave the broader module in a bad compile state.

### 3. Canonical build and lint gates

Commands:

```bash
make build
make lint
```

Why this matters:

- `make build` proves the repository still builds through the canonical entrypoint after `internal/tracker` becomes a real dependency surface.
- `make lint` catches structural issues in the new contract and tests that narrow package assertions may miss.

### 4. E2E applicability check

Command:

```bash
make test-e2e
```

Why this matters:

- `T10` does not adopt the new reader into the runtime yet, so e2e is not the primary proof of tracker-contract correctness in this task.
- Running the standard e2e command still verifies the repository command contract remains healthy.
- If `make test-e2e` is only acting as a command-contract check rather than meaningful `T10` behavior proof at this phase, that limitation must be recorded explicitly in `workspace/T10/todo.md` and the task board instead of being silently skipped.

## Acceptance Threshold

`T10` is ready to leave verification only if:

- `go test ./internal/tracker/... ./internal/trackers/memory/...` proves the exact read-only contract and memory-reader semantics in `workspace/T10/final_impl.md`
- `go test ./...`, `make build`, and `make lint` pass
- `make test-e2e` either passes or its exact current applicability limit is recorded explicitly
- the observed implementation still keeps runtime adoption deferred: this task freezes the tracker contract and memory adapter, but does not silently expand into `internal/orchestrator` or other unowned runtime packages
