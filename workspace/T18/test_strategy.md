# T18 Test Strategy

## Purpose

T18 proves the executable host behavior that was intentionally deferred from earlier runtime, API, terminal dashboard, and web dashboard tasks. The meaningful evidence is process-level CLI behavior plus one final no-network composed-runtime smoke.

## Behavior-to-Test Mapping

| Behavior | Primary tests | What the tests prove |
| --- | --- | --- |
| Guardrails acknowledgement | `go test ./internal/cli/... -run TestMainRequiresGuardrailsAcknowledgement` | The CLI stops before file checks, log setup, port override, or runtime startup when the long acknowledgement flag is missing. |
| Argument parsing | `go test ./internal/cli/... -run TestParseCLIArgs` | Workflow path, `--logs-root`, and `--port` match Symphony syntax, including order-agnostic flags, repeated values, invalid values, and usage errors. |
| Workflow startup errors | `go test ./internal/cli/... -run TestMainWorkflow` | Missing workflow files and runtime startup failures are surfaced with operator-facing messages and do not render offline status. |
| Log root behavior | `go test ./internal/cli/... -run TestConfigureLogFile` | `--logs-root` is expanded before building `<root>/log/symphony.log`, parent directories are created, and tests restore process-wide logger state. |
| Normal shutdown | `go test ./internal/cli/... -run TestMainRendersOfflineOnNormalShutdown` | Context cancellation closes the runtime and emits exactly the minimal offline dashboard frame without normal running/retry sections. |
| Runtime port override | `go test ./internal/cli/... -run TestRuntimeServerPortOverride` | CLI `--port` semantics are represented in runtime options and override workflow `server.port`, including ephemeral port `0`. |
| Dashboard/API reachability | `go test ./internal/cli/... -run TestRuntimeStartsWebServerWhenPortConfigured` plus e2e-tagged smoke | The composed runtime still serves `/`, static assets, and `/api/v1/state` when a port is active. |
| Thin executable bootstrap | `go test ./cmd/symphony/...` | `cmd/symphony` still delegates to `internal/cli` rather than duplicating host logic. |

## Verification Commands

Targeted gates:

```bash
go test -count=1 ./internal/cli/... ./cmd/symphony/...
go test -count=1 ./internal/web/... ./internal/httpapi/... ./internal/dashboard/... ./internal/observability/...
```

Broad gates:

```bash
go test -count=1 ./...
make build
make lint
make test
make test-e2e
make verify
openspec validate --type change cli-full-parity-test-sweep
openspec validate --specs
git diff --check
```

## E2E Position

`make test-e2e` must run in the default environment. For T18, it should include a no-network e2e-tagged smoke test that starts the memory-provider runtime, binds an ephemeral dashboard/API listener, and shuts down cleanly.

Real Linear/Codex live e2e remains environment-gated. If credentials or opt-in environment variables are not available, that check must be recorded in `workspace/T18/todo.md` as not run rather than implied by the no-network smoke.

## Acceptance Threshold

- All targeted CLI/runtime tests pass.
- Broad repo verification passes.
- OpenSpec validates the active change and synced specs.
- Any skipped real-provider e2e is explicitly documented.
