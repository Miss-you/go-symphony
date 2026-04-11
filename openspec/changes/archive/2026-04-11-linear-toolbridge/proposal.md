## Why

`go-symphony` needs the Linear ToolBridge surface that Symphony already exposes, but the current Go code only covers the read side and the Codex boundary is still too narrow to carry Symphony-style dynamic tool results and raw string tool arguments. This change closes that gap without widening the provider-neutral core.

## What Changes

- Add a compatibility-shell Linear ToolBridge that exposes one dynamic Codex tool, `linear_graphql`.
- Preserve raw GraphQL passthrough behavior, including trimmed query handling, JSON-object variables, and stable failure reporting.
- Add provider-specific Linear write helpers in the compatibility shell for comment creation and issue state updates.
- Extend the generic Codex protocol boundary so dynamic tool calls can preserve raw string arguments and top-level `contentItems` tool results.
- Keep tracker/core write interfaces unchanged.
- Keep Linear write behavior out of `internal/tracker`, `internal/domain`, and `internal/orchestrator`.

## Capabilities

### New Capabilities
- `linear-toolbridge`: Compatibility-shell Linear ToolBridge behavior, including `linear_graphql`, dynamic-tool result shaping, and Linear-specific write helpers outside core.

### Modified Capabilities
- `codex-app-server-protocol`: Dynamic tool calls must preserve raw string arguments, and tool results must carry top-level `contentItems` in the protocol shape expected by the compatibility shell.
- `compatibility-contract`: The V1 parity contract must explicitly include the Linear ToolBridge surface and the core-versus-compatibility-shell boundary for provider-specific write behavior.

## Impact

Affected code includes `internal/codex` and the new `internal/toolbridge/linear` compatibility-shell package. The bridge may reuse the existing Linear GraphQL HTTP client shape, but this change does not modify the Linear reader adapter or expand tracker reads. This change also affects OpenSpec parity documentation for the Codex app-server protocol and the compatibility contract, but it does not implement workflow/assembly injection, expand tracker reads, or introduce a universal write API.
