# T14 New Implementation Notes

## Scope

This note records what is already present in the current Go repository state for T14 and what still needs glue to turn those pieces into complete runs.

The focus is narrow:

- `internal/orchestrator`
- `internal/workspace`
- `internal/runner`
- `internal/codex`
- `internal/tracker` plus `internal/trackers/memory` and `internal/trackers/linear`
- `internal/toolbridge/linear`
- `internal/workflow`
- `internal/domain` snapshots and run events
- existing T06-T13 specs and workspace artifacts

This document does not propose product code changes. It only captures the current integration surface and the remaining composition gap.

## Go Source Files Inspected

- `cmd/symphony/main.go`
- `cmd/symphony/main_test.go`
- `internal/domain/types.go`
- `internal/domain/domain_contract_test.go`
- `internal/orchestrator/service.go`
- `internal/orchestrator/state.go`
- `internal/orchestrator/service_test.go`
- `internal/workspace/manager.go`
- `internal/workspace/manager_test.go`
- `internal/runner/execution_host.go`
- `internal/runner/execution_host_test.go`
- `internal/tracker/tracker.go`
- `internal/trackers/memory/reader.go`
- `internal/trackers/memory/reader_test.go`
- `internal/trackers/linear/reader.go`
- `internal/trackers/linear/reader_test.go`
- `internal/toolbridge/linear/bridge.go`
- `internal/toolbridge/linear/bridge_test.go`
- `internal/codex/session.go`
- `internal/codex/session_test.go`
- `internal/workflow/select.go`
- `internal/workflow/compat_linear_default.go`
- `internal/workflow/types.go`
- `internal/workflow/workflow_test.go`
- `internal/config/workflow.go`
- `internal/config/settings.go`
- `internal/config/store.go`

I also checked:

- `docs/plans/2026-04-10-go-symphony-design.md`
- `docs/plans/2026-04-10-go-symphony-design-task.md`
- `openspec/specs/{runtime-orchestrator,workspace-lifecycle,runtime-runner-execution-host,runtime-tracker-reader,codex-app-server-protocol,linear-toolbridge,workflow-selection}/spec.md`
- `workspace/T06` through `workspace/T13`

## Existing Contracts

### Domain model

`internal/domain/types.go` already freezes the provider-neutral runtime vocabulary:

- `WorkItem`
- `Blocker`
- `ActiveRun`
- `RetryEntry`
- `PollingState`
- `Snapshot`
- `CodexTotals`
- `RateLimits`
- `RunEventKind`
- `RunEvent`

`internal/domain/domain_contract_test.go` enforces the field shape and the absence of provider-specific fields like `Issue`, `Linear`, `Tracker`, and `ProjectSlug`.

This is the payload language for the rest of T14. Nothing in the core should need provider-specific runtime state added to these types.

### Orchestrator

`internal/orchestrator` already owns the mutable scheduling model:

- private claimed, running, and retry maps
- poll cadence and refresh coalescing
- continuation and failure retry lineage
- reconcile and stall recovery
- aggregate token and rate-limit projection
- host-load projection for admission

The important seam is `serviceDeps`. It already shows the shape the real runtime needs:

- `listCandidates`
- `refreshItems`
- `startRun`
- `stopRun`
- `hostSelection`
- `admitRun`

`internal/orchestrator/service_test.go` proves the package already handles:

- startup poll timing
- refresh coalescing
- retry scheduling
- stale retry rejection
- dispatch ordering and gating
- runner host selection integration
- reconcile and stall recovery
- snapshot ordering and aggregate totals

What is still private is also the key constraint: there is no exported runtime API yet. The package is ready to be driven, but not yet assembled into a top-level run loop.

### Workspace

`internal/workspace/manager.go` already owns the lifecycle contract:

- deterministic path derivation
- identifier normalization
- canonical root and symlink-escape safety
- create/reuse/remove behavior
- run-phase hooks
- terminal cleanup fan-out across worker hosts
- delegation of command execution to `runner.Executor`

`internal/workspace/manager_test.go` and `internal/workspace/lifecycle_test.go` cover the safety and lifecycle matrix.

This package already gives T14 a concrete place to create, run in, and remove workspaces. The remaining gap is not the workspace code itself; it is the code that decides when to call it from a real runtime run.

### Runner

`internal/runner/execution_host.go` already defines the host-aware execution contract:

- local command execution when host is empty
- SSH command execution when host is set
- timeout normalization
- exit-status normalization
- `HostSelection.Select(...)` for stateless admission

The tests already prove:

- local commands execute in the requested directory
- SSH arguments are built compatibly
- IPv6 hosts are handled safely
- host selection respects preferred host, load, and per-host capacity

This is enough for T14 to wire real admission and execution without inventing a second host-selection model.

### Codex

`internal/codex/session.go` already provides the app-server session boundary:

- workspace validation before launch
- `initialize` then `thread/start`
- reusable turns on one session
- approval handling
- non-interactive user-input fallback
- dynamic tool dispatch through an injected `ToolHandler`
- raw string and object-shaped tool arguments
- top-level `contentItems` in tool results
- read timeout and turn timeout classification
- normalized protocol events through an event sink

This is the piece T14 needs to turn worker runtime activity into `domain.RunEvent` updates. The session itself is present; the orchestration glue around it is not.

### Tracker readers

`internal/tracker/tracker.go` already freezes the provider-neutral read-only contract:

- `ListCandidates`
- `ListByStates`
- `RefreshByIDs`

`internal/trackers/memory/reader.go` provides a deterministic local/test reader.

`internal/trackers/linear/reader.go` provides the Linear read adapter with:

- candidate listing
- state filtering
- refresh-by-id
- pagination
- assignee routing
- error classification
- normalization of Linear payloads into `domain.WorkItem`

The reader tests show the intended split very clearly: the core can read from memory for local closure, or from Linear for the provider-backed path, without widening the core tracker interface.

### Linear tool bridge

`internal/toolbridge/linear/bridge.go` already contains the compatibility-shell write surface:

- exactly one advertised dynamic tool: `linear_graphql`
- Symphony-style tool result shaping
- raw GraphQL execution
- provider-specific comment and issue-state helpers
- dependency isolation from core tracker/domain/orchestrator packages

This is already the right leaf for Linear writes. T14 does not need to redesign it; it needs to make sure the live runtime selects and injects it correctly.

### Workflow selection

`internal/workflow/select.go` and `internal/workflow/compat_linear_default.go` already select the Linear bundle explicitly:

- `config.ProviderLinear` selects `compat_linear_default`
- non-Linear providers fail explicitly
- the bundle reuses `config.EffectivePromptTemplate`
- the bundle exposes the Linear ToolBridge tool specs and handler directly

`workspace/T13/final_compare.md` confirms the workflow package intentionally stays out of `internal/orchestrator`, `internal/tracker`, `internal/workspace`, `internal/runner`, and `internal/domain`.

### Existing specs and workspace artifacts

The main specs now align with the code above:

- `runtime-orchestrator`
- `workspace-lifecycle`
- `runtime-runner-execution-host`
- `runtime-tracker-reader`
- `codex-app-server-protocol`
- `linear-toolbridge`
- `workflow-selection`

The T06-T13 workspace artifacts are useful because they explicitly defer the remaining integration point:

- `workspace/T09/final_compare.md` says `internal/codex` is not yet wired into orchestrator run workers and that belongs to T14.
- `workspace/T07/final_impl.md` and `workspace/T08/final_impl.md` both defer the full runtime assembly.
- `workspace/T10/final_impl_v1.md` defers first real runtime consumption of `TrackerReader`.
- `workspace/T12/final_compare.md` and `workspace/T13/final_compare.md` show the provider-specific write and workflow bundle pieces are already closed.

One practical repo-state note: `cmd/symphony/main.go` is still an empty `main`, and `workspace/T14/` is not present in this checkout even though the task-board log already records the claim.

## Missing Integration

The remaining gap is composition, not new low-level contracts.

What is missing today:

1. A top-level runtime assembly that loads config, selects the workflow bundle, builds the tracker reader, workspace manager, runner executor, codex session, and orchestrator service.
2. A real run loop that turns codex protocol events into `domain.RunEvent` updates and feeds them back to the orchestrator.
3. Concrete `startRun` and `stopRun` wiring that actually create workspaces, start Codex sessions, stop sessions, and clean up terminal workspaces.
4. Provider selection for end-to-end runs. The repo has both memory and Linear readers, but no shared bootstrap that chooses one and drives a full lifecycle.
5. Snapshot exposure to whatever later compatibility surface reads runtime state. `internal/domain.Snapshot` exists, but `internal/httpapi`, `internal/dashboard`, `internal/web`, and `internal/observability` are still only placeholder packages.
6. A command entrypoint in `cmd/symphony/main.go` that does more than return immediately.

In short, the primitives exist; the live glue does not.

## Minimal T14 Shape

The smallest sensible integration layer is a thin runtime composition package or a very small `main`-side bootstrap that owns wiring only.

Recommended shape:

```text
config + workflow
      |
      v
bundle selection
      |
      v
tracker reader -----> orchestrator <----- workspace manager <----- runner executor
      |                   |                     |
      |                   v                     v
      |               domain.Snapshot      codex session
      |                                         |
      +-----------------------------------------+
```

Recommended API shape:

- `NewRuntime(...)` or a similarly small constructor that accepts already-loaded `config.Settings`, `config.Workflow`, and concrete deps.
- `Start(ctx)` or `Run(ctx)` that owns the lifecycle of one process.
- A shutdown path that closes the orchestrator, Codex sessions, timers, and any open workspace/session handles.

The runtime constructor should stay boring. It should only:

- choose the workflow bundle
- select the tracker backend
- wire `orchestrator.serviceDeps`
- pass `runner.HostSelection` admission data from orchestrator-owned running state
- bind `codex.EventSink` to `domain.RunEvent`
- bind `startRun` to workspace create + codex session bootstrap + turn execution
- bind `stopRun` to session close + cleanup intent

For provider-backed runs, the shape should probably stay provider-neutral at the boundary and switch only at the adapter layer:

- `memory` for local/test closure
- `linear` for real tracker reads

That lets T14 prove the orchestration path without inventing a new core abstraction.

## Test Hooks / Fixtures

The repo already has the right fixtures to prove T14 if the glue is added carefully.

Use these existing test hooks:

- `internal/orchestrator/service_test.go`
  - fake clock
  - fake timer factory
  - direct state injection
  - retry and host-selection assertions
- `internal/workspace/manager_test.go` and `internal/workspace/lifecycle_test.go`
  - temporary directories
  - fake command transport
  - hook behavior
- `internal/runner/execution_host_test.go`
  - fake process starter
  - host-selection table tests
- `internal/codex/session_test.go`
  - scripted transport
  - injected tool handler
  - non-interactive and timeout coverage
- `internal/trackers/memory/reader_test.go`
  - deterministic local reader
  - deep-copy isolation
- `internal/trackers/linear/reader_test.go`
  - fake GraphQL client
  - pagination and routing fixtures
- `internal/toolbridge/linear/bridge_test.go`
  - fake Linear client
  - dynamic-tool payload assertions

For T14 specifically, the most valuable new tests would be integration-style tests that combine these fixtures:

1. Memory tracker + workspace + runner + codex session + orchestrator.
2. Linear tracker + workflow selection + Linear tool bridge + orchestrator admission.
3. Continuation retry and terminal cleanup through the full composition layer.

Those tests should prove:

- a candidate can move from tracker read to workspace create to Codex turn to orchestrator event handling
- completion schedules continuation retry
- failure schedules failure retry
- terminal cleanup removes the workspace when asked
- memory and Linear modes both reach the same runtime closure points

## Risks / Unknowns

- `cmd/symphony/main.go` is still a no-op, so the exact CLI/runtime surface is not yet anchored in code.
- `workspace/T14/` is missing in this checkout, so the task-board claim has not been matched with a workspace artifact here.
- `internal/httpapi`, `internal/dashboard`, `internal/web`, and `internal/observability` are still placeholders, so snapshot consumers are not yet part of the T14 proof surface.
- The best T14 API shape is an inference from the existing seams, not a spec-frozen exported interface. The code suggests a thin runtime composition layer, but there is not yet a canonical package name or constructor in the tree.
- The repo already supports both memory and Linear tracker readers, but it is still ambiguous whether T14 should expose both as runtime modes or keep one as test-only closure until a later task.

The cleanest reading of the current tree is that T14 should wire the existing pieces together without broadening any of their contracts.
