## Why

The Go runtime now exposes the snapshot data needed by observability surfaces, but the terminal dashboard is still only a package stub. T16 closes that user-visible parity gap by rendering the Symphony-compatible terminal status frame from `domain.Snapshot`.

## What Changes

- Add a projection-only dashboard view model under `internal/observability`.
- Add a pure ANSI terminal renderer under `internal/dashboard`.
- Add a small presentation-only render gate that preserves live redraw cadence, coalescing, and idle rerender semantics.
- Add Codex event humanization for the dashboard-compatible event text already carried by runtime snapshots.
- Add exact snapshot fixtures and executable fixture provenance checks against copied Elixir source fixtures.

## Capabilities

### New Capabilities

- `terminal-dashboard-compatibility`: Terminal dashboard projection, ANSI rendering, live redraw timing, event humanization, and fixture provenance.

### Modified Capabilities

- None.

## Impact

- Affected code: `internal/observability`, `internal/dashboard`, and a narrow `internal/cli` event-message summarization call.
- Affected tests: package tests for observability projection, dashboard fixtures/live gate/humanization, and CLI event message emission.
- No new third-party dependencies.
- No change to orchestrator state ownership, tracker interfaces, HTTP API routes, or web dashboard behavior.
