## 1. Protocol Model and Test Harness

- [x] 1.1 Define the minimal Codex app-server protocol request, response, event, and result types needed for T09.
- [x] 1.2 Add a scripted in-memory transport that records outbound JSON, feeds inbound newline-delimited protocol lines, and simulates slow or missing responses.
- [x] 1.3 Add transcript tests that lock the expected bootstrap and turn flow before any real process launch is wired in.

## 2. Session Bootstrap and Workspace Validation

- [x] 2.1 Implement workspace validation that rejects the workspace root, out-of-root paths, and symlink escapes after resolving the real path.
- [x] 2.2 Implement transport factory startup for a validated workspace and deterministic session close.
- [x] 2.3 Implement `initialize` followed by `thread/start` with dynamic tool specs and stored thread identity.

## 3. Turn Lifecycle and Event Normalization

- [x] 3.1 Implement `turn/start` and the receive loop that reuses one session across multiple turns.
- [x] 3.2 Emit normalized protocol events for session start, turn completion, turn failure, turn cancellation, tool call lifecycle, approval answers, malformed messages, and unknown messages.
- [x] 3.3 Keep orchestration state ownership outside `internal/codex` by routing facts through an event sink only.

## 4. Approval Handling and Dynamic Tool Dispatch

- [x] 4.1 Implement approval handling for the configured policy, including automatic approval when the policy is `never`.
- [x] 4.2 Implement non-interactive handling for `item/tool/requestUserInput`.
- [x] 4.3 Dispatch `item/tool/call` through an injected handler and return structured failures for unsupported tools.

## 5. Timeout and Error Classification

- [x] 5.1 Enforce read timeouts around `initialize`, `thread/start`, and `turn/start` request/response boundaries.
- [x] 5.2 Enforce a separate whole-turn timeout for streamed turns.
- [x] 5.3 Classify read timeouts, turn timeouts, malformed input, and explicit protocol failures into distinct result paths.

## 6. Verification

- [x] 6.1 Add package-scoped tests covering bootstrap, workspace validation, turn reuse, approvals, user input, tool dispatch, malformed input, and timeout behavior.
- [x] 6.2 Run `go test ./internal/codex/...` and fix any failures until the package gate passes.
- [x] 6.3 Run `openspec validate codex-app-server-protocol` and confirm the change is apply-ready.
