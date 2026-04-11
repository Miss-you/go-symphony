# T12 Test Strategy

## Purpose

T12 has one verification job: prove that the Linear compatibility shell can expose `linear_graphql`, preserve the Codex protocol shape the app-server expects, and keep Linear write behavior out of the Go core.

This strategy maps each check to the contract it proves. The goal is evidence, not a test inventory.

## Proof Matrix

| Behavior or risk | Check | What it proves |
| --- | --- | --- |
| `internal/toolbridge/linear` can stand up the Linear bridge without widening core dependencies | `go test ./internal/toolbridge/linear/...` | The bridge compiles and runs as a leaf package, with a narrow client interface and no need for `internal/tracker`, `internal/orchestrator`, or `internal/domain` symbols. |
| Exactly one dynamic tool is exposed and it stays named `linear_graphql` | `go test ./internal/toolbridge/linear/...` | `ToolSpecs()` stays stable enough for Codex injection and does not accrete extra Linear tool names in this slice. |
| `linear_graphql` accepts raw GraphQL input and preserves trimmed queries | `go test ./internal/toolbridge/linear/...` | The bridge dispatches the raw query string to the client after trimming, and the handler can reject blank or malformed query payloads with the expected Symphony message. |
| `linear_graphql` preserves object arguments and rejects invalid variables | `go test ./internal/toolbridge/linear/...` | Query plus `variables` stay in the compatibility-shell bridge, `variables` must be an object when present, and the bridge does not silently coerce bad input. |
| Unknown tool names fail with `supportedTools` in the payload | `go test ./internal/toolbridge/linear/...` | Unsupported tool calls stay a bridge-local failure path, not a catch-all dispatcher, and the failure response names `linear_graphql` as the supported tool. |
| Linear GraphQL transport/status/GraphQL errors stay distinguishable | `go test ./internal/toolbridge/linear/...` | Request failures, non-200 responses, and GraphQL `errors` payloads are not collapsed into one generic failure class. |
| `CreateComment` preserves the old Linear mutation contract | `go test ./internal/toolbridge/linear/...` | The helper issues `commentCreate(input: {issueId, body})`, accepts only `success == true`, and maps failure to `ErrCommentCreateFailed`. |
| `UpdateIssueState` resolves the target state before mutation | `go test ./internal/toolbridge/linear/...` | The helper first looks up the state id, then calls `issueUpdate` with that id, and maps missing state or failed mutation to the expected helper errors. |
| The bridge does not grow a provider-neutral write API by accident | `go test ./internal/toolbridge/linear/...` plus compile-time assertions in the package tests | The package remains a compatibility-shell helper layer, not a new core tracker abstraction. |
| Raw string `item/tool/call` arguments survive the Codex boundary | `go test ./internal/codex/...` | `ToolCall.Arguments` can carry a raw string value through session handling instead of being forced into `map[string]any`. |
| Tool results serialize `contentItems` at the top level of the JSON-RPC result | `go test ./internal/codex/...` | The session writes the protocol envelope as `{"id":..., "result":...}` where `result.contentItems` is present directly, and not nested under an extra `result.result`. This is the check that closes the review gap on protocol shape. |
| Unsupported tool responses remain protocol-compatible | `go test ./internal/codex/...` | The JSON-RPC response still carries the failure payload expected by Symphony, including `supportedTools` for the bridge-level unsupported-tool case. |
| The new bridge boundary does not break the rest of the repository | `go test ./...` | The codex protocol change, the bridge package, and the existing reader/core packages still compile and link together after the slice lands. |
| The repository still produces the normal binary | `make build` | The canonical build path continues to work with the new codex/toolbridge boundary. |
| Static checks still pass across the new protocol and bridge code | `make lint` | Formatting, vet, and static analysis stay green, especially around JSON shaping, context handling, and error classification. |
| The repo-level e2e entrypoint still runs | `make test-e2e` | The task does not regress the top-level e2e command contract, even though this slice is not yet wiring the bridge into the full runtime path. |
| The OpenSpec change is structurally valid | `openspec validate linear-toolbridge` | The change artifacts and task-scoped spec files are internally consistent before implementation closes. |
| The spec tree still accepts the delta after sync | `openspec validate --specs` | The synced spec set remains valid once the change is folded back into the main spec tree. |

## Package-Scoped Verification

The first proof gate is package-local:

```bash
go test ./internal/toolbridge/linear/... ./internal/codex/...
```

This gate should answer the core T12 questions directly:

1. Can the bridge register exactly `linear_graphql` and call the Linear client with the expected query and variables?
2. Can the bridge return the old failure payload shape for unsupported tools, including `supportedTools`?
3. Can the Codex session preserve raw string arguments from `item/tool/call`?
4. Can the Codex session emit `contentItems` at the top level of the JSON-RPC `result` object?
5. Do the Linear write helpers preserve the old mutation sequencing and error mapping?

If any of those fail, the change is not ready for broader verification.

## Broader Gates

After the package gate passes, run:

```bash
go test ./...
make build
make lint
make test-e2e
openspec validate linear-toolbridge
```

These checks do not prove new T12 behavior by themselves. They prove the change did not leak into unrelated packages, break the normal build, or violate the OpenSpec change contract.

`make test-e2e` is a regression guard only. This slice does not yet wire the new bridge into the full runtime path, so a green e2e command is useful but not the primary proof of T12 behavior.

## Dependency Boundary Guard

The dependency guard is a compile-time fence, not a runtime assertion.

It should prove that `internal/toolbridge/linear` only needs the narrow compatibility-shell surface:

- `internal/config`
- `internal/codex`
- the Linear GraphQL client interface used by the bridge

The package tests should fail if the bridge starts depending on `internal/tracker`, `internal/orchestrator`, or `internal/domain` to satisfy Linear writes. That failure is the point: it keeps the write helpers in the compatibility shell and prevents a new core write API from sneaking in.

## Out Of Scope

This strategy does not try to prove:

- workflow selection
- injection into `codex.Config.DynamicTools` or `SessionOptions.ToolHandler`
- any full runtime or end-to-end Linear run
- dashboard, HTTP API, or CLI presentation behavior
- provider-neutral tracker writes
- non-Linear providers

Those are later integration concerns. T12 only proves the bridge, the Codex protocol shape, the Linear write helpers, and the boundary that keeps both out of core.
