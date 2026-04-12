# T18 Final Comparison

## Implemented

- Guardrails acknowledgement is mandatory before startup side effects. Missing acknowledgement returns the warning banner and does not validate workflow files, configure logs, override ports, or start the runtime.
- CLI parsing accepts the default `WORKFLOW.md`, one explicit workflow path, order-agnostic `--logs-root` and `--port`, and rejects unknown flags, missing values, flag-looking option values, invalid ports, and extra positional paths.
- `--logs-root` expands to `<logs-root>/log/symphony.log` and restores logger state in tests.
- `--port` overrides configured `server.port`, including `0` for ephemeral listeners.
- Startup errors are operator-facing and do not render the offline dashboard frame.
- Normal shutdown closes the runtime and renders `dashboard.RenderOffline()` exactly once.
- Runtime server wiring is covered through reachable `/`, `/api/v1/state`, and listener shutdown checks in an e2e-tagged no-network smoke test.

## Accepted Limits

- Real Linear/Codex live e2e was not run because this environment has no explicit live-test opt-in or dedicated test credentials. The limitation is recorded in `workspace/T18/todo.md`.
- The log helper preserves the compatibility-relevant path behavior but does not add Elixir-style rotating log handlers in V1.

## Verdict

T18 matches the approved `cli-full-parity-test-sweep` artifacts and closes the full parity-oriented verification matrix available in this environment.
