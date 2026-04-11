# T12 Final Comparison

## Result

T12 is aligned with the accepted task goal: `linear_graphql` and Linear-specific write helpers live in the compatibility shell, while the core tracker/domain/orchestrator boundaries remain provider-neutral and write-free.

## Symphony Parity Check

- `linear_graphql` remains the only advertised Codex dynamic tool, matching `SymphonyElixir.Codex.DynamicTool.tool_specs/0`.
- Tool arguments preserve raw string queries and object-shaped `{query, variables}` input. Blank queries, invalid argument types, and invalid variables return the Symphony-compatible messages.
- Tool output uses top-level `contentItems` inside the JSON-RPC `result` object, matching Elixir app-server framing.
- Unknown tool names return a failed dynamic-tool payload with `supportedTools: ["linear_graphql"]`.
- GraphQL responses with non-empty `errors` are marked unsuccessful while preserving the response body.
- Linear comment creation and issue state updates exist as provider-specific bridge helpers, mirroring the old Linear adapter behavior without adding a universal tracker write API.

## Boundary Check

- `internal/codex` only gained generic protocol shape: raw tool arguments, `inputSchema`, and content-item tool results.
- `internal/toolbridge/linear` does not import `internal/tracker`, `internal/domain`, or `internal/orchestrator`.
- `internal/tracker.TrackerReader` remains read-only.
- No Linear write logic moved into orchestrator or domain.

## Verification Evidence

- `go test ./internal/codex/... ./internal/toolbridge/...`
- `go test ./...`
- `make build`
- `make lint`
- `make test`
- `make test-e2e`
- `make verify`
- `openspec validate linear-toolbridge`

## Remaining Notes

Residual non-blocking risks are recorded in `workspace/T12/todo.md`.
