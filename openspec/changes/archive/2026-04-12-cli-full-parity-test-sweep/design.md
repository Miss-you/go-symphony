## Context

`cmd/symphony` already delegates to `internal/cli`, and `internal/cli.StartRuntime` already assembles the runtime, tracker, workspace manager, orchestrator, Codex worker path, HTTP API, and web dashboard. The remaining gap is the process host contract: the current CLI accepts only a positional workflow path and does not implement Symphony's acknowledgement gate, `--logs-root`, `--port`, or shutdown/offline rendering.

The approved design requires user-visible compatibility first. T18 should finish the executable host without widening core runtime abstractions or moving provider-specific behavior into neutral packages.

## Goals / Non-Goals

**Goals:**

- Preserve Symphony-compatible CLI argument semantics.
- Prevent startup side effects until the guardrails acknowledgement flag is present.
- Apply CLI `--port` precedence over workflow `server.port`.
- Configure process logging under the selected logs root.
- Render the minimal offline dashboard frame on normal shutdown.
- Add focused CLI tests plus a no-network e2e-tagged runtime smoke test.

**Non-Goals:**

- Do not change orchestrator scheduling, tracker interfaces, or provider write behavior.
- Do not change HTTP API or web dashboard payload contracts.
- Do not require real Linear/Codex credentials for default verification.
- Do not add a broad logging subsystem or log rotation framework.
- Do not create a second dashboard or runtime state owner.

## Decisions

### Keep `cmd/symphony` thin

`cmd/symphony/main.go` should continue to own only signal-aware context setup, argument forwarding, and `os.Exit`. All parsing and host lifecycle behavior belongs in `internal/cli` so tests can exercise it without running the actual process.

Alternative considered: parse flags in `cmd/symphony`. Rejected because it would make process behavior harder to test and would duplicate `internal/cli` seams.

### Use a small order-agnostic parser

The CLI parser should accept flags before or after the optional workflow path, reject unknown flags, and use the last value for repeated `--logs-root` and `--port`. The standard `flag` package stops at the first positional argument, so a small parser is clearer and more faithful to Elixir `OptionParser`.

Alternative considered: use `flag.FlagSet`. Rejected because it would narrow accepted argument ordering unless additional preprocessing were added.

### Model CLI port precedence as a runtime option

Add `RuntimeOptions.ServerPortOverride *int`. `StartRuntime` applies it to the loaded settings before HTTP server startup. This keeps CLI concerns out of `config.ParseSettings` while giving tests a narrow runtime seam.

Alternative considered: mutate the workflow file or config store. Rejected because CLI overrides are process-local and should not alter repo-owned workflow content.

### Configure logs through a restorable helper

Add `DefaultLogFile(root string)` and a helper that opens `<root>/log/symphony.log`, redirects the standard logger, and returns a restore/close function. Tests must restore the previous logger writer to avoid global state leakage.

Alternative considered: implement rotating logs. Rejected for V1 because the task requires `--logs-root` behavior, not a full logging subsystem.

### Render offline through the existing dashboard renderer

Use `dashboard.RenderOffline()` for shutdown output. That reuses the already-tested T16 renderer and avoids a second terminal frame implementation.

Alternative considered: print a hand-written shutdown string. Rejected because it could drift from the terminal dashboard compatibility contract.

## Risks / Trade-offs

- [Risk] Process-wide logger redirection leaks between tests. -> Mitigation: helper returns a restore function and tests assert restoration behavior.
- [Risk] CLI parsing narrows Elixir-compatible ordering. -> Mitigation: parser tests cover flags before and after the workflow path.
- [Risk] `--port 0` reports the configured value instead of the live bound port. -> Mitigation: runtime tests assert `DashboardURL()` uses the listener's actual port.
- [Risk] Real Linear/Codex live e2e is unavailable in default environments. -> Mitigation: add a no-network e2e-tagged smoke test and record real-provider e2e as skipped unless credentials are provided.
- [Risk] Live terminal rendering could become another state machine. -> Mitigation: any CLI terminal loop must read only `Runtime.Snapshot()` and use `observability.Projector` plus `dashboard.RenderGate`.
