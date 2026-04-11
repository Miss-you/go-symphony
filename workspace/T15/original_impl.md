# T15 Original Implementation Research

## Source Files

- `/Users/apple/Documents/Github/symphony/SPEC.md`, section 13.7.2
- `/Users/apple/Documents/Github/symphony/elixir/lib/symphony_elixir_web/router.ex`
- `/Users/apple/Documents/Github/symphony/elixir/lib/symphony_elixir_web/controllers/observability_api_controller.ex`
- `/Users/apple/Documents/Github/symphony/elixir/lib/symphony_elixir_web/presenter.ex`
- `/Users/apple/Documents/Github/symphony/elixir/lib/symphony_elixir/orchestrator.ex`
- `/Users/apple/Documents/Github/symphony/elixir/lib/symphony_elixir/http_server.ex`
- `/Users/apple/Documents/Github/symphony/elixir/test/symphony_elixir/extensions_test.exs`

## Route And Method Matrix

| Route | Supported Method | Success | Unsupported Method |
| --- | --- | --- | --- |
| `/api/v1/state` | `GET` | `200` JSON state summary | `405 method_not_allowed` |
| `/api/v1/refresh` | `POST` | `202` JSON refresh acknowledgement | `405 method_not_allowed` |
| `/api/v1/:issue_identifier` | `GET` | `200` JSON issue detail | `405 method_not_allowed` |
| `/` | `GET` is web dashboard | handled outside T15 | non-GET returns `405 method_not_allowed` |
| other paths | - | - | `404 not_found` |

The router gives `GET /api/v1/:issue_identifier` a literal snapshot lookup by issue identifier. It does not query the tracker for unknown identifiers.

## State Payload

`GET /api/v1/state` returns:

- `generated_at`
- `counts.running`
- `counts.retrying`
- `running[]`
- `retrying[]`
- `codex_totals`
- `rate_limits`

Each `running[]` entry contains:

- `issue_id`
- `issue_identifier`
- `state`
- `session_id`
- `turn_count`
- `last_event`
- `last_message`
- `started_at`
- `last_event_at`
- `tokens.input_tokens`
- `tokens.output_tokens`
- `tokens.total_tokens`

Each `retrying[]` entry contains:

- `issue_id`
- `issue_identifier`
- `attempt`
- `due_at`
- `error`

The presenter humanizes `last_message` through `StatusDashboard.humanize_codex_message/1`. T15 should preserve the field and avoid blocking later T16 dashboard work on a full humanizer port.

## Issue Payload

`GET /api/v1/:issue_identifier` returns a debug payload for an item present in either the running or retrying snapshot list:

- `issue_identifier`
- `issue_id`
- `status`, either `running` or `retrying`, with running winning if both entries exist
- `workspace.path`, derived from configured workspace root joined with the issue identifier
- `attempts.restart_count`
- `attempts.current_retry_attempt`
- `running`, or `null`
- `retry`, or `null`
- `logs.codex_session_logs`, currently `[]`
- `recent_events`, present and often `[]`
- `last_error`, derived from retry entry when present
- `tracked`, currently `{}`

For retry entries, `restart_count` is `max(attempt - 1, 0)`. Missing issue identifiers return `404` with `issue_not_found`.

## Refresh Payload

`POST /api/v1/refresh` calls `Orchestrator.request_refresh/1` and returns `202 Accepted` on success:

- `queued`
- `coalesced`
- `requested_at`
- `operations`

The original refresh operation is a best-effort global trigger for immediate poll and reconciliation. If a poll is already running or already due, it is coalesced. If the orchestrator is unavailable, the controller returns `503 orchestrator_unavailable`.

## Error Semantics

All API errors use:

```json
{"error":{"code":"...","message":"..."}}
```

Observed error mappings:

| Condition | Status | Code | Message |
| --- | --- | --- | --- |
| unknown issue in issue route | `404` | `issue_not_found` | `Issue not found` |
| unsupported method on known route | `405` | `method_not_allowed` | `Method not allowed` |
| unknown route | `404` | `not_found` | `Route not found` |
| refresh with unavailable orchestrator | `503` | `orchestrator_unavailable` | `Orchestrator is unavailable` |
| state snapshot timeout | `200` | `snapshot_timeout` | `Snapshot timed out` |
| state snapshot unavailable | `200` | `snapshot_unavailable` | `Snapshot unavailable` |

The Elixir API returns state snapshot timeout/unavailable as a `200` payload with an `error` object, not as HTTP 5xx.

## Server Details

`HttpServer.start_link/1` starts the optional Phoenix observability endpoint. It accepts host, port, orchestrator, and snapshot timeout options. Empty host defaults to `127.0.0.1`, `port: 0` binds an ephemeral port, and `port: nil` disables the server.

T15 should implement the HTTP API package and enough server construction to support the JSON API. Web dashboard and static assets are left to T17, and CLI parity flags/lifecycle are left to T18 unless needed as narrow glue.

## Explicit Unknowns

- The Elixir route layer does not document `Allow` headers for `405`.
- API array ordering follows snapshot ordering, not a route-specific sort.
- Rich Codex message humanization is shared with the dashboard in Elixir and may be completed in T16.
- The refresh endpoint accepts empty body and form posts in tests; it does not depend on a request body schema.
