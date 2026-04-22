# T20 TODO

## Completed
- [x] slog + lumberjack infrastructure (`internal/logging`)
- [x] Log file redirection with rotation (`internal/cli/log_file.go`)
- [x] Codex stderr capture (no longer discarded)
- [x] Linear API error visibility (`listCandidates`, `refreshItems`)
- [x] HTTP server error capture
- [x] Runtime startup/shutdown structured logs
- [x] Orchestrator lifecycle logs (dispatch, stall, retry, invalidate)
- [x] Workspace lifecycle logs (create, remove, hook)
- [x] Config reload failure logs
- [x] Signal handling logs
- [x] All existing tests pass (`go test ./...`)
- [x] Build passes (`make build`)

## Future Enhancements (Out of Scope for T20)
- [ ] Add dedicated tests for `internal/logging` (rotation behavior, level filtering)
- [ ] Make log level configurable via WORKFLOW.md or CLI flag
- [ ] Add request-level logging to `internal/httpapi`
- [ ] Add per-item trace/correlation ID propagation through all logs
- [ ] Metrics export (Prometheus-style counters for runs, retries, stalls)
