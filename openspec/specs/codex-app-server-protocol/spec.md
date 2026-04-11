## Purpose

Define the Codex app-server protocol contract for session bootstrap, workspace validation, thread and turn handling, approvals, dynamic tools, timeout classification, and normalized protocol events.

## Requirements

### Requirement: Session bootstrap validates workspace context and starts the protocol thread
The system MUST validate the workspace path before launch and MUST reject the workspace root, paths outside the configured workspace root, and symlink escapes after resolving the real path.

The system MUST start the app-server session by sending `initialize` and then `thread/start`, and `thread/start` MUST advertise the dynamic tool specs supplied by the caller.

The session MUST store the protocol thread identity returned by the app-server and MUST support deterministic close of the transport handle.

#### Scenario: Workspace root is rejected
- **WHEN** the caller passes the configured workspace root as the workspace path
- **THEN** the session refuses to start
- **AND** returns a workspace validation failure

#### Scenario: Symlink escape is rejected
- **WHEN** the caller passes a path that resolves outside the configured workspace root through a symlink
- **THEN** the session refuses to start
- **AND** does not launch the transport

#### Scenario: Bootstrap sends initialize then thread start
- **WHEN** the caller starts a valid session
- **THEN** the session sends `initialize`
- **AND** sends `thread/start` after `initialize`
- **AND** includes the advertised dynamic tool specs in `thread/start`

### Requirement: Turns reuse one session and terminate on terminal protocol events
The system MUST allow a single session to drive multiple turns for the same active item until the caller stops the session or the item is no longer active.

The system MUST send `turn/start` for each turn and MUST continue receiving protocol traffic until it observes a terminal turn event, cancellation, timeout, or explicit failure.

The system MUST treat terminal turn events as the end of the current turn and MUST preserve non-terminal traffic as part of the in-flight turn.

#### Scenario: Terminal event ends the turn
- **WHEN** a turn stream contains a terminal turn event
- **THEN** the turn completes cleanly
- **AND** the session remains reusable for a later turn

#### Scenario: Non-terminal events keep the turn open
- **WHEN** a turn stream contains approvals, tool calls, or other non-terminal traffic
- **THEN** the turn stays open
- **AND** the session keeps reading until a terminal condition arrives

### Requirement: Approval and user input handling is non-interactive
The system MUST answer approval requests according to the configured approval policy.

When `approval_policy` is `never`, the session MUST auto-approve using the protocol decision strings expected by the app-server behavior.

The system MUST handle `item/tool/requestUserInput` non-interactively and MUST return a deterministic answer that does not block the session.

#### Scenario: Never policy auto-approves
- **WHEN** approval policy is `never` and the app-server requests approval
- **THEN** the session returns an automatic approval decision
- **AND** the turn continues without waiting for user interaction

#### Scenario: User input prompt is answered without blocking
- **WHEN** the app-server requests user input during a turn
- **THEN** the session returns a non-interactive response
- **AND** does not require human input to progress the turn

### Requirement: Dynamic tool calls are dispatched through an injected handler
The system MUST dispatch `item/tool/call` requests through an injected tool handler.

The system MUST keep the dispatch boundary generic so compatibility-shell code can provide provider-specific behavior later.

The system MUST preserve object-shaped and raw string tool arguments when forwarding a tool call to the injected handler.

The system MUST allow handler results to include top-level `contentItems` inside the JSON-RPC `result` object so compatibility-shell tools can return Symphony-style dynamic-tool output.

When the session receives a tool call for an unsupported tool, it MUST return a structured failure result instead of stalling the turn.

#### Scenario: Tool call is dispatched to handler
- **WHEN** the app-server issues an `item/tool/call`
- **THEN** the session forwards the call to the injected tool handler
- **AND** returns the handler result to the protocol stream

#### Scenario: Raw string arguments are preserved
- **WHEN** the app-server issues an `item/tool/call` whose `params.arguments` value is a raw string
- **THEN** the injected handler receives that raw string value
- **AND** the session does not replace it with an empty object

#### Scenario: Tool content items are top-level within the tool result
- **WHEN** the injected handler returns a tool result with content items
- **THEN** the session writes a JSON-RPC response whose `result.contentItems` contains those content items
- **AND** the content items are not nested under `result.result.contentItems`

#### Scenario: Unsupported tool returns failure
- **WHEN** the app-server requests a tool that the injected handler does not support
- **THEN** the session returns a structured failure result
- **AND** the turn does not hang waiting for an implementation that does not exist

### Requirement: Read timeout and turn timeout are distinguishable
The system MUST enforce a read timeout around request/response boundaries for `initialize`, `thread/start`, and `turn/start`.

The system MUST enforce a separate whole-turn timeout around the full streamed turn execution.

Timeout failures MUST be distinguishable from non-zero exits, malformed input, and explicit protocol failures.

#### Scenario: Initialization read timeout is distinct
- **WHEN** the app-server does not answer `initialize` before the read timeout
- **THEN** the session returns a read-timeout failure
- **AND** does not report the failure as a turn timeout

#### Scenario: Turn timeout is distinct
- **WHEN** the streamed turn exceeds the configured turn timeout
- **THEN** the session returns a turn-timeout failure
- **AND** the failure is distinguishable from a request/response timeout

### Requirement: Malformed and unknown input are normalized protocol events
The system MUST treat malformed protocol lines and unknown messages as normalized protocol events rather than immediate session death.

The system MUST publish protocol facts such as session start, turn completion, turn failure, turn cancellation, tool call lifecycle, approval answers, malformed messages, and unknown messages through an event sink.

The system MUST NOT mutate orchestrator runtime state directly.

#### Scenario: Malformed line does not crash the loop
- **WHEN** the transport yields a malformed JSON line
- **THEN** the session emits a malformed-message event
- **AND** continues the receive loop until a terminal condition occurs

#### Scenario: Unknown message is surfaced
- **WHEN** the transport yields a message type the session does not recognize
- **THEN** the session emits an unknown-message event
- **AND** keeps the session alive long enough for the caller to decide the next step
