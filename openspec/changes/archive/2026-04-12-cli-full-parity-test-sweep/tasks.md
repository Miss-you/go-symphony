## 1. CLI Contract Tests

- [x] 1.1 Add TDD tests for acknowledgement gating, usage errors, default and explicit workflow path handling, and side-effect ordering.
- [x] 1.2 Add TDD tests for order-agnostic `--logs-root` and `--port` parsing, including repeated values and invalid inputs.
- [x] 1.3 Add TDD tests for startup error text, normal shutdown offline rendering, startup-failure no-offline behavior, and runtime close failure handling.

## 2. CLI Implementation

- [x] 2.1 Implement the order-agnostic CLI parser and guardrails acknowledgement banner in `internal/cli`.
- [x] 2.2 Implement log file path/configuration helpers with test-safe logger restoration.
- [x] 2.3 Wire `cli.Main` through the parser, workflow existence check, log setup, runtime startup, context wait, shutdown close, and offline rendering.

## 3. Runtime Port Override

- [x] 3.1 Add `RuntimeOptions.ServerPortOverride` and apply it before HTTP server startup.
- [x] 3.2 Add runtime tests proving CLI port precedence, ephemeral listener binding, reachable dashboard/API routes, and operator-friendly dashboard URL display.

## 4. E2E and Verification Artifacts

- [x] 4.1 Add an e2e-tagged no-network runtime smoke test for memory provider dashboard/API startup and shutdown.
- [x] 4.2 Write `workspace/T18/test_strategy.md` mapping each verification command to the behavior it proves.
- [x] 4.3 Write `workspace/T18/todo.md` with live-provider e2e limits and any accepted residual risks.
- [x] 4.4 Run targeted gates, broad gates, OpenSpec validation, and diff checks, then record verification evidence.
