# T09 Final Compare

## Compared Inputs

- Upstream implementation notes in `workspace/T09/original_impl.md`
- Accepted plan in `workspace/T09/final_impl.md`
- OpenSpec change `codex-app-server-protocol`
- Implemented Go package `internal/codex`
- Review findings in `workspace/T09/code_review.md`

## Parity Result

The Go implementation preserves the T09 compatibility target:

- sessions validate workspace scope before launch;
- app-server startup sends `initialize`, `initialized`, and `thread/start`;
- `thread/start` advertises caller-supplied dynamic tool specs;
- `turn/start` uses the stored thread id, prompt input, cwd, title, approval policy, and sandbox policy;
- terminal turn events complete, fail, or cancel the current turn;
- malformed and unknown messages are surfaced as protocol events without crashing the loop;
- approval requests under literal policy `never` are auto-approved with the expected protocol decisions;
- non-interactive user-input prompts receive deterministic answers;
- dynamic tool calls go through an injected handler;
- unsupported tools return `unsupported_tool_call`;
- read timeouts and turn timeouts are distinct error sentinels;
- Codex emits facts through an event sink and does not own orchestrator state.

## Deliberate Differences

- The Go transport keeps stderr separate from stdout and drains it instead of parsing it as protocol JSON. This follows the upstream language-neutral SPEC and avoids corrupting the JSON stream.
- `internal/codex` exposes a generic `ToolHandler` boundary only. Linear-specific `linear_graphql` execution remains for T12.
- The package does not wire sessions into orchestrator run workers yet. That belongs to T14 after toolbridge and workflow tasks exist.

## Risk Assessment

No unrecorded high-severity risk remains for T09.

The remaining risks are expected sequencing constraints rather than T09 defects:

- e2e remains low-signal for protocol behavior until T14 wires complete runs;
- real Codex app-server compatibility may need additional message variants as integration tests expose them;
- provider-specific dynamic tools are intentionally absent until T12.
