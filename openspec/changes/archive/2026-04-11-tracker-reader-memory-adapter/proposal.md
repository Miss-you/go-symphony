## Why

`T06` proved the Go scheduler core can operate on provider-neutral `domain.WorkItem` values, but `internal/tracker` and `internal/trackers/memory` are still placeholders. `T10` needs to freeze the read-only tracker boundary now so later Linear, workspace-cleanup, and end-to-end wiring tasks stop depending on ad hoc assumptions instead of a stable core contract.

## What Changes

- add the first real provider-neutral `TrackerReader` contract under `internal/tracker`
- freeze the tracker read surface to candidate listing, state-based listing, and refresh-by-id reads only
- add a deterministic in-memory tracker reader under `internal/trackers/memory`
- lock contract semantics with package-scoped tests, including empty-input behavior, normalized state matching, request-order refresh, and deep-copy isolation for returned `domain.WorkItem` values
- explicitly defer tracker write behavior, runtime adoption into `internal/orchestrator`, and Linear-specific normalization to later tasks

## Capabilities

### New Capabilities

- `runtime-tracker-reader`: provider-neutral tracker read contract plus deterministic memory-backed implementation for local/test use

### Modified Capabilities

None.

## Impact

- Affected code: `internal/tracker`, `internal/trackers/memory`
- Closely related existing code: `internal/domain`, `internal/config`
- Downstream dependents unlocked: later `internal/trackers/linear`, workspace-cleanup wiring, and runtime integration tasks
- Affected task artifacts: `workspace/T10/`, `docs/plans/2026-04-10-go-symphony-design-task.md`
