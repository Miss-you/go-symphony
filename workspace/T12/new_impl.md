# T12 Linear ToolBridge Research

## Current Shape

The Go implementation already has the right protocol seam for provider-specific tools:

- `internal/codex` accepts `DynamicTools []ToolSpec` during `thread/start`.
- `internal/codex` routes `item/tool/call` through a generic `ToolHandler`.
- `internal/trackers/linear` is read-only and already owns the reusable Linear GraphQL transport.
- `internal/tracker` is intentionally read-only and must stay that way.
- `internal/orchestrator` owns runtime state and snapshot projection, so it should not learn Linear write details.

That means T12 does not need a new core abstraction. It needs a compatibility-shell leaf package that can be plugged into Codex and can speak Linear GraphQL without widening the tracker contract.

## Recommended Implementation Shape

Create `internal/toolbridge/linear` as a small compatibility package with three responsibilities:

1. define the Linear-specific Codex tools
2. translate Codex tool calls into Linear write operations
3. keep all write semantics outside core packages

Recommended public surface:

```go
type Client interface {
    GraphQL(context.Context, string, map[string]any) (map[string]any, error)
}

type Bridge struct { ... }

func New(settings config.ProviderSettings, client Client) (*Bridge, error)
func (b *Bridge) ToolSpecs() []codex.ToolSpec
func (b *Bridge) Handler() codex.ToolHandler
func (b *Bridge) HandleTool(context.Context, codex.ToolCall) (codex.ToolResult, error)
```

Key point: the bridge should depend on a tiny GraphQL client interface, not on `tracker.TrackerReader` or `orchestrator`. That keeps the write path leaf-like. Existing `internal/trackers/linear.HTTPClient` already satisfies the client shape, so the bridge can reuse it without importing the reader behavior.

### Tool Routing

Start with one explicit tool:

- `linear_graphql`

The first implementation should keep this tool raw and direct:

- input: GraphQL query string plus optional variables
- execution: `Client.GraphQL`
- output: pass through the returned JSON data shape on success
- failure: return a structured `ToolResult{Success:false, Error:...}` and preserve transport / GraphQL error details

If later Symphony parity proves that other Linear writes deserve dedicated tool names, add them in this same package as explicit handlers. Do not push them into core, and do not invent a universal write API to support them.

### Assembly Flow

The expected dependency flow is:

```text
workflow bundle / CLI assembly
        |
        v
internal/toolbridge/linear -> codex.ToolSpec + codex.ToolHandler
        |
        v
internal/codex.Session
        |
        v
Linear GraphQL client
```

The orchestrator should remain unaware of `linear_graphql`. It only sees Codex events and `RunEvent`s. The workflow layer, once it exists, can decide when to inject the bridge into `codex.Config.DynamicTools` and when to attach the handler.

## Boundary Constraints

This task should keep the following lines intact:

- no new methods on `tracker.TrackerReader`
- no provider-specific write fields in `domain.WorkItem`
- no Linear write logic in `orchestrator`
- no `linear_graphql` references in `codex` beyond generic tool plumbing
- no observability state for tool execution beyond the existing Codex event stream

The bridge can carry Linear workflow state if a later workflow bundle needs it, but that state should stay in the compatibility shell and not leak into core packages.

## Tests To Write

Package-scoped tests should prove the bridge is a leaf and that tool dispatch is deterministic:

1. `TestBridgeConstructsWithLinearClient`
   - validates config / client wiring
   - checks missing API key / endpoint / project behavior, if the bridge requires them

2. `TestToolSpecsExposeLinearGraphQL`
   - verifies `ToolSpecs()` returns `linear_graphql`
   - verifies the tool schema stays stable enough for Codex injection

3. `TestHandleToolDispatchesLinearGraphQL`
   - sends a `codex.ToolCall` named `linear_graphql`
   - checks the GraphQL client is called with the expected query and variables
   - checks the returned `ToolResult` is successful and preserves the JSON payload shape

4. `TestHandleToolRejectsUnknownTool`
   - confirms unknown tool names return `codex.ErrUnsupportedTool`
   - keeps the bridge from becoming a catch-all dispatcher

5. `TestHandleToolPropagatesGraphQLErrorDetails`
   - covers request failures, non-200 status, and GraphQL error payloads
   - ensures the error text is visible to Codex without mutating core state

6. `TestBridgeUsesNarrowClientInterface`
   - compile-time assertion that `*Bridge` or its handler satisfies `codex.ToolHandler`
   - compile-time assertion that `trackers/linear.HTTPClient` can be passed directly as the client dependency

7. `TestBridgeDoesNotRequireTrackerWriteAPI`
   - guards the design goal that tracker writes stay out of `internal/tracker`
   - this can be a package-level compile/test guard rather than a runtime test

## Open Decisions

- Whether `linear_graphql` should return the raw GraphQL envelope or only `data` on success.
- Whether any write helper beyond `linear_graphql` is needed before T13 lands the workflow bundle.
- Whether the bridge should validate `ProviderSettings.Project` eagerly even if a given write tool does not need it.

My recommendation is to keep the first cut minimal: one bridge, one raw GraphQL tool, one narrow client interface, and no changes to the core tracker or orchestrator surfaces.
