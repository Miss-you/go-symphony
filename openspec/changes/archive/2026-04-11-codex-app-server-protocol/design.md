## Context

T09 implements the Codex app-server layer inside `internal/codex`, but the Go port is not allowed to turn that package into a scheduler, a runner, or a provider-specific write layer. The design needs to preserve the observed app-server flow from Symphony: validate workspace context, start transport, initialize, start a thread with dynamic tools, start turns, stream newline-delimited protocol traffic, and close deterministically.

The main compatibility risks are protocol drift, timeout ambiguity, and accidental leakage of mutable runtime state into the wrong package boundary.

## Goals / Non-Goals

**Goals:**
- Match Symphony's app-server session shape closely enough for compatibility.
- Keep `internal/codex` narrow, testable, and provider-neutral.
- Enforce clear workspace validation rules before launching a session.
- Distinguish read timeouts from whole-turn timeouts.
- Normalize protocol facts into events without making `internal/codex` own orchestration state.

**Non-Goals:**
- No Linear GraphQL writes or provider-specific tool execution in this layer.
- No workflow selection, runner refactor, or orchestration policy changes.
- No universal tracker write API or Lark-specific runtime behavior.
- No attempt to model every possible Codex protocol message in T09.

## Decisions

1. **Session owns the protocol lifecycle.** `Session` will own the transport handle, validated workspace context, thread identity, approval policy, sandbox policy, dynamic tool specs, and timeout settings. This keeps bootstrap, turn execution, and close behavior in one place. An alternative would be to split bootstrap and turn execution across multiple packages, but that would spread protocol state and make shutdown semantics harder to reason about.

2. **Use a transport abstraction plus a scripted test transport.** Real sessions will start through a transport factory, while package tests will use an in-memory scripted transport that records outbound messages and feeds inbound protocol lines. This keeps protocol transcripts deterministic in tests and avoids baking process-launch assumptions into the session logic. A pure mock at the handler boundary would miss the request/response timing and newline-delimited parsing behavior T09 needs.

3. **Validate workspace paths before launch with real-path checks.** The session will reject the workspace root, out-of-root paths, and symlink escapes after resolving the real path. This matches the source-faithful requirement that the app-server must not start in ambiguous or escaped locations. An alternative would be to trust caller-supplied paths, but that would move the safety boundary out of the protocol layer and make later failures harder to classify.

4. **Treat malformed and unknown input as protocol facts, not fatal transport failures.** The receive loop will surface malformed lines and unknown messages as normalized events while continuing until a terminal condition, timeout, cancellation, or explicit failure occurs. This preserves observability and avoids conflating parser noise with session death. The alternative, aborting on the first unknown line, would make compatibility brittle and would hide recovery opportunities from the orchestrator.

5. **Keep approval and tool handling behind generic handler interfaces.** Approval decisions and non-interactive user-input responses will use an approval handler, while dynamic tool calls will use a tool handler that returns structured results. Unsupported tools will return structured failures instead of stalling the turn. This keeps `internal/codex` generic enough for T12 to inject Linear behavior later without changing the protocol core.

6. **Separate read timeout from turn timeout.** Request/response boundaries for `initialize`, `thread/start`, and `turn/start` will use a read timeout, while the full streamed turn will use a turn timeout. This gives callers a stable error class for transient protocol stalls versus overall turn overruns. Collapsing both into one timeout would blur recovery behavior and make retry policy less precise.

7. **Emit normalized events through an event sink.** The session will publish protocol facts such as session start, turn completion, tool call lifecycle, approval answers, malformed messages, and unknown messages through an event sink. `internal/codex` will not mutate orchestrator state directly. The alternative would be to let the session own more runtime state, which would violate the intended package boundary.

## Risks / Trade-offs

- [Risk] Protocol enums or message shapes may diverge from the current Symphony app-server contract. → Keep the spec narrow, add transcript-based tests, and prefer explicit normalized failure categories over implicit string matching.
- [Risk] Workspace validation may reject edge-case paths that existing callers relied on. → Make the failure explicit and keep the validation logic in one place so later compatibility adjustments are local.
- [Risk] The transport abstraction could become too generic and hide protocol-specific behavior. → Keep the abstraction minimal: read newline-delimited messages, write JSON, and close cleanly.
- [Risk] Event normalization might duplicate information already present in protocol messages. → Accept some duplication to preserve observability at the orchestration boundary.

## Migration Plan

1. Land the change spec and tests for the protocol contract.
2. Implement the session and transport harness in `internal/codex`.
3. Wire orchestration to consume normalized events without moving state ownership.
4. Keep later Linear tool and workflow work in their compatibility-shell tasks.

Rollback is simple: stop at the protocol boundary and leave later tasks unimplemented if any compatibility issue appears.

## Open Questions

- Exact wire-format details for any currently unmodeled Codex protocol messages should stay out of T09 unless tests require them.
- Unsupported-tool failure payloads should remain structured enough for orchestration, but the final shape can stay implementation-specific as long as the spec-level behavior is preserved.
