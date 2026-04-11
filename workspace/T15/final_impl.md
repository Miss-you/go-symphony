# T15 Final Implementation Proposal V1

## Goal

Implement the Symphony-compatible JSON API for current runtime state:

- `GET /api/v1/state`
- `POST /api/v1/refresh`
- `GET /api/v1/:issue_identifier`

The implementation should be a compatibility shell over the existing Go runtime projection. It should not create a second observability state owner and should not widen the provider-neutral tracker or orchestrator boundaries.

## Scope

In scope:

- Add `internal/httpapi` handler, DTO projection, route handling, and error envelopes.
- Project `domain.Snapshot` into the existing Symphony JSON field names.
- Add state snapshot and refresh acknowledgement semantics through narrow function seams.
- Add package-level `httptest` coverage for state, issue, refresh, route, method, and error behavior.

Out of scope:

- Web dashboard at `/` and static asset serving. Those belong to T17.
- Terminal dashboard rendering and full Codex message humanization. Those belong to T16.
- CLI `--port`, logs root, acknowledgement flow, and full parity sweep. Those belong to T18.
- Tracker writes, provider-specific API expansion, or identifier lookups through the tracker.
- HTTP listener lifecycle and server host/port binding. T15 provides an `http.Handler`; T18 wires it into CLI/server lifecycle.

## Package Shape

Add `internal/httpapi` production files:

- `handler.go`
- `dto.go`

The handler should stay projection-only and use function seams instead of importing orchestrator or CLI packages:

```go
type SnapshotFunc func(context.Context) (domain.Snapshot, error)
type RefreshFunc func(context.Context) (RefreshResult, error)

type RefreshResult struct {
    Queued    bool
    Coalesced bool
}

type Options struct {
    Snapshot SnapshotFunc
    Refresh RefreshFunc
    WorkspaceRoot string
    Now func() time.Time
}
```

Add sentinel errors owned by `httpapi`:

```go
var (
    ErrSnapshotTimeout     = errors.New("snapshot timeout")
    ErrSnapshotUnavailable = errors.New("snapshot unavailable")
    ErrRefreshUnavailable  = errors.New("refresh unavailable")
)
```

`NewHandler(options Options) http.Handler` returns a handler that owns only routing and JSON projection. It uses `encoding/json` and `net/http`, not a framework. If `Options.Snapshot` is nil, state behaves as `ErrSnapshotUnavailable`. If `Options.Refresh` is nil, refresh behaves as `ErrRefreshUnavailable`.

The handler should use `errors.Is` for these sentinel errors so wrapped adapter errors still map to the compatibility envelopes.

## Route Semantics

Implement exact route handling for this task:

| Route | Method | Result |
| --- | --- | --- |
| `/api/v1/state` | `GET` | `200` state payload |
| `/api/v1/state` | any other method | `405 method_not_allowed` |
| `/api/v1/refresh` | `POST` | `202` refresh payload |
| `/api/v1/refresh` | any other method | `405 method_not_allowed` |
| `/api/v1/<identifier>` | `GET` | `200` issue payload or `404 issue_not_found` |
| `/api/v1/<identifier>` | any other method | `405 method_not_allowed` |
| other paths | any method | `404 not_found` |

T15 can return `404 not_found` for `/` because web dashboard implementation is not in this task. It should not preclude T17 from taking over `/`.

## DTO Semantics

### State

`GET /api/v1/state` returns:

- `generated_at`: UTC RFC3339 second precision, always present
- `counts.running`
- `counts.retrying`
- `running[]`
- `retrying[]`
- `codex_totals`
- `rate_limits`

Map Go fields to compatibility JSON:

| JSON field | Source | Nullability |
| --- | --- | --- |
| `running[].issue_id` | `ActiveRun.ItemID` | string, may be empty |
| `running[].issue_identifier` | `ActiveRun.ItemIdentifier` | string, may be empty |
| `running[].state` | `ActiveRun.State` | string, may be empty |
| `running[].session_id` | `ActiveRun.SessionID` | string, may be empty |
| `running[].turn_count` | `ActiveRun.TurnCount` | number |
| `running[].last_event` | `ActiveRun.LastEventKind` | string, empty kind encodes as `""` |
| `running[].last_message` | `ActiveRun.LastEventMessage` | string, empty message encodes as `""` |
| `running[].started_at` | `ActiveRun.StartedAt` | `null` if zero time |
| `running[].last_event_at` | `ActiveRun.LastEventAt` | `null` if nil |
| `running[].tokens.input_tokens` | `ActiveRun.CodexTotals.InputTokens` | number |
| `running[].tokens.output_tokens` | `ActiveRun.CodexTotals.OutputTokens` | number |
| `running[].tokens.total_tokens` | `ActiveRun.CodexTotals.TotalTokens` | number |
| `retrying[].issue_id` | `RetryEntry.ItemID` | string, may be empty |
| `retrying[].issue_identifier` | `RetryEntry.ItemIdentifier` | string, may be empty |
| `retrying[].attempt` | `RetryEntry.Attempt` | number |
| `retrying[].due_at` | `RetryEntry.DueAt` | `null` if zero time |
| `retrying[].error` | `RetryEntry.LastError` | string, may be empty |
| `codex_totals.input_tokens` | `Snapshot.CodexTotals.InputTokens` | number |
| `codex_totals.output_tokens` | `Snapshot.CodexTotals.OutputTokens` | number |
| `codex_totals.total_tokens` | `Snapshot.CodexTotals.TotalTokens` | number |
| `codex_totals.seconds_running` | `Snapshot.CodexTotals.SecondsRunning` | number |
| `rate_limits` | `Snapshot.RateLimits` | `null` if nil |

`running` and `retrying` should encode as empty arrays, not `null`.

If `SnapshotFunc` returns `ErrSnapshotTimeout`, state returns HTTP `200`:

```json
{"generated_at":"...","error":{"code":"snapshot_timeout","message":"Snapshot timed out"}}
```

If `SnapshotFunc` returns `ErrSnapshotUnavailable` or any other error, state returns HTTP `200`:

```json
{"generated_at":"...","error":{"code":"snapshot_unavailable","message":"Snapshot unavailable"}}
```

### Issue Detail

`GET /api/v1/:issue_identifier` searches the current snapshot's running and retrying entries by `ItemIdentifier`.

The response includes:

- `issue_identifier`
- `issue_id`
- `status`
- `workspace.path`
- `attempts.restart_count`
- `attempts.current_retry_attempt`
- `running`
- `retry`
- `logs.codex_session_logs`
- `recent_events`
- `last_error`
- `tracked`

Status rules:

- running entry present -> `running`
- retry entry only -> `retrying`
- both present -> `running`
- neither present -> `404 issue_not_found`

Workspace path rules:

- Use the running entry `WorkspacePath` when present.
- Otherwise use the retry entry `WorkspacePath` when present.
- Otherwise fall back to `filepath.Join(workspaceRoot, issueIdentifier)`.

Retry attempt rules:

- no retry -> `restart_count=0`, `current_retry_attempt=0`
- retry -> `current_retry_attempt=attempt`, `restart_count=max(attempt-1, 0)`

`running` detail fields:

| JSON field | Source | Nullability |
| --- | --- | --- |
| `session_id` | `ActiveRun.SessionID` | string, may be empty |
| `turn_count` | `ActiveRun.TurnCount` | number |
| `state` | `ActiveRun.State` | string, may be empty |
| `started_at` | `ActiveRun.StartedAt` | `null` if zero time |
| `last_event` | `ActiveRun.LastEventKind` | string, empty kind encodes as `""` |
| `last_message` | `ActiveRun.LastEventMessage` | string, empty message encodes as `""` |
| `last_event_at` | `ActiveRun.LastEventAt` | `null` if nil |
| `tokens.input_tokens` | `ActiveRun.CodexTotals.InputTokens` | number |
| `tokens.output_tokens` | `ActiveRun.CodexTotals.OutputTokens` | number |
| `tokens.total_tokens` | `ActiveRun.CodexTotals.TotalTokens` | number |

`retry` detail fields:

| JSON field | Source | Nullability |
| --- | --- | --- |
| `attempt` | `RetryEntry.Attempt` | number |
| `due_at` | `RetryEntry.DueAt` | `null` if zero time |
| `error` | `RetryEntry.LastError` | string, may be empty |

`recent_events` is a deliberate compatibility inference from the current Go last-event fields, because Go does not yet carry an event history. It should include one event only when running `LastEventAt` is present:

- `at`
- `event`
- `message`

Example:

```json
[{"at":"2026-04-12T00:00:00Z","event":"codex_event_received","message":"working"}]
```

Otherwise it should be an empty array. `logs.codex_session_logs` should be an empty array and `tracked` should be an empty object.

### Refresh

`POST /api/v1/refresh` calls `RefreshFunc` and returns `202 Accepted` when no error is returned:

- `queued`
- `coalesced`
- `requested_at`
- `operations`

`operations` should be `["poll", "reconcile"]` for available refreshes. If refresh is unavailable, return:

```json
{"error":{"code":"orchestrator_unavailable","message":"Orchestrator is unavailable"}}
```

with HTTP status `503`.

`ErrRefreshUnavailable` and any other refresh error map to the same `503 orchestrator_unavailable` envelope.

## Error Envelopes

Use the shared JSON shape:

```json
{"error":{"code":"...","message":"..."}}
```

Supported T15 error mappings:

- `404 issue_not_found`, `Issue not found`
- `404 not_found`, `Route not found`
- `405 method_not_allowed`, `Method not allowed`
- `503 orchestrator_unavailable`, `Orchestrator is unavailable`

Snapshot timeout and unavailable responses are in scope through the typed `SnapshotFunc` errors even though the current runtime adapter may not produce them yet.

## Tests

Add `internal/httpapi/handler_test.go` before production code.

Tests should prove:

- State DTO includes counts, running entry, retrying entry, token totals, and rate limits.
- Issue DTO returns running detail with workspace fallback, attempts, recent event, logs, and tracked fields.
- Issue DTO returns retry detail with retry attempt and last error.
- Unknown issue returns `404 issue_not_found`.
- Refresh success returns `202` with queued/coalesced/requested_at/operations.
- Refresh unavailable returns `503 orchestrator_unavailable`.
- State snapshot timeout returns `200 snapshot_timeout`.
- State snapshot unavailable returns `200 snapshot_unavailable`.
- Wrong methods on known routes return `405 method_not_allowed`.
- Unknown routes return `404 not_found`.

## Verification

Primary task gate:

```bash
go test ./internal/httpapi/...
```

Closure checks before marking T15 done:

```bash
go test ./...
make build
make lint
make test-e2e
make verify
openspec validate --type change http-api-compatibility
openspec validate --specs
git diff --check
```

If any closure check is not applicable or fails for pre-existing reasons, record the reason in `workspace/T15/todo.md` and the task board. The package-specific acceptance bar remains `go test ./internal/httpapi/...`.

## Risks

- The package can reproduce Elixir snapshot timeout/unavailable envelopes through typed errors, but T18 still needs to decide which runtime adapter can actually emit those errors.
- Full message humanization is deferred to dashboard work; T15 keeps the field stable and projects existing messages.
- T17 will need to compose the `/` web route with this API handler without changing API behavior.
- T18 will need to wire server lifecycle to CLI flags and shutdown behavior; T15 should not overreach into full CLI parity.
