# T05 Test Strategy

## Goal

`T05 Domain Model` must prove that `internal/domain` freezes a provider-neutral runtime contract that later orchestration, tracker, Codex, and observability tasks can safely compile against. The proof is not “some structs exist”; it is that the exported vocabulary captures the required runtime facts without leaking provider-specific names or orchestrator-private runtime wiring.

## What Must Be Proven

1. `WorkItem` keeps the approved prompt-visible and orchestration fields that current Symphony already exposes, while excluding provider config, tracker writes, and generic metadata escape hatches.
2. `Blocker`, `ActiveRun`, `RetryEntry`, `PollingState`, `Snapshot`, `CodexTotals`, and the rate-limit structs are explicit core-domain types rather than ad hoc maps hidden in later packages.
3. `RetryEntry` and `PollingState` use Go-native timing types so the core contract stays transport-agnostic.
4. `Snapshot` is complete enough for later API/dashboard/web projection work without exposing process refs, timer refs, or other orchestrator-private state.
5. `RunEvent` and `RunEventKind` freeze the worker-reporting vocabulary named in the approved design without becoming a generic dump of worker-local details.
6. The exported `internal/domain` API surface resists drift back toward `Issue`, `Linear`, `tracker`, GraphQL, or generic metadata-bag terminology.

## Verification Matrix

### 1. Package-scoped domain contract tests

Command:

```bash
go test ./internal/domain/...
```

Why this matters:

- This is the task-board gate for `T05`.
- `T05` is a contract-freezing task, so package tests must assert the API shape directly rather than only compile the package.
- Reflection-based assertions are appropriate here because the main risk is contract drift, not algorithmic behavior.
- These tests should assert the approved exported type names, field names, field types, and `RunEventKind` values.
- These tests should explicitly lock the pointer-backed `WorkItem.Routable` contract so the “not explicitly denied” routing state is preserved.
- These tests should assert the concrete helper shapes for `ActiveRun`, `CodexTotals`, `RateLimits`, `RateLimitBucket`, and `RateLimitCredits`, because those exported helpers are now part of the landed contract.
- These tests should fail if provider-specific names (`Issue`, `Linear`, `tracker`, GraphQL fragments) or orchestrator-private wiring appear in the exported core surface.
- If later tasks introduce any additional exported helper types beyond this set, those tests must be extended in the same change instead of allowing silent surface growth.

### 2. Repository-wide compile safety

Command:

```bash
go test ./...
```

Why this matters:

- `T05` adds the first real types under `internal/domain`, which later packages will import.
- A green package-level domain test is not enough if the new exported names collide with existing placeholders or leave the broader module in a bad compile state.

### 3. Canonical build and lint checks

Commands:

```bash
make build
make lint
```

Why this matters:

- `make build` proves the repository still builds through the canonical entrypoint after `internal/domain` stops being a placeholder package.
- `make lint` catches structural issues in exported core types and tests that narrower contract assertions may miss.

### 4. E2E applicability check

Command:

```bash
make test-e2e
```

Why this matters:

- `T05` does not yet expose a full end-to-end runtime path, so e2e is not the primary proof of correctness for this task.
- Running the standard e2e command still verifies the repository command contract remains healthy.
- If it is only acting as a command-contract check rather than meaningful behavior proof for `T05`, that limitation must be recorded explicitly in `workspace/T05/todo.md` and the task board instead of being silently skipped.

## Acceptance Threshold

`T05` is ready to leave verification only if:

- `go test ./internal/domain/...` proves the exported contract shape and boundary rules
- `go test ./...`, `make build`, and `make lint` pass, or any non-applicable verification is recorded explicitly
- the observed domain contract matches `workspace/T05/final_impl.md` and the `domain-model` OpenSpec change for provider-neutral work items, runtime projections, and worker event reporting
