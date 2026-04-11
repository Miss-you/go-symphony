# T15 Final Impl V1 Review 1

Score: 66/100

## High Severity

- `final_impl_v1.md` defers snapshot timeout and unavailable handling, but `original_impl.md` records those as part of the HTTP contract. `/api/v1/state` must preserve the `200` error-envelope behavior. Fix by keeping exact `snapshot_timeout` and `snapshot_unavailable` envelopes in scope through a narrow typed snapshot-error seam that handler tests can exercise now.

## Medium And Low

- Pulling T15 into `internal/cli` and an optional `server.go` is boundary creep. T18 owns CLI/bootstrap/server lifecycle. T15 should stay at `internal/httpapi` DTOs, routing, and handler tests.
- DTOs are not pinned tightly enough for exact parity. Spell out `session_id`, `turn_count`, `started_at`, `last_event_at`, and null versus empty-array behavior.
- `RefreshResult.Available` invents extra state not present in the current orchestrator refresh result. Prefer a result plus error or sentinel mapping at the handler boundary.
- Verification section is broader than T15 needs. Keep the primary gate centered on `go test ./internal/httpapi/...`, with broader gates reserved for closure.
- `recent_events` one-item projection is an inference. Label it as deliberate or defer it.

## Required Fixes

1. Restore state timeout/unavailable envelopes.
2. Remove CLI and server helper ownership from T15.
3. Expand DTO tables.
4. Simplify refresh handling without `Available`.
5. Trim verification scope.

Process note about task-board state was checked against the wrong path; the isolated worktree task board already has `T14=done` and `T15=research`.
