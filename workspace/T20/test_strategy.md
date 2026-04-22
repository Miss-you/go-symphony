# T20 Test Strategy: Logging and Observability

## Goal
Replace ad-hoc stdlib logging with structured `slog` + `lumberjack` rotation, capture silent failure paths, and add runtime lifecycle logs.

## Verification Gates

### 1. Build / Compile
- `go build ./...` must pass with zero errors.
- New package `internal/logging` must compile without issues.

### 2. Unit Tests
- All existing tests must continue to pass (`go test ./...`).
- `internal/cli` tests verify log file redirection and restore behavior.
- No regressions in `internal/orchestrator`, `internal/codex`, `internal/workspace`, `internal/config`.

### 3. Integration / Behavioral Verification
- `internal/cli/log_file.go`: Verify that `configureLogFile` creates a rotated log file and restores both `slog.Default()` and `log.Writer()` on close.
- `internal/codex/session.go`: Verify Codex stderr is captured via `slog.Debug` instead of being discarded to `io.Discard`.
- `internal/orchestrator/service.go`: Verify `listCandidates` errors are logged via `slog.Error` instead of silently swallowed.
- `internal/orchestrator/state.go`: Verify `reconcileRunning` errors are logged; `reconcileStalled`, `dispatchItem`, `invalidateRun`, and `scheduleFailureRetry` emit structured logs.
- `internal/cli/runtime.go`: Verify HTTP server errors are logged; startup and shutdown emit structured summary logs.

### 4. Manual / Live Verification
- Run `./symphony-verify linear --limit 10 WORKFLOW.md` and inspect `log/symphony.log` for debug-level output.
- Confirm log file rotates when size threshold is hit (can be simulated with small `MaxSize` config).

## What We Are NOT Testing
- We are not adding new business logic; logs are additive observability.
- We are not changing error return values or public API signatures.
