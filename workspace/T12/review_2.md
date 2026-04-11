# T12 Final Implementation Plan Review

Score: 72/100

- Symphony alignment and source fidelity: 19/30
- Go-native simplicity and maintainability: 17/20
- No overdesign / clean boundaries: 16/20
- Implementation clarity and testability: 10/15
- Verification coverage and landing safety: 10/15

Accepted: No

## High-Severity Issues

1. The plan does not fully specify the unsupported-tool payload shape that Symphony already exposes.
   - The plan says unsupported tool names must return a failed result that includes `supportedTools` ([`workspace/T12/final_impl_v1.md`](./final_impl_v1.md#L62-L64), [`workspace/T12/final_impl_v1.md`](./final_impl_v1.md#L175-L177)).
   - But the proposed `internal/codex` change only adds `ToolContentItem` and `ContentItems`, plus the `ToolCall.Arguments` widening ([`workspace/T12/final_impl_v1.md`](./final_impl_v1.md#L144-L149)).
   - The current Go app-server path only writes `ToolResult` back as `{"id":..., "result": ...}` and maps `ErrUnsupportedTool` to `unsupported_tool_call` ([`internal/codex/session.go`](../../internal/codex/session.go#L522-L547)).
   - There is no explicit field or encoding contract for `supportedTools`, so this plan cannot yet reproduce the Elixir unsupported-tool response shape with confidence.

## Medium / Low Issues

1. The scope is narrower than the approved T12 compatibility-shell slot.
   - The approved design places `toolbridge/linear` in the compatibility shell for `linear_graphql`, Linear-specific tracker write behavior needed by the workflow, and workpad/comment semantics ([`docs/plans/2026-04-10-go-symphony-design.md`](../../docs/plans/2026-04-10-go-symphony-design.md#L133-L136)).
   - The task board gives T12 the goal of implementing `linear_graphql` and other Linear-specific write behavior in the compatibility shell only ([`docs/plans/2026-04-10-go-symphony-design-task.md`](../../docs/plans/2026-04-10-go-symphony-design-task.md#L45-L45)).
   - The plan explicitly defers write helpers and workflow selection, and says to expose exactly one tool in this slice ([`workspace/T12/final_impl_v1.md`](./final_impl_v1.md#L78-L99), [`workspace/T12/final_impl_v1.md`](./final_impl_v1.md#L236-L248)).
   - That is a reasonable first cut for the raw GraphQL bridge, but it leaves part of the task-board goal unplanned.

2. `TestBridgeDoesNotRequireCoreTrackerWrites` is too vague as written.
   - A compile-time import guard would be a sharper proof than a package-level runtime test.
   - The intent is right, but the test description should spell out the exact dependency constraint being enforced.

## Decision

Do not accept this plan yet. It is close on the bridge shape and the raw string argument path, but it still needs a concrete `supportedTools` encoding contract before it can claim parity on unknown-tool behavior.
