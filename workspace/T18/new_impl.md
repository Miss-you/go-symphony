# T18 Go Implementation Notes

## Sources Inspected

- `cmd/symphony/main.go`
- `cmd/symphony/main_test.go`
- `internal/cli/main.go`
- `internal/cli/runtime.go`
- `internal/cli/runtime_test.go`
- `internal/config/settings.go`
- `internal/config/settings_test.go`
- `internal/dashboard/renderer.go`
- `internal/dashboard/live.go`
- `internal/web/handler.go`
- `internal/web/handler_test.go`
- `openspec/specs/compatibility-contract/spec.md`
- `openspec/specs/terminal-dashboard-compatibility/spec.md`
- `openspec/specs/web-dashboard-static-assets/spec.md`

## Current Behavior

`cmd/symphony/main.go` is correctly thin: it creates a signal-aware context and delegates to `cli.Main`.

`internal/cli.Main` is currently too small for T18 parity:

- it treats `args[0]` as the workflow path
- it starts `StartRuntime`
- it prints startup errors to stderr
- it blocks until context cancellation
- it closes the runtime on return

There is no CLI parser, acknowledgement gate, usage text, logs-root handling, port override, or offline shutdown rendering.

`StartRuntime` already owns the right process assembly boundary:

- config store lifecycle
- tracker reader selection
- startup terminal cleanup
- workspace manager construction
- orchestrator startup
- worker/Codex session lifecycle
- configured HTTP server startup when `settings.Server.Port` is present

The lower layers needed by T18 already exist:

- `internal/dashboard.RenderOffline` emits the offline frame.
- `internal/dashboard.Render` plus `internal/observability.Projector` can render the terminal dashboard.
- `internal/web.NewHandler` serves `/`, static assets, and delegated `/api/v1/*`.
- Runtime tests already prove web routes are reachable when `server.port` is configured.

## Gaps

- CLI side effects are not ordered behind the acknowledgement gate.
- CLI cannot reject unknown options or extra positional arguments.
- CLI cannot override `server.port`.
- CLI cannot configure a log file root.
- CLI shutdown does not print the offline dashboard frame.
- No T18-specific tests cover startup stdout/stderr, exit codes, or port/log flag semantics.
- `make test-e2e` exists, but there is no e2e-tagged runtime smoke test for the final CLI-era server/dashboard path.

## Implementation Implications

The implementation should stay mostly in `internal/cli` and keep `cmd/symphony` thin.

The runtime may gain narrow support for CLI-owned settings:

- `RuntimeOptions.ServerPortOverride *int`
- a read-only way for CLI tests or dashboard wiring to inspect effective settings, if needed

It should not move CLI flag parsing into `config`, `web`, `httpapi`, or the orchestrator.

The terminal dashboard renderer can be used by CLI shutdown without changing renderer semantics. If live terminal output is wired, it must remain presentation-only over `Runtime.Snapshot()`.
