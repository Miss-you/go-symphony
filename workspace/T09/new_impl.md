# T09 New Implementation: Current Go State

## Current State

- `internal/codex` is a stub package with only `doc.go`.
- `internal/domain` already exposes Codex-shaped runtime projection fields: `ActiveRun.SessionID`, `TurnCount`, `CodexTotals`, `RateLimits`, and the stable `RunEventKind` vocabulary.
- `internal/orchestrator` already owns mutable runtime state and consumes `domain.RunEvent` values. It updates session id, turn counts, token totals, rate limits, completion, failure, and retry metadata from worker events.
- `internal/config` already parses Codex settings: command, approval policy, thread sandbox, turn sandbox policy, turn timeout, read timeout, and stall timeout.
- `internal/runner` currently provides one-shot command execution for local and SSH hosts. It is useful for terminal commands, but it does not model a long-lived streaming app-server process.

## Relevant Boundaries

- `internal/codex` belongs in the core runtime because the app-server protocol is a provider-neutral compatibility target.
- `internal/toolbridge/linear` does not exist yet and should own Linear-specific write behavior in T12.
- `internal/orchestrator` must remain the single owner of mutable runtime state. Codex can report facts, but must not schedule retries or mutate snapshots directly.
- `internal/runner` should keep command execution and host concerns. T09 can introduce a streaming process boundary in `internal/codex` without changing runner's one-shot contract.

## Gaps To Fill

- No Codex session lifecycle API.
- No app-server transport abstraction for newline-delimited JSON.
- No protocol message/request/response model.
- No read-timeout or turn-timeout behavior.
- No approval or user-input handling.
- No dynamic tool callback boundary.
- No tests for bootstrap, event normalization, approvals, tool calls, malformed messages, or timeout handling.

## Recommended Go Shape

Build `internal/codex` as a small protocol engine:

```text
Session
  -> validates workspace context supplied by caller
  -> starts streaming transport
  -> initialize
  -> thread/start
  -> turn/start
  -> receive loop
     -> normalize messages into Events
     -> answer approvals
     -> execute dynamic tools
     -> return terminal TurnResult
  -> close transport
```

Use interfaces for edges:

- `Transport` for reading/writing JSON protocol messages and closing the process.
- `TransportFactory` for starting the app-server process.
- `ToolHandler` for dynamic tools, so T12 can inject Linear behavior later.
- `ApprovalHandler` for policy decisions and non-interactive prompts.
- `EventSink` callback or channel for normalized Codex events that can become `domain.RunEvent`s at the orchestration boundary.

## Testing Direction

The package can be tested with an in-memory scripted transport, avoiding a real Codex process. Package tests should prove:

- startup sends `initialize` and `thread/start` and stores the thread id;
- `RunTurn` sends `turn/start` with approval and sandbox fields;
- terminal events return completed/failed/cancelled results;
- non-terminal messages are emitted and malformed lines do not crash the loop;
- approval requests are answered under `approval_policy == "never"`;
- dynamic tool calls invoke the callback and write the tool result;
- unsupported tools produce structured failures;
- read and turn timeouts are distinguishable.
