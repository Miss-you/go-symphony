# T20 Final Implementation: Logging and Observability

## Problem Statement
The go-symphony runtime had minimal logging and several "silent failure" paths:
1. Codex app-server stderr was discarded (`io.Copy(io.Discard, stderr)`).
2. Linear API errors in `listCandidates` and `refreshItems` were silently swallowed.
3. HTTP server goroutine errors were discarded.
4. Runtime startup and shutdown produced no structured output.
5. Log file had no rotation and was written via ad-hoc stdlib `log` calls.

## Solution Overview

### 1. Structured Logger Infrastructure (`internal/logging`)
- New package `internal/logging` wraps `log/slog` (Go 1.21+ standard library) with `lumberjack.v2` for file rotation.
- Configuration: debug level, text format, 100MB max size, 5 backups, 30-day max age, compression enabled.
- `DefaultConfig(path)` provides sensible defaults.

### 2. Log File Integration (`internal/cli/log_file.go`)
- `configureLogFile` now:
  - Creates a `slog.Logger` via `internal/logging.New`.
  - Sets `slog.SetDefault(logger)` for structured logs.
  - Also sets `log.SetOutput(file)` so legacy `log.Print` calls and existing tests still work.
  - On restore, reverts both `slog.Default()` and `log.Writer/Flags/Prefix`.

### 3. Silent Failure Fixes

#### Codex stderr (`internal/codex/session.go`)
- Replaced `io.Copy(io.Discard, stderr)` with `bufio.NewScanner(stderr)` + `slog.Debug("codex stderr", "line", ...)`.
- Added `slog.Info` on session start, `slog.Debug` on session close, `slog.Error` on transport/bootstrap failures.

#### Orchestrator API errors (`internal/orchestrator/service.go`)
- `handlePollCycle`: `listCandidates` error now logged via `slog.Error` before being skipped.

#### Orchestrator state reconciliation (`internal/orchestrator/state.go`)
- `reconcileRunning`: `refreshItems` error now logged via `slog.Error`.
- `reconcileStalled`: stall detection emits `slog.Warn` with duration and item ID.
- `dispatchItem`: success emits `slog.Info`; failure emits `slog.Error` with attempt count.
- `invalidateRun`: emits `slog.Info` with item ID and state.
- `scheduleFailureRetry`: emits `slog.Warn` with attempt, delay, and error.
- `processCandidates`: emits `slog.Debug` for dispatch decisions and `slog.Info` for dispatched count.

#### HTTP server (`internal/cli/runtime.go`)
- `startHTTPServer`: goroutine now checks `errors.Is(err, http.ErrServerClosed)` and logs actual errors via `slog.Error`.
- `StartRuntime`: emits structured startup summary (provider, project, workspace root, max agents, dashboard URL).
- `Close`: emits shutdown start/completion logs; individual component close errors are logged.

#### Workspace lifecycle (`internal/workspace/manager.go`)
- `Create`: logs workspace creation success and `after_create` hook failure.
- `Remove`: logs workspace removal success and failure.
- `runHook`: logs hook failure (non-best-effort) and success (debug).

#### Config reload (`internal/config/store.go`)
- `reloadLocked`: workflow reload failure logged via `slog.Error`.

#### Signal handling (`cmd/symphony/main.go`)
- Added goroutine that waits on signal context and logs shutdown signal via `slog.Info`.

## Files Changed
- `go.mod`, `go.sum` — added `gopkg.in/natefinch/lumberjack.v2`
- `internal/logging/logging.go` — new file
- `internal/cli/log_file.go` — rewrote to use slog + lumberjack
- `internal/codex/session.go` — stderr capture, session lifecycle logs
- `internal/orchestrator/service.go` — listCandidates error logging
- `internal/orchestrator/state.go` — reconcile, dispatch, stall, retry, invalidate logs
- `internal/cli/runtime.go` — HTTP server error, startup/shutdown logs
- `internal/workspace/manager.go` — create/remove/hook logs
- `internal/config/store.go` — reload failure log
- `cmd/symphony/main.go` — signal shutdown log
