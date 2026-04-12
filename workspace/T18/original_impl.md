# T18 Original Implementation Notes

## Sources Inspected

- `/Users/apple/Documents/Github/symphony/elixir/bin/symphony`
- `/Users/apple/Documents/Github/symphony/elixir/mix.exs`
- `/Users/apple/Documents/Github/symphony/elixir/lib/symphony_elixir/cli.ex`
- `/Users/apple/Documents/Github/symphony/elixir/lib/symphony_elixir.ex`
- `/Users/apple/Documents/Github/symphony/elixir/lib/symphony_elixir/http_server.ex`
- `/Users/apple/Documents/Github/symphony/elixir/lib/symphony_elixir/status_dashboard.ex`
- `/Users/apple/Documents/Github/symphony/elixir/lib/symphony_elixir/config.ex`
- `/Users/apple/Documents/Github/symphony/elixir/lib/symphony_elixir/config/schema.ex`
- `/Users/apple/Documents/Github/symphony/elixir/lib/symphony_elixir/workflow.ex`
- `/Users/apple/Documents/Github/symphony/elixir/test/symphony_elixir/cli_test.exs`
- `/Users/apple/Documents/Github/symphony/elixir/test/symphony_elixir/extensions_test.exs`
- `/Users/apple/Documents/Github/symphony/elixir/test/symphony_elixir/orchestrator_status_test.exs`
- `/Users/apple/Documents/Github/symphony/elixir/test/symphony_elixir/workspace_and_config_test.exs`

## Behavior Summary

The Elixir executable is an escript whose main module is `SymphonyElixir.CLI`.

Startup requires the exact long acknowledgement flag:

```text
--i-understand-that-this-will-be-running-without-the-usual-guardrails
```

Without that flag, `CLI.evaluate/2` returns the red guardrails banner and stops before checking the workflow file, setting the log root, setting the port override, or starting the application.

The CLI accepts:

- optional positional workflow path, defaulting to `./WORKFLOW.md`
- `--logs-root <path>`, expanded before use
- `--port <port>`, where `0` is valid and negative values are rejected

`--port` is applied before app startup and overrides `server.port` in `WORKFLOW.md`. The HTTP server records its live bound port so dashboard links prefer the actual listener port, especially when the configured port is `0`.

`Application.start/2` configures a rotating disk log, then starts workflow store, orchestrator, optional HTTP server, and status dashboard. `Application.stop/1` renders a minimal terminal frame containing `SYMPHONY STATUS` and `app_status=offline`, with no timestamp.

Dashboard URL display normalizes wildcard hosts such as `0.0.0.0`, `::`, and `[::]` to `127.0.0.1`; IPv6 literals are bracketed for display.

Config resolution remains part of the broader user-visible contract: `LINEAR_API_KEY`, `LINEAR_ASSIGNEE`, `$VAR` indirection, and path expansion are handled by the existing config layer rather than the CLI parser.

## Parity Requirements

- Preserve the acknowledgement flag name and banner meaning.
- Require acknowledgement before all startup side effects.
- Keep default workflow path behavior as `WORKFLOW.md` in the current working directory.
- Expand explicit workflow paths and report missing files as startup errors.
- Support `--logs-root` and build the log file under `<logs-root>/log/symphony.log`.
- Support `--port`, including `0`, and make it override workflow `server.port`.
- Surface startup failures on stderr and return a nonzero exit code.
- On normal shutdown, render the offline dashboard frame once.
- Keep dashboard URL display operator-friendly by using the live bound port and loopback display for wildcard bind hosts.
- Keep live Linear e2e credentials optional and document any skipped real-provider checks.
