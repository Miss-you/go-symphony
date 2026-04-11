# T15 Final Comparison

## Compared Sources

- `workspace/T15/original_impl.md`
- `workspace/T15/final_impl.md`
- `openspec/changes/http-api-compatibility/specs/http-api-compatibility/spec.md`
- `internal/httpapi/handler.go`
- `internal/httpapi/dto.go`
- `internal/httpapi/handler_test.go`

## Result

No high-severity parity or scope drift found.

## Parity Checks

- `GET /api/v1/state` is implemented as a JSON handler route and projects `generated_at`, counts, running entries, retrying entries, Codex totals, and rate limits.
- Running and retrying entries preserve compatibility field names: `issue_id`, `issue_identifier`, `last_event`, `last_message`, `started_at`, `last_event_at`, and token fields.
- Empty arrays and `null` timestamp/rate-limit behavior are covered by package tests.
- `GET /api/v1/:issue_identifier` resolves only from current snapshot running/retrying entries, matching the Elixir in-memory lookup behavior.
- Running entries win when a running and retry entry share the same identifier.
- `POST /api/v1/refresh` returns `202` with queued/coalesced/requested_at/operations on success and `503 orchestrator_unavailable` on refresh errors.
- Snapshot timeout and unavailable are preserved as HTTP `200` state payloads with compatibility error envelopes.
- Unsupported methods and unknown routes use the expected JSON error envelopes.

## Boundary Checks

- `internal/httpapi` uses package-local `SnapshotFunc` and `RefreshFunc` seams.
- It does not import orchestrator, CLI, tracker, provider-specific adapters, or toolbridge packages.
- It remains an `http.Handler` compatibility layer and does not start a listener or own runtime state.

## Recorded Residuals

The remaining differences are intentional and recorded in `workspace/T15/todo.md`:

- CLI/server lifecycle belongs to T18.
- Web dashboard and static assets belong to T17.
- Terminal dashboard and full Codex message humanization belong to T16.
- Live runtime adapters for snapshot timeout/unavailable can be mapped in later server wiring; the HTTP envelopes are already implemented and tested.
