# T06 Test Strategy

## Goal

`T06 Orchestrator Core` must prove that `internal/orchestrator` becomes the single owner of mutable scheduling state without leaking provider-specific behavior or later-package interfaces into the core. The proof is not “a scheduler package exists”; it is that the landed service enforces the approved polling, dispatch, retry, reconcile, stall, and snapshot rules precisely enough that later runtime tasks can build on one source of truth.

## What Must Be Proven

1. Polling state is orchestrator-owned, including immediate startup polling, refresh coalescing, and stale tick protection.
2. Candidate ordering and dispatch gating match the approved runtime rules for priority, age, blockers, routing eligibility, claimed/running state, and concurrency limits.
3. Retry behavior distinguishes continuation from failure while freezing claim retention, attempt lineage, capped backoff, and stale retry-delivery handling.
4. Reconcile refreshes still-valid running items in place and removes terminal, missing, non-active, unroutable, and stalled runs with the correct retry or cleanup intent.
5. `domain.RunEvent` is the only worker-to-orchestrator mutation input, and aggregate token/rate-limit tracking follows the approved semantics.
6. `domain.Snapshot` projects only private orchestrator state, with deterministic running/retrying ordering and no leaked timer or handle wiring.
7. The new package compiles cleanly with the rest of the repo and passes canonical build/lint gates.

## Verification Matrix

### 1. Package-scoped orchestrator behavior tests

Command:

```bash
go test ./internal/orchestrator/...
```

Why this matters:

- This is the task-board gate for `T06`.
- `T06` is primarily a behavior-freezing task, so package tests must prove the state transitions directly rather than just compile the package.
- These tests should cover:
  - immediate startup poll and `Polling.Checking` / `NextPollAt` transitions
  - refresh coalescing and stale tick guards
  - deterministic candidate ordering
  - blocked `Todo`, `Routable=false`, claimed, running, global-cap, and per-state-cap dispatch denial
  - revalidation preventing stale dispatch
  - continuation retry at attempt `1` with short delay and retained claim
  - failure retry attempt progression and capped exponential backoff
  - capacity-blocked continuation falling into failure backoff
  - stale retry token ignoring older retry deliveries
  - reconcile outcomes for active, terminal, missing, non-active, and unroutable runs
  - stall detection based on last activity or start time
  - cumulative aggregate `CodexTotals`, latest non-nil `RateLimits`, and deterministic snapshot ordering

### 2. Repository-wide compile safety

Command:

```bash
go test ./...
```

Why this matters:

- `T06` introduces the first real runtime package that later tasks will depend on.
- A green package-level orchestrator suite is not enough if the new package collides with existing placeholders or leaves the module in a bad compile state.

### 3. Canonical build and lint gates

Commands:

```bash
make build
make lint
```

Why this matters:

- `make build` proves the repository still builds through the canonical entrypoint after `internal/orchestrator` stops being a placeholder.
- `make lint` catches structural problems in the new state/service/test code that narrow behavior tests may not expose.

### 4. E2E applicability check

Command:

```bash
make test-e2e
```

Why this matters:

- `T06` still lands before real workspace/runner/Codex/tracker integration, so e2e is not the primary proof of scheduler correctness yet.
- Running the command still verifies the repo command contract remains healthy.
- If `make test-e2e` only acts as a command-contract check rather than meaningful orchestrator behavior proof at this phase, that limitation must be recorded explicitly in `workspace/T06/todo.md` and the task board instead of being silently skipped.

## Acceptance Threshold

`T06` is ready to leave verification only if:

- `go test ./internal/orchestrator/...` proves the exact poll, dispatch, retry, reconcile, stall, aggregate counter, and snapshot rules in `workspace/T06/final_impl.md` and the `orchestrator-core` OpenSpec change
- `go test ./...`, `make build`, and `make lint` pass
- `make test-e2e` either passes or its exact current applicability limit is recorded explicitly
- the observed behavior still keeps mutable runtime state inside `internal/orchestrator`, accepts worker facts only through `domain.RunEvent`, and projects only `domain.Snapshot`
