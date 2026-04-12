## Why

The approved V1 design still has one unimplemented user-facing surface: the executable CLI host. Without this change, the Go binary can start the runtime, but it does not preserve Symphony's acknowledgement gate, CLI flag behavior, port precedence, log-root selection, or offline shutdown display.

## What Changes

- Add Symphony-compatible CLI parsing for the guardrails acknowledgement flag, optional workflow path, `--logs-root`, and `--port`.
- Fail startup before side effects when acknowledgement is missing, the workflow file is absent, arguments are invalid, or runtime startup fails.
- Apply CLI `--port` as an override over workflow `server.port`, including ephemeral port `0`.
- Configure the process log path under the selected logs root while keeping tests isolated from process-wide logger state.
- Render the minimal offline terminal dashboard frame once on normal shutdown.
- Add a parity-oriented verification sweep, including an e2e-tagged no-network runtime smoke test and explicit documentation for real-provider e2e limits.

## Capabilities

### New Capabilities

- `cli-bootstrap-parity`: Defines the CLI/bootstrap contract for acknowledgement, argument parsing, startup failures, port/log overrides, shutdown rendering, and final parity verification.

### Modified Capabilities

- None. The existing `compatibility-contract` already names CLI/bootstrap behavior and shutdown/offline rendering as V1 parity surfaces; this change adds the detailed capability spec without changing the broader parity scope.

## Impact

- Affects `internal/cli`, `cmd/symphony`, runtime startup options, and CLI-focused tests.
- Adds a new OpenSpec capability under `openspec/specs/cli-bootstrap-parity/`.
- Does not change orchestrator ownership, tracker interfaces, provider-specific write behavior, HTTP API payloads, or web dashboard contracts.
