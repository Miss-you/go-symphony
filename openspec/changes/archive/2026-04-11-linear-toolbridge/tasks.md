## 1. Codex Protocol Boundary

- [x] 1.1 Add generic `ToolContentItem` support to `codex.ToolResult` without adding Linear-specific logic to `internal/codex`.
- [x] 1.2 Widen `codex.ToolCall.Arguments` to preserve raw string arguments and object arguments from `item/tool/call`.
- [x] 1.3 Add Codex session tests that prove raw string arguments reach the handler and tool `contentItems` are written as `response.result.contentItems`, not `response.result.result.contentItems`.

## 2. Linear ToolBridge Package

- [x] 2.1 Create `internal/toolbridge/linear` with a narrow GraphQL client interface, bridge constructor, and `codex.ToolHandler` implementation.
- [x] 2.2 Implement `ToolSpecs()` for exactly one dynamic tool, `linear_graphql`, with the Symphony-compatible schema.
- [x] 2.3 Implement `linear_graphql` argument normalization, GraphQL dispatch, pretty JSON content-item output, GraphQL-error detection, and Symphony-compatible failure messages.
- [x] 2.4 Implement bridge-local unknown-tool failure payloads with `supportedTools: ["linear_graphql"]`.

## 3. Linear Write Helpers

- [x] 3.1 Implement `CreateComment(ctx, issueID, body)` using the old `commentCreate` mutation and failure mapping.
- [x] 3.2 Implement `UpdateIssueState(ctx, issueID, stateName)` with state-id lookup followed by `issueUpdate` and provider-specific failure mapping.
- [x] 3.3 Add tests proving write helper mutation sequencing, success checks, and helper-specific errors.

## 4. Boundary And Verification

- [x] 4.1 Add a dependency-boundary guard proving `internal/toolbridge/linear` does not depend on `internal/tracker`, `internal/domain`, or `internal/orchestrator`.
- [x] 4.2 Run package gates: `go test ./internal/codex/... ./internal/toolbridge/...`.
- [x] 4.3 Run broad gates: `go test ./...`, `make build`, `make lint`, `make test-e2e`, and `openspec validate linear-toolbridge`.
