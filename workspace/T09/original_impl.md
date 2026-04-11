# T09 Original Implementation: Symphony Codex App-Server

Source repo inspected: `/Users/apple/Documents/Github/symphony`.

The path named in the delivery workflow, `/Users/lihui/Documents/GitHub/symphony`, does not exist on this machine. The adjacent local upstream repo at `/Users/apple/Documents/Github/symphony` was used instead.

## Summary

Upstream Symphony treats Codex as an app-server protocol client, not as a thin command wrapper. `SymphonyElixir.Codex.AppServer` starts a long-lived subprocess, bootstraps a protocol session, starts a thread, drives one turn at a time, handles server requests during the turn, and closes the process explicitly.

The important compatibility target is:

```text
workspace cwd
  -> validate cwd
  -> spawn bash -lc <codex command>
  -> initialize
  -> thread/start with dynamic tool specs
  -> turn/start with prompt, approval policy, and sandbox policy
  -> newline JSON receive loop
     -> terminal turn events
     -> approvals
     -> dynamic tool calls
     -> user-input tool prompts
     -> malformed/unknown messages
  -> close session
```

## Session Lifecycle

- `AppServer.run/4` wraps one session and always closes the port in an `after` block.
- `start_session/1` validates the workspace, starts the subprocess, sends `initialize`, sends `thread/start`, and stores session/thread state.
- `start_turn/7` reuses the thread, sends the prompt plus `approvalPolicy` and `sandboxPolicy`, and receives a `turn_id`.
- `AgentRunner.run_codex_turns/4` starts one app-server session per issue and reuses it across continuation turns until the issue is no longer active or `max_turns` is reached.

Evidence:

- `/Users/apple/Documents/Github/symphony/elixir/lib/symphony_elixir/codex/app_server.ex`
- `/Users/apple/Documents/Github/symphony/elixir/lib/symphony_elixir/agent_runner.ex`

## Protocol Events

The receive loop consumes newline-delimited JSON until a terminal turn event or timeout.

- `turn/completed`, `turn/failed`, and `turn/cancelled` finish the turn.
- Unknown JSON messages are emitted as non-terminal protocol events.
- Malformed non-JSON lines are emitted as malformed messages and do not immediately fail the turn.
- Event emission attaches timestamps and metadata such as process identity and usage payloads.

## Approvals And Tool Calls

The app-server can ask the client to decide or execute work during a turn.

- Approval-like requests include command execution approval, exec approval, patch approval, and file-change approval.
- With `approval_policy == "never"`, Symphony auto-approves using the protocol decision strings expected by the app-server.
- `item/tool/requestUserInput` is handled non-interactively. Recognizable approval choices get approval answers; otherwise the client returns a fallback that operator input is unavailable.
- `item/tool/call` is executed client-side. Symphony dispatches the named dynamic tool and returns a structured result to the app-server.
- The built-in dynamic tool is `linear_graphql`, advertised during `thread/start` through `dynamicTools`.
- Unsupported tools fail with a structured error result instead of hanging the turn.

Provider-specific tool behavior remains outside the neutral runtime shape. The Codex client owns protocol dispatch and callback wiring, while Linear-specific execution belongs to the compatibility shell.

## Timeout Handling

Symphony keeps separate timeout responsibilities:

- `read_timeout_ms` applies to individual protocol request/response waits during `initialize`, `thread/start`, and `turn/start`.
- `turn_timeout_ms` applies to the whole post-start turn stream.
- `stall_timeout_ms` is enforced by orchestrator-level liveness recovery, not by the app-server client itself.

Default upstream values are `read_timeout_ms: 5000`, `turn_timeout_ms: 3600000`, and `stall_timeout_ms: 300000`.

## Workspace Guard

The app-server client validates the session cwd before launch. It rejects the workspace root itself and rejects paths outside the configured workspace root, including symlink escapes. The process runs with cwd set to the issue workspace.

## Go Implementation Implications

- `internal/codex` needs a stateful session object with process handle, thread id, policy state, and workspace context.
- The protocol parser must distinguish startup responses, turn terminal events, non-terminal notifications, approval requests, tool calls, and malformed lines.
- Approvals and user input prompts must be first-class protocol interactions.
- Dynamic tools should be injected through an interface so T12 can provide `linear_graphql` without putting Linear writes in core Codex code.
- Request read deadlines and turn deadlines belong in `internal/codex`; stall recovery remains orchestrator-owned.
- Codex should emit normalized events for orchestrator projection rather than owning global run state.
