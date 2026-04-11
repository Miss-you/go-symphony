# T15 Current Go Implementation Research

## Existing Primitives

- `internal/domain.Snapshot` already carries the projection source for observability: `Running`, `Retrying`, `Polling`, `CodexTotals`, and `RateLimits`.
- `internal/domain.ActiveRun` exposes the running item fields needed by the API: item ID, item identifier, state, worker host, workspace path, session ID, turn count, started time, last event time, last event kind, last event message, and Codex totals.
- `internal/domain.RetryEntry` exposes the retry item fields needed by the API: item ID, item identifier, attempt, due time, last error, worker host, and workspace path.
- `internal/orchestrator.Service` already provides the narrow runtime seam: `Snapshot()`, `RequestRefresh()`, `ApplyRunEvent()`, and `Close()`.
- `internal/cli.Runtime` exposes `Snapshot()` but does not yet expose `RequestRefresh()`.
- `config.Settings` already includes `Workspace.Root` and `Server.Host` / `Server.Port`.
- `internal/httpapi` is currently only a package stub.

## Missing Pieces

- No HTTP handler, route wiring, DTOs, or error envelope exist.
- No Go equivalent of the presenter exists for `/api/v1/state`, `/api/v1/refresh`, or `/api/v1/:issue_identifier`.
- No runtime facade exists for `httpapi` to call both `Snapshot()` and `RequestRefresh()` from `cli.Runtime`.
- No HTTP server bootstrap consumes `config.Settings.Server`; this may be limited to a package-level `Server` helper for T15 and fully wired to CLI in T18.
- No tests freeze compatibility response fields, status codes, or method handling.

## Package Boundary Constraints

- `internal/httpapi` must consume `domain.Snapshot` and a small runtime interface. It must not reach into orchestrator-private state.
- The API must not widen `internal/tracker` or introduce tracker writes.
- The API must not introduce a second observability state machine. DTOs should be pure projections of a supplied snapshot plus config.
- Compatibility field names may use `issue_*` in JSON because the HTTP API is a compatibility surface; core Go types should continue using item-neutral names.
- Web dashboard and static assets belong to later tasks. T15 can reserve behavior through 404/405 routing without implementing `/` UI.

## Implementation Seam

Use a thin `internal/httpapi` package with:

- a `Runtime` interface:
  - `Snapshot() domain.Snapshot`
  - `RequestRefresh() RefreshResult`
- an `Options` struct:
  - runtime
  - workspace root
  - optional clock for deterministic tests
- a `Handler` implementing `http.Handler`
- DTO conversion helpers for state, issue detail, retry, tokens, rate limits, errors, and refresh acknowledgement

Expose a `RefreshResult` in `httpapi` or reuse an adapter-compatible local struct so the package does not depend on `internal/orchestrator`.

Add `RequestRefresh()` to `internal/cli.Runtime` so it satisfies the HTTP runtime interface without exposing orchestrator internals.

## Testing Hooks

Package tests can use `httptest` with a fake runtime returning a deterministic `domain.Snapshot` and refresh result.

Required package-level tests:

- `GET /api/v1/state` returns compatibility DTO fields from a static snapshot.
- `GET /api/v1/:issue_identifier` returns running detail and retry detail.
- unknown issue returns `404 issue_not_found`.
- unsupported methods on defined routes return `405 method_not_allowed`.
- unknown routes return `404 not_found`.
- `POST /api/v1/refresh` returns `202` with queued/coalesced/operations/requested_at.
- unavailable refresh returns `503 orchestrator_unavailable`.

Integration-level tests can check `cli.Runtime` satisfies snapshot and refresh behavior after exposing `RequestRefresh()`.

## Risks And Decisions

- The current Go snapshot cannot represent snapshot timeout/unavailable because `Snapshot()` is synchronous and returns an empty value for nil runtime. T15 should model the normal and nil-runtime paths, but not invent timeout machinery unless a real async snapshot seam appears.
- Elixir computes issue `workspace.path` from config root plus identifier, while Go `ActiveRun` and `RetryEntry` already carry `WorkspacePath`. To preserve compatibility and retain actual runtime evidence, use the entry workspace path when present and fall back to `filepath.Join(workspaceRoot, identifier)`.
- Full Codex message humanization belongs with dashboard/event presentation. T15 should expose `last_message` and `recent_events` from existing `LastEventMessage` without expanding the dashboard task.
- Server host/port lifecycle can be implemented as a small helper in `httpapi` if needed by tests, but CLI flag parity should remain T18.
