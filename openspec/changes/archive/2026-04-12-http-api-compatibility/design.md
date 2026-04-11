## Context

The Go runtime already projects orchestrator-owned state through `domain.Snapshot`, including running items, retrying items, Codex totals, and rate-limit facts. The HTTP API is a compatibility surface, so it must expose Symphony's existing `issue_*` JSON field names while keeping provider-neutral core packages unchanged.

`internal/httpapi` is currently a stub. T15 adds the JSON API handler only; web dashboard assets, listener lifecycle, CLI flags, and full startup/shutdown integration remain later tasks.

## Goals / Non-Goals

**Goals:**

- Serve `GET /api/v1/state`, `POST /api/v1/refresh`, and `GET /api/v1/:issue_identifier` from a thin handler.
- Preserve Symphony-compatible JSON DTO fields, error envelopes, status codes, nullability, and empty-array behavior.
- Preserve state snapshot timeout and unavailable envelopes through typed snapshot errors.
- Keep the HTTP layer projection-only and free of orchestrator-private state, tracker APIs, CLI lifecycle, or provider-specific logic.
- Add deterministic package-level tests using fixed clock and fake state/refresh functions.

**Non-Goals:**

- Start or manage an HTTP listener from CLI config.
- Serve `/` web dashboard or static assets.
- Implement terminal dashboard event humanization.
- Query trackers by issue identifier or add tracker write APIs.
- Create a new observability state store.

## Decisions

### Use Function Seams Instead Of Runtime Imports

`internal/httpapi` will accept `SnapshotFunc` and `RefreshFunc` options. This avoids importing `internal/orchestrator`, `internal/cli`, or provider packages into the HTTP layer while still letting later CLI/server work adapt the active runtime.

Alternative considered: require a runtime interface with `Snapshot()` and `RequestRefresh()` methods. That would force this task either to change `internal/cli` now or to import orchestrator result types into `httpapi`. Function seams are narrower and easier to test.

### Keep Snapshot Timeout/Unavailable In Scope Through Typed Errors

Elixir returns `200` with an error envelope for snapshot timeout and unavailable states. The Go runtime does not currently expose async snapshot failures, but the HTTP handler can still preserve the contract by mapping `ErrSnapshotTimeout` and `ErrSnapshotUnavailable` from `SnapshotFunc`.

Alternative considered: defer these envelopes until an async runtime seam exists. That would drop a documented API behavior from T15, so it is not acceptable.

### Keep Routing Exact And Handler-Local

The handler will route fixed paths before issue-detail lookup:

- `/api/v1/state`
- `/api/v1/refresh`
- `/api/v1/<identifier>`

Unsupported methods on known routes return `405 method_not_allowed`; unknown paths return `404 not_found`. T15 returns `404` for `/` because T17 owns the web dashboard route.

### Preserve Existing JSON Field Names At The Compatibility Boundary

The Go domain remains provider-neutral, but JSON DTOs use Symphony's existing `issue_id` and `issue_identifier` names. `domain.ActiveRun` and `domain.RetryEntry` are mapped into DTOs without widening core types.

### Treat `recent_events` As A Last-Event Inference

Go does not yet keep a bounded recent-event history. The issue-detail response will emit an empty `recent_events` array unless a running entry has `LastEventAt`; in that case it emits one inferred event with `at`, `event`, and `message`. This keeps the JSON shape compatible without inventing a stateful event store.

## Risks / Trade-offs

- Snapshot timeout/unavailable can be tested through typed errors now, but live runtime adapters may not emit those errors until later server wiring. Mitigation: keep the handler behavior stable and let T18 choose adapter behavior.
- Full Codex message humanization is deferred. Mitigation: preserve `last_message` and `recent_events.message` fields using the current last-event message.
- `/` is not implemented in T15. Mitigation: return `404 not_found` now and let T17 compose the web route without changing API routes.
- Function seams require later wiring code. Mitigation: the handler stays easy to adapt because the seams match existing snapshot and refresh operations.
