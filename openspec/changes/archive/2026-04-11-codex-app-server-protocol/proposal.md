## Why

The Go port needs a compatibility-faithful Codex app-server layer so `internal/codex` can drive real protocol sessions instead of pretending the app-server is a thin stdio wrapper. This change captures the protocol surface T09 needs now, including bootstrap, turns, approvals, dynamic tools, and timeout behavior, while keeping provider-specific behavior out of the core runtime.

## What Changes

- Define a Go-native Codex app-server protocol capability for session startup, thread lifecycle, turn execution, and shutdown.
- Validate workspace paths before session launch, including rejecting the workspace root, out-of-root paths, and symlink escapes.
- Carry through protocol handling for terminal events, malformed lines, unknown messages, approvals, non-interactive user input, and tool calls.
- Enforce separate read-timeout and whole-turn timeout behavior.
- Normalize protocol facts into events that orchestration can consume without moving mutable runtime state into `internal/codex`.
- Keep Linear writes, workflow selection, and orchestration policy out of the Codex core.

## Capabilities

### New Capabilities
- `codex-app-server-protocol`: Protocol session lifecycle, workspace validation, thread and turn handling, approval handling, dynamic tool dispatch, timeout enforcement, and normalized protocol events for Symphony-compatible Codex sessions.

### Modified Capabilities

- None

## Impact

- Affects `internal/codex` implementation boundaries, tests, and protocol fixtures.
- Affects orchestration integration points that consume normalized protocol events.
- Introduces a new compatibility spec that other T09/T12 work can build on without re-litigating the app-server contract.
