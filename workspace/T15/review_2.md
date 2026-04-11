# T15 Final Impl V1 Review 2

Score: 78/100

## High Severity

None reported.

## Medium

- `RefreshResult.Available` is not Go-native. Current orchestrator refresh exposes only `Queued` and `Coalesced`; unavailable should use an error path or be handled as nil runtime mapping.
- The `internal/cli` refresh delegation test is not optional if T15 claims to add that seam. If CLI remains out of scope, remove that ownership from T15.
- `recent_events` is under-specified. Current Go has no event history, only last event fields. If projected, document it as a deliberate last-event inference and pin the shape in tests.

## Low

- Optional `server.go` may overreach into T18.

## Required Fixes

1. Replace `Available` with error-based unavailable handling.
2. Either make CLI refresh delegation mandatory or remove CLI ownership.
3. Pin or defer `recent_events`.
4. Keep server helper out unless a hard consumer appears now.
