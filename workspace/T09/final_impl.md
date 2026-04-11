# T09 Final Implementation: Codex App-Server Protocol

## Review Gate

`final_impl_v1.md` passed rubric review without a correction round:

- `review_1.md`: 88 / 100, no high-severity issues
- `review_2.md`: 94 / 100, no high-severity issues
- average: 91 / 100

Two source-faithfulness notes were accepted into this final plan: workspace validation must explicitly reject out-of-root paths and symlink escapes, and `thread/start` must advertise dynamic tool specs.

## Goal

Implement `internal/codex` as a small Go protocol engine that matches Symphony's app-server behavior closely enough for compatibility, without turning it into a runner, scheduler, or provider-specific tool layer.

T09 covers:

- session bootstrap
- thread and turn lifecycle
- protocol event normalization
- approval handling
- dynamic tool dispatch boundary
- read and turn timeout handling
- deterministic shutdown

T09 does not cover Linear writes, workflow selection, orchestrator state ownership, or runner/SSH concerns.

## Source-Faithful Shape

Symphony's Codex integration is not a thin stdio wrapper. The app-server session is:

```text
workspace path
  -> validate workspace context
  -> start transport
  -> initialize
  -> thread/start with dynamic tool specs
  -> turn/start
  -> receive newline-delimited JSON
      -> terminal turn events
      -> approvals
      -> tool calls
      -> malformed / unknown messages
  -> close transport
```

The Go version should preserve that shape. One session owns one subprocess and one protocol thread, and a single session can drive multiple turns for the same active item until the caller stops or the item is no longer active.

## Package Boundaries

| Package | Owns | Does not own |
| --- | --- | --- |
| `internal/codex` | Protocol session lifecycle, transport, parser, turn execution, approval handling, tool dispatch, timeout enforcement. | Orchestration policy, provider writes, workspace naming, host selection. |
| `internal/orchestrator` | Mutable runtime state and scheduling. | Codex protocol details. |
| `internal/runner` | One-shot local/SSH execution host behavior. | Long-lived app-server protocol state. |
| `internal/toolbridge/linear` | Linear-specific dynamic tool behavior in T12. | Core Codex protocol logic. |

## Proposed `internal/codex` API Surface

Keep the surface narrow and testable:

- `Session` owns the transport handle, thread id, workspace context, approval policy, sandbox policies, dynamic tool specs, and timeout settings.
- `Transport` reads newline-delimited protocol messages, writes JSON protocol messages, and closes the process.
- `TransportFactory` starts the app-server process for a validated workspace.
- `ApprovalHandler` makes protocol approval decisions and non-interactive prompt answers.
- `ToolHandler` executes dynamic tool calls.
- `EventSink` receives normalized protocol facts that can become `domain.RunEvent` values at the orchestration boundary.
- `TurnResult` reports completed, failed, cancelled, or timed-out turns.

The package may know Codex protocol concepts like approvals and dynamic tools. It must not know Linear data models, Linear GraphQL semantics, or workflow bundle details.

## Behavior To Preserve

- Validate the workspace path before launch.
- Reject the workspace root itself.
- Reject paths outside the configured workspace root.
- Reject symlink escapes after resolving the real path.
- Send `initialize`, then `thread/start`, then `turn/start`.
- Include dynamic tool specs in `thread/start`; T09 provides the protocol boundary and T12 provides Linear-specific tool execution.
- Pass through settings already parsed by `internal/config`: command, approval policy, thread sandbox, turn sandbox policy, turn timeout, and read timeout.
- Treat `approval_policy == "never"` as automatic approval using protocol decision strings.
- Handle `item/tool/requestUserInput` non-interactively.
- Dispatch `item/tool/call` through an injected handler.
- Return structured failures for unsupported tools instead of stalling.
- Distinguish request/response read timeouts from whole-turn timeouts.
- Treat malformed or unknown lines as protocol events, not immediate session death.
- Leave stall recovery in `internal/orchestrator`, which already owns liveness and retry scheduling.

## Implementation Increments

### 1. Lock The Protocol Model

Define the smallest request, response, event, and result types needed for Symphony's current app-server flow.

Model only:

- `initialize`
- `thread/start`, including dynamic tool advertisement
- `turn/start`
- terminal turn events
- approval requests
- user-input tool prompts
- dynamic tool calls
- malformed and unknown input

### 2. Build A Scripted Transport Test Harness

Add an in-memory transport for package tests. It should record outgoing JSON, feed scripted inbound lines, simulate slow or missing responses, and close cleanly.

Use this harness to define the expected protocol transcript before connecting any real process launch.

### 3. Implement Session Bootstrap

Implement:

- workspace validation that rejects the workspace root, out-of-root paths, and symlink escapes
- transport creation
- `initialize` request
- `thread/start` request with dynamic tool specs
- thread/session identity storage
- deterministic close

This proves the session starts correctly without requiring a real Codex process in unit tests.

### 4. Implement Turn Execution

Implement:

- `turn/start`
- turn-level receive loop
- terminal turn detection
- normalized event emission
- stable session reuse across turns

The loop should continue through non-terminal protocol traffic until it sees a terminal result or the caller stops the session.

### 5. Implement Approval And Tool Handling

Add first-class handling for:

- approvals under `never`
- non-interactive user input prompts
- dynamic tool calls through `ToolHandler`
- unsupported tool failure results

Keep the handler boundary generic so T12 can inject Linear behavior later without changing the Codex core.

### 6. Add Timeout And Error Classification

Enforce:

- read timeout around `initialize`, `thread/start`, and `turn/start` request/response boundaries
- turn timeout around the full streamed turn
- clean close on timeout or cancellation

Return clear failure categories so callers do not need to infer timeout class from string matching.

### 7. Normalize Events For Orchestration

Emit protocol facts that can become `domain.RunEvent` values at the orchestration boundary:

- session/thread started
- turn completed
- turn failed
- turn cancelled
- tool call received/completed/failed
- approval answered
- malformed message received
- unknown message received

Do not let `internal/codex` mutate orchestrator state directly.

## Non-Goals

Do not introduce in T09:

- a universal tracker write API
- Linear-specific tool execution
- workflow selection
- a second runtime state store
- runner refactors beyond what the Codex package needs to compile and start its transport

## Verification Strategy

Primary package gate:

```bash
go test ./internal/codex/...
```

Required package coverage:

- bootstrap sends `initialize` and `thread/start`
- `thread/start` advertises dynamic tool specs
- workspace validation rejects root, out-of-root paths, and symlink escapes
- the session stores thread identity
- `turn/start` carries approval and sandbox policy data
- terminal turn events end the turn cleanly
- malformed lines do not crash the loop
- unknown messages are emitted as protocol events
- approval requests are answered under `never`
- user-input prompts receive non-interactive answers
- tool calls dispatch through the injected handler
- unsupported tools return structured failures
- read timeout and turn timeout are distinguishable

Broader gates:

```bash
go test ./...
make build
make lint
make test-e2e
```

`make test-e2e` is expected to be low-signal until T14 wires complete runs, but the command should still pass or the limitation must be recorded in `workspace/T09/todo.md`.

## Acceptance Bar

T09 is done when `internal/codex` can:

- start an app-server session through a streaming transport boundary;
- drive a turn with the current Symphony protocol shape;
- handle approvals, tool calls, malformed traffic, and unsupported tools;
- enforce configured read and turn timeouts;
- report normalized protocol facts without owning runtime scheduling state.

That keeps the package faithful to Symphony's current behavior while staying narrow enough for the rest of the Go port to build on it cleanly.
