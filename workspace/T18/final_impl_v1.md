# T18 Final Implementation Proposal V1

## Goal

Close the last V1 parity task by making the Go executable behave like the Symphony CLI host:

- require the explicit guardrails acknowledgement before startup side effects
- parse the optional workflow path plus `--logs-root` and `--port`
- apply CLI port precedence over workflow `server.port`
- configure log output under the selected logs root
- render the offline terminal dashboard on normal shutdown
- run and document the full parity-oriented verification matrix

## Scope

In scope:

- `internal/cli` process-level parsing, output, startup, shutdown, and test seams
- narrow `StartRuntime` support for CLI port override
- lightweight log file path/configuration helpers in `internal/cli`
- e2e-tagged no-network runtime smoke coverage for the final composed dashboard/API path
- OpenSpec capability for CLI/bootstrap parity
- T18 workspace parity matrix and residual-risk notes

Out of scope:

- changing orchestrator scheduling semantics
- changing tracker read/write behavior
- introducing generic provider write APIs
- changing web or HTTP API payload contracts except through already exposed runtime mounting
- real Linear/Codex live e2e unless credentials and environment opt-ins are available
- adding a new logging framework or log rotation layer in V1

## CLI Contract

Add the acknowledgement flag:

```text
--i-understand-that-this-will-be-running-without-the-usual-guardrails
```

If it is absent, the CLI prints the guardrails banner to stderr and exits nonzero before checking the workflow file, configuring logs, applying port overrides, or starting runtime.

Accepted syntax:

```text
symphony [--logs-root <path>] [--port <port>] [path-to-WORKFLOW.md]
```

Rules:

- zero positional args means `WORKFLOW.md` in the current working directory
- one positional arg is the workflow path
- two or more positional args return usage
- unknown flags return usage
- `--logs-root` requires a nonblank value; if repeated, the last value wins
- `--port` requires an integer `>= 0`; if repeated, the last value wins
- explicit and default workflow paths are expanded before file existence checks
- missing workflow file returns `Workflow file not found: <expanded path>`
- runtime startup errors return `Failed to start Symphony with workflow <expanded path>: <reason>`

## Runtime Wiring

Extend `RuntimeOptions` with:

```go
ServerPortOverride *int
```

`StartRuntime` applies this after loading settings and before constructing workspace, tracker, orchestrator, or HTTP server state. A non-nil override replaces `settings.Server.Port`, including `0`.

Keep `server.host` in workflow config. For operator-facing `DashboardURL`, bind using the configured host but display wildcard hosts as loopback:

- `""`, `0.0.0.0`, `::`, `[::]` -> `127.0.0.1`
- IPv6 literals are bracketed
- the actual listener port is used when `0` is configured

## Logging

Add a small `internal/cli` log helper:

- default root: current working directory
- custom root: `--logs-root`
- log path: `<root>/log/symphony.log`
- create parent directories
- redirect Go's standard logger to the file for the process lifetime

This is not a full rotating logger. The parity-relevant user behavior is that `--logs-root` changes where Symphony writes process logs.

## Shutdown and Terminal Output

On normal context cancellation:

1. stop any CLI-owned terminal dashboard loop, if running
2. close the runtime
3. render `dashboard.RenderOffline()` to stdout once
4. return exit code `0`

Startup failures must not render the offline frame.

If runtime close returns an error, print it to stderr and return nonzero after best-effort cleanup.

The implementation may wire live terminal rendering with `observability.Projector`, `dashboard.Render`, and `dashboard.RenderGate`, but this must stay projection-only over `Runtime.Snapshot()` and must not create a second runtime state owner.

## Tests

Use TDD. Write failing tests before production edits.

Package-level CLI tests:

- missing acknowledgement returns banner and performs no startup side effects
- default workflow path is `WORKFLOW.md`
- explicit workflow path is expanded and passed to runtime
- missing workflow file returns the compatibility error
- unknown flags, extra positionals, blank logs root, and invalid port return usage
- `--logs-root` expands the selected root and configures `<root>/log/symphony.log`
- `--port 0` passes a non-nil override to runtime
- startup error includes the expanded workflow path
- normal context cancellation closes runtime and renders `app_status=offline`
- startup error does not render offline

Runtime integration tests:

- `ServerPortOverride` takes precedence over workflow `server.port`
- ephemeral port override exposes `/`, `/dashboard.css`, and `/api/v1/state`
- dashboard URL uses the live bound port and loopback display for wildcard hosts

E2E-tagged smoke:

- with a memory tracker and scripted Codex transport, runtime starts with `ServerPortOverride=0`
- the live HTTP dashboard/API path is reachable
- shutdown closes without leaking the listener

## Verification Matrix

Run these before closing:

```bash
go test -count=1 ./internal/cli/... ./cmd/symphony/...
go test -count=1 ./internal/web/... ./internal/httpapi/... ./internal/dashboard/... ./internal/observability/...
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

Record live-provider e2e limits in `workspace/T18/todo.md` if no real Linear/Codex credentials are available.
