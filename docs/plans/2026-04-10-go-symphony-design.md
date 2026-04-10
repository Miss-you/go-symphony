# Go Symphony Design

Date: 2026-04-10
Status: Approved design draft

## Goal

Build a Go implementation of Symphony that preserves user-facing capability parity with the current Elixir implementation, while allowing the internal architecture to become more Go-native and easier to maintain.

This first design targets full parity for:

- `WORKFLOW.md` loading semantics
- default unattended Linear workflow behavior
- HTTP API
- terminal dashboard
- web dashboard at `/`
- CLI/bootstrap behavior
- Linear-backed orchestration
- Codex app-server integration

This design does not try to invent a universal tracker workflow on day one. It intentionally keeps Linear-specific workflow and tool behavior in the compatibility shell, while keeping the runtime core provider-neutral where Symphony already has real commonality.

## Design Summary

The Go version is split into two layers:

1. Domain core
   - Provider-neutral runtime pieces that Symphony truly needs regardless of tracker.
   - Examples: orchestrator, runtime state, `WorkItem`, workspace lifecycle, runner boundary, Codex app-server protocol handling, tracker read interface, snapshot generation.

2. Compatibility shell
   - User-facing parity surfaces and provider-specific behavior.
   - Examples: Linear reader adapter, `linear_graphql`, Linear default workflow bundle, HTTP API DTOs, terminal dashboard rendering, web dashboard, CLI semantics.

The system remains intentionally narrow:

- One authoritative runtime state owner: the orchestrator.
- No second observability state machine.
- No universal tracker write interface.
- No universal workpad abstraction.
- No fake provider-agnostic default workflow.

## Scope and Non-Goals

### In Scope

- User-facing parity with current Symphony
- Local and SSH worker execution
- Linear-backed orchestration
- Dynamic tool injection for Linear compatibility
- `WORKFLOW.md` hot reload with last-known-good semantics
- Snapshot-based HTTP, terminal, and web observability
- CLI parity, including acknowledgement and shutdown behavior

### Explicit Non-Goals for V1

- A generic tracker write API shared by all providers
- A generic workpad or comment persistence layer
- A universal default workflow shared by Linear and future providers
- Lark task support in the first implementation phase
- Deep package fragmentation before behavior stabilizes

## Guiding Principles

1. Preserve user-facing behavior first.
2. Keep the core narrow.
3. Put provider-specific workflow and writing behavior outside the core.
4. Treat Codex app-server as a protocol compatibility target, not a thin process wrapper.
5. Keep the orchestrator as the single runtime source of truth.
6. Add extension points only where current Symphony already proves they are real.

## Symphony to Go Terminology Mapping

This table is mandatory reference material for future work. Internal naming may change, but mappings must stay explicit.

| Symphony / Elixir term | Go internal term | Notes |
| --- | --- | --- |
| `issue` | `WorkItem` | External compatibility surfaces may still say `issue`. |
| `SymphonyElixir.Linear.Issue` | `domain.WorkItem` | Provider-neutral runtime model. |
| `tracker.kind` | `provider` internally | Input remains backward-compatible. |
| running issue | active run / active item | Runtime state, not tracker domain. |
| retry issue | retry entry | Separate runtime concept. |
| Linear workpad comment | provider workflow state | Not part of core domain. |
| `linear_graphql` | Linear ToolBridge capability | Compatibility shell only. |
| issue state | item state | Core uses provider-neutral language. |
| issue identifier | item identifier | Preserved in compatibility DTOs. |
| project slug | provider-specific config | Core must not depend on it. |

## Architecture

### Core

- `config`
  - Parses external config and normalizes it into an internal provider-neutral configuration model.
  - Supports `WORKFLOW.md` parsing, prompt/template loading, hot reload, and last-known-good retention.

- `domain`
  - Defines `WorkItem`, `Blocker`, `RunEvent`, `Snapshot`, `RetryEntry`, and related runtime structures.
  - Must not contain provider-specific write semantics.

- `orchestrator`
  - Owns polling, claiming, dispatching, retry, reconcile, stall recovery, and snapshot generation.
  - Is the single owner of mutable runtime state.

- `tracker`
  - Defines `TrackerReader`.
  - Core only depends on read operations required by the Symphony spec.

- `workspace`
  - Owns workspace path rules, safe identifiers, hooks, cleanup, and terminal-state cleanup semantics.

- `runner`
  - Owns local and SSH execution behavior, host selection, host capacity limits, remote process launch, and execution cleanup.
  - Keeps SSH-specific behavior out of `workspace`.

- `codex`
  - Owns the Codex app-server protocol.
  - Handles process lifecycle, session bootstrap, `thread/start`, `turn/start`, dynamic tool calls, approval flow, timeouts, and event normalization.

### Compatibility Shell

- `trackers/linear`
  - Linear reader adapter
  - Linear-specific normalization
  - Assignee routing behavior
  - Error classification and pagination behavior

- `workflow`
  - Workflow selector
  - `compat_linear_default` workflow bundle
  - Prompt rendering and workflow text compatible with current Symphony behavior

- `toolbridge/linear`
  - `linear_graphql`
  - Linear-specific tracker write behavior needed by the workflow
  - Workpad/comment semantics for the Linear path

- `observability`
  - Snapshot projection layer
  - Bounded recent-event buffer
  - Shared presenter model for API, terminal dashboard, and web dashboard
  - Not a second runtime state machine

- `httpapi`
  - `/api/v1/state`
  - `/api/v1/refresh`
  - `/api/v1/:issue_identifier`
  - DTOs, error codes, and refresh semantics frozen as compatibility contracts

- `dashboard`
  - Terminal snapshot renderer
  - ANSI formatting
  - Throughput, retry queue, rate-limit display, event humanization
  - Snapshot fixtures

- `web`
  - `/` dashboard
  - Static assets
  - Web observability compatibility

- `cli`
  - Startup behavior
  - Guardrails acknowledgement
  - `--logs-root`
  - `--port`
  - Offline shutdown rendering

## Core Runtime Model

### State Ownership

The orchestrator is the only owner of mutable scheduling state:

- running items
- claimed items
- retry entries
- completion bookkeeping
- poll countdown and poll-in-progress state
- aggregate Codex runtime counters needed for snapshots

Workers do not mutate shared state directly. They emit `RunEvent` messages back to the orchestrator.

### Worker Communication

Each run worker reports through a buffered event channel using a stable `RunEvent` model. Expected event categories include:

- workspace created
- workspace path discovered
- runner host selected
- Codex app-server event received
- turn completed
- run completed
- run failed
- retry scheduled

This keeps the runtime observable without duplicating business state.

### Snapshot Model

The orchestrator emits a snapshot that becomes the source for:

- HTTP API payloads
- terminal dashboard rendering
- web dashboard presentation

`observability` is projection-only. It may keep a bounded recent-event buffer, but it must not infer or own the business truth of run state.

## Key Anti-Overdesign Decisions

These constraints are intentional and should not be relaxed without evidence.

1. No universal tracker write interface in the core.
   - Core only keeps `TrackerReader`.
   - Writing belongs in provider-specific `ToolBridge` implementations.

2. No universal workpad abstraction in the core.
   - Linear may support a persistent editable workpad.
   - Future providers, including Lark, may need append-only comments or an external note backend.

3. No fake provider-agnostic default workflow.
   - The first workflow is explicitly `compat_linear_default`.
   - Future providers can add their own workflow bundles.

4. No second runtime state store for observability.
   - Snapshots and projections are enough for V1.

5. No oversized `WorkItem`.
   - `WorkItem` keeps only fields required for orchestration and prompt rendering.
   - Provider-specific metadata stays out of core logic.

## Proposed V1 Package Layout

V1 starts flatter than the final idealized structure. The goal is behavioral stability before package refinement.

```text
go-symphony/
  cmd/
    symphony/
  docs/
    plans/
  internal/
    cli/
    codex/
    config/
    dashboard/
    domain/
    httpapi/
    observability/
    orchestrator/
    runner/
    tracker/
    trackers/
      linear/
      memory/
    web/
    workflow/
    workspace/
```

Additional subpackages should be introduced only after a clear need emerges.

## Compatibility Checklist

The implementation should explicitly track parity for these areas:

- `WORKFLOW.md` path semantics
- `WORKFLOW.md` hot reload and last-known-good behavior
- prompt rendering behavior
- tracker polling and reconciliation behavior
- workspace hook lifecycle
- local and SSH worker execution behavior
- Codex app-server protocol handling
- `linear_graphql` dynamic tool behavior
- Linear reader semantics and normalization
- HTTP API routes and payload semantics
- terminal dashboard rendering behavior
- web dashboard at `/`
- CLI bootstrap and acknowledgement behavior
- shutdown and offline dashboard rendering

## Phase Plan

### Phase 1: Compatibility Contract and Core Shape

- Freeze compatibility contract and terminology mapping.
- Establish repo skeleton.
- Implement `WORKFLOW.md` loading and config normalization.
- Define core domain model.

### Phase 2: Core Runtime Closed Loop

- Implement orchestrator event loop and state machine.
- Implement workspace lifecycle.
- Implement runner / execution host boundary.
- Implement Codex app-server protocol compatibility.
- Implement memory tracker and local happy-path closed loop.

### Phase 3: Linear Compatibility Shell

- Implement Linear reader adapter.
- Implement Linear ToolBridge and `linear_graphql`.
- Implement `compat_linear_default`.
- Integrate the complete Linear-backed run path.

### Phase 4: Observability and User-Facing Surfaces

- Implement observability projection layer.
- Implement HTTP API compatibility.
- Implement terminal dashboard compatibility.
- Implement web dashboard compatibility.
- Implement CLI/bootstrap/shutdown behavior.

### Phase 5: Parity Hardening

- Add full test matrix.
- Run live parity checks.
- Record intentional compatibility gaps, if any remain.

## Rough Task Breakdown

### T01 Compatibility Contract

Goal: Freeze Symphony to go-symphony compatibility scope, terminology mapping, and explicit non-goals.

Acceptance:

- parity checklist exists
- terminology mapping exists
- in-scope and out-of-scope items are explicit

### T02 Repo Skeleton

Goal: Establish the V1 repository structure and top-level package layout.

Depends on: `T01`

Acceptance:

- repository layout exists
- packages remain provider-neutral in the core
- empty build is possible

### T03 `WORKFLOW.md` Loader

Goal: Implement front matter parsing, prompt/template loading, hot reload, and last-known-good semantics.

Depends on: `T02`

Acceptance:

- behavior matches current workflow loading semantics
- invalid reload keeps last known good config

### T04 Internal Config Model

Goal: Accept current external config shape while normalizing it into an internal provider-neutral model.

Depends on: `T03`

Acceptance:

- old-style input remains accepted
- internal config shape no longer leaks Linear-only concepts into the core

### T05 Domain Model

Goal: Define `WorkItem`, `Blocker`, `RunEvent`, `Snapshot`, `RetryEntry`, and `PollingState`.

Depends on: `T04`

Acceptance:

- no `Linear` naming in core domain types
- no provider-specific write semantics in `WorkItem`

### T06 Orchestrator Core

Goal: Implement single-owner scheduling state with polling, claim, dispatch, retry, reconcile, stall recovery, and snapshot generation.

Depends on: `T05`

Acceptance:

- orchestrator is sole runtime state owner
- workers only report through `RunEvent`

### T07 Workspace Lifecycle

Goal: Implement workspace paths, safe identifiers, hooks, cleanup, and terminal-state workspace cleanup.

Depends on: `T06`

Acceptance:

- `after_create`
- `before_run`
- `after_run`
- `before_remove`

must match current semantic behavior for success, failure, and timeout.

### T08 Runner / ExecutionHost

Goal: Separate local/SSH execution, host capacity semantics, and remote process launch from workspace logic.

Depends on: `T07`

Acceptance:

- local and SSH execution share the same runtime contract
- SSH-specific behavior is isolated outside `workspace`

### T09 Codex App-Server Protocol

Goal: Implement app-server lifecycle, session bootstrap, `thread/start`, `turn/start`, tool calls, approvals, timeouts, and event normalization.

Depends on: `T08`

Acceptance:

- treated as a protocol compatibility target
- not implemented as a thin stdio wrapper only

### T10 `TrackerReader` + Memory Adapter

Goal: Finalize the core tracker read interface and provide a memory adapter for tests and local runtime closure.

Depends on: `T05`, `T06`

Acceptance:

- core tracker surface is read-only
- memory path can drive integration tests without Linear

### T11 Linear Reader Adapter

Goal: Implement current Symphony-compatible Linear read behavior.

Depends on: `T10`

Acceptance:

- candidate fetch
- state fetch by names
- state refresh by IDs
- normalization
- pagination
- assignee routing
- error classification

### T12 Linear ToolBridge

Goal: Implement `linear_graphql` and other Linear-specific write capabilities in the compatibility shell.

Depends on: `T09`, `T11`

Acceptance:

- write behavior does not expand core tracker interfaces
- `linear_graphql` is injected through the compatibility shell

### T13 Linear Workflow Bundle

Goal: Implement the workflow selector plus `compat_linear_default`.

Depends on: `T12`

Acceptance:

- workflow selection can grow later
- first concrete workflow is explicitly Linear-specific

### T14 End-to-End Run Integration

Goal: Connect orchestrator, workspace, runner, codex, tracker, tool bridge, and workflow into complete run lifecycles.

Depends on: `T06`, `T07`, `T08`, `T09`, `T11`, `T12`, `T13`

Acceptance:

- memory and Linear paths both run end to end
- continuation, retry, and terminal cleanup behavior are covered

### T15 HTTP API Compatibility

Goal: Recreate `/api/v1/state`, `/api/v1/refresh`, and `/api/v1/:issue_identifier` compatibility behavior.

Depends on: `T14`

Acceptance:

- DTO fields are compatibility-checked
- error codes and refresh semantics match the current product

### T16 Terminal Dashboard Compatibility

Goal: Recreate terminal dashboard output behavior, including ANSI rendering, throughput, retry queue, rate limits, event humanization, and fixture testing.

Depends on: `T14`, `T15`

Acceptance:

- snapshot fixtures validate output
- dashboard is treated as a compatibility surface, not just a convenience view

### T17 Web Dashboard + Static Assets

Goal: Recreate web observability at `/` and related static assets.

Depends on: `T15`

Acceptance:

- current web dashboard capability is present
- static assets are available

### T18 CLI + Full Parity Test Sweep

Goal: Recreate CLI behavior, acknowledgement flow, logs-root/port options, offline shutdown display, and run the full parity-oriented test matrix.

Depends on: `T14`, `T15`, `T16`, `T17`

Acceptance:

- CLI semantics are tested
- unit, adapter, integration, snapshot, and live e2e coverage exist
- remaining intentional gaps, if any, are documented

## Review Outcome

This reviewed version is preferred over the original rough draft because it:

- keeps the core narrow
- removes fake universal tracker write abstractions
- avoids pretending the Linear workflow is already generic
- avoids building a second observability state system
- preserves the real extension points needed for future Lark support

## Expected Future Extension Points

These are the intended long-term seams:

- `domain.WorkItem`
- `tracker.TrackerReader`
- `domain.RunEvent`
- `runner.ExecutionHost`
- provider-specific ToolBridge implementations
- workflow selector with provider-specific bundles

These seams are worth preserving from day one.
