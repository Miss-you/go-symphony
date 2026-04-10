## Context

`T04` established a provider-neutral typed config contract, but the runtime still has no concrete provider-neutral domain vocabulary in Go. `internal/domain` is only a placeholder package, while the approved design already expects downstream tasks to share stable definitions for `WorkItem`, `Blocker`, `RunEvent`, `Snapshot`, `RetryEntry`, and `PollingState`.

The Elixir reference implementation currently spreads those concepts across a Linear-normalized issue struct, orchestrator-private maps, and snapshot maps built for observability. `T05` must preserve the runtime semantics that later tasks depend on, while removing provider-specific naming and avoiding an oversized “future-proof” core model.

## Goals / Non-Goals

**Goals:**

- freeze a provider-neutral runtime vocabulary in `internal/domain`
- keep `WorkItem` large enough for current orchestration and prompt/template compatibility, but no larger
- define snapshot and event semantics with concrete helper types only where the landed implementation and contract tests prove they are needed
- define a stable worker-to-orchestrator `RunEvent` contract
- use Go-native timing types in the core model
- add package-scoped tests that lock the exported domain contract for later tasks

**Non-Goals:**

- implementing orchestrator state machines, retry scheduling, or reconcile logic
- implementing tracker normalization, Linear adapter behavior, or toolbridge writes
- creating HTTP, dashboard, or web DTOs in `internal/domain`
- storing process refs, timers, or goroutine wiring in exported core types
- introducing generic metadata escape hatches or broader provider abstractions

## Decisions

### 1. Freeze a small exported type set in `internal/domain`

`internal/domain` will export only the types later runtime packages demonstrably need:

- `WorkItem`
- `Blocker`
- `ActiveRun`
- `RetryEntry`
- `PollingState`
- `Snapshot`
- `RunEvent`
- `RunEventKind`
- `CodexTotals`
- `RateLimits`
- `RateLimitBucket`
- `RateLimitCredits`

Alternative considered:

- export even broader helper families or generic metadata bags up front for every possible runtime subshape. Rejected because that would cross into speculative future-proofing. The landed helper types are the minimum concrete shapes the implementation and tests already proved necessary.

### 2. Keep `WorkItem` prompt-capable but provider-neutral

`WorkItem` will keep the fields the current Symphony runtime already depends on for orchestration or prompt rendering:

- identity: `ID`, `Identifier`
- prompt/session labeling: `Title`, `Description`
- dispatch logic: `State`, `Priority`, `BlockedBy`, `Routable`
- compatibility prompt fields: `BranchName`, `URL`, `AssigneeID`, `Labels`, `CreatedAt`, `UpdatedAt`

It will explicitly exclude project config, provider payload fragments, workpad/comment state, and generic metadata maps.

Alternative considered:

- keep `WorkItem` extremely narrow and move prompt-facing fields into later workflow packages. Rejected because current Symphony prompt rendering consumes the normalized issue directly, so dropping those fields here would force later tasks either to rehydrate provider payloads or to widen the domain model again.

### 3. Keep routing eligibility explicit, narrow, and tri-state

`WorkItem.Routable` is the provider-neutral bridge for the current adapter-computed “assigned to this worker” gate.

Its semantics are intentionally narrow:

- adapters compute it from provider-specific routing rules
- the orchestrator still decides dispatch from active state, terminal state, blockers, and concurrency
- this field is not a second routing policy engine
- `nil` means “not explicitly denied by the adapter”
- `false` means “do not dispatch on this worker”
- `true` means “explicitly routable”

Alternative considered:

- move worker-routing eligibility entirely out of `WorkItem`. Rejected because the current runtime already depends on a normalized eligibility gate, and forcing later tasks to recompute that from provider payloads would reintroduce adapter leakage into core scheduling code.

### 4. Separate schedulable items from runtime projection state

The domain package will not export a giant mutable “runtime state” struct. Instead it will separate:

- `WorkItem`: schedulable item facts
- `ActiveRun`: running-item projection
- `RetryEntry`: retry-queue projection
- `PollingState`: poll-loop projection
- `Snapshot`: composed projection for later observability layers
- `CodexTotals` plus `RateLimits` helper types for the aggregate observability data already exposed by the runtime

Alternative considered:

- mirror the orchestrator’s eventual internal maps directly in `internal/domain`. Rejected because the design explicitly says the orchestrator is the sole mutable state owner and observability is projection-only.

### 5. Keep time Go-native in the core model

Use `time.Time` and `time.Duration` in `RetryEntry` and `PollingState`:

- `RetryEntry.DueAt time.Time`
- `PollingState.NextPollAt *time.Time`
- `PollingState.Interval time.Duration`

Compatibility shells can derive `due_in_ms`, ISO timestamps, or countdown text later.

Alternative considered:

- store milliseconds in the core model for parity with current Elixir snapshot payloads. Rejected because that leaks transport/presentation concerns into the core and makes the Go model less natural to use.

### 6. Freeze the concrete snapshot helper types that later packages already need

The current runtime already exposes aggregate token totals and structured rate-limit information, and the landed implementation represents those facts with concrete exported types:

- `ActiveRun`
- `CodexTotals`
- `RateLimits`
- `RateLimitBucket`
- `RateLimitCredits`

These stay in scope because the implementation and reflection-based contract tests already freeze them as part of the public core-domain surface.

Alternative considered:

- leave Codex totals and rate-limit data as untyped `map[string]any` forever. Rejected because later runtime and observability tasks need a stable semantic contract rather than ad hoc maps.

### 7. Keep `RunEvent` tagged and intentionally small

`RunEvent` will be a tagged worker-reporting envelope with:

- item identity
- timestamp
- attempt/workspace/host/session context
- optional message/error context
- optional aggregate Codex totals
- optional aggregate rate-limit data

`RunEventKind` will freeze the initial event vocabulary named in the approved design:

- `workspace_created`
- `workspace_path_discovered`
- `runner_host_selected`
- `codex_event_received`
- `turn_completed`
- `run_completed`
- `run_failed`
- `retry_scheduled`

Alternative considered:

- model one bespoke struct per event kind. Rejected because it adds ceremony without evidence that the event surface is stable enough yet to justify many event-specific types.

## Risks / Trade-offs

- `[Prompt-capable WorkItem remains too broad]` → Keep the field list explicit and block raw provider metadata bags or provider config from entering the type.
- `[Later tasks may try to widen `internal/domain` with extra helper types]` → Lock the exported contract with reflection-based package tests and keep additions justified by direct downstream need.
- `[Routable may get misread as a full dispatch policy]` → Make its adapter-only and tri-state semantics explicit in docs and tests.
- `[Typed rate-limit details may drift from future Codex protocol details]` → Freeze semantic presence now, and extend concrete helper shapes deliberately only when later protocol work proves new stable fields are required.

## Migration Plan

No user-facing migration is required. The rollout is internal:

1. add the exported domain types and doc comments under `internal/domain`
2. add package tests that freeze the exported shape and boundary rules
3. update later tasks to compile against `internal/domain` rather than placeholder packages or ad hoc structs

Rollback is straightforward: revert the `T05` changes and return `internal/domain` to a placeholder package, at the cost of blocking downstream work.

## Open Questions

- None for `T05`. The current design is specific enough to implement without opening a broader architecture loop.
