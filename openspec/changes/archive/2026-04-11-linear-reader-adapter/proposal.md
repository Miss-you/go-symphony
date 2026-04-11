## Why

The Go core already freezes a provider-neutral read contract, but `internal/trackers/linear` is still a placeholder. T11 needs the real Linear reader behavior so the Go implementation can match Symphony's current candidate polling, cleanup-oriented state reads, refresh-by-ID behavior, routing, and error handling without widening the core into provider-specific write paths.

## What Changes

- Add a real read-only Linear tracker adapter under `internal/trackers/linear`.
- Normalize Linear issue payloads into `domain.WorkItem` values for candidate, state-based, and refresh-by-ID reads.
- Preserve Linear candidate pagination, refresh-by-ID ordering, assignee routing, and error classification.
- Keep Linear writes, `linear_graphql`, and toolbridge behavior out of this change.
- Keep `TrackerReader` read-only and provider-neutral in the core.

## Capabilities

### New Capabilities
- `linear-reader-adapter`: Linear reader behavior for candidate fetch, state-based reads, refresh-by-ID, routing, normalization, and error classification.

### Modified Capabilities


## Impact

- `internal/trackers/linear` gains the first concrete reader implementation and package-scoped tests.
- `internal/tracker` remains read-only and provider-neutral.
- `domain.WorkItem` continues to carry the routing and normalization fields that the Linear adapter populates.
- Later workflow/toolbridge work remains deferred to T12 and beyond.
