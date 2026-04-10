## Why

`T04` froze the provider-neutral runtime config contract, but the Go port still lacks the provider-neutral runtime domain types that the orchestrator, tracker adapters, Codex integration, and observability layers must share. `T05` needs to land that contract now so later tasks stop depending on placeholder packages, ad hoc maps, or Linear-shaped runtime structs.

## What Changes

- add a typed provider-neutral runtime domain model under `internal/domain`
- freeze the stable exported surface around `WorkItem`, `Blocker`, `ActiveRun`, `RetryEntry`, `PollingState`, `Snapshot`, `RunEvent`, `RunEventKind`, `CodexTotals`, and the rate-limit helper types
- keep `WorkItem` limited to fields already required by current orchestration and prompt/template compatibility
- define snapshot and run-event semantics that preserve running-item, retry, polling, and aggregate Codex observability facts without exposing orchestrator-private refs
- add package-scoped contract tests that lock the exported domain shape for downstream tasks

## Capabilities

### New Capabilities
- `runtime-domain-model`: provider-neutral runtime vocabulary for work items, blockers, worker events, retry state, polling state, and snapshots

### Modified Capabilities

None.

## Impact

- `internal/domain/`
- later core packages that will compile against the exported domain contract, especially `internal/orchestrator`, `internal/tracker`, `internal/codex`, and `internal/observability`
- `workspace/T05/`
- `docs/plans/2026-04-10-go-symphony-design-task.md`
- `openspec/specs/runtime-domain-model/spec.md`
