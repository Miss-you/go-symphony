# T12 Final Implementation v1

## Task Goal

Implement the Linear ToolBridge entirely in the compatibility shell, preserving Symphony's `linear_graphql` behavior without widening any core runtime boundary.

This task should preserve:

- one Linear dynamic tool named `linear_graphql`
- raw GraphQL passthrough with trimmed query strings
- optional variables as a JSON object
- stable success and failure semantics in tool results
- the existing `internal/codex` protocol boundary
- the current separation between read-only core tracker code and provider-specific write behavior

This task must not introduce:

- a universal tracker write API
- any provider-specific write fields in `domain`
- any Linear write logic in `orchestrator`
- any `tracker.TrackerReader` expansion
- any new core abstraction just to support Linear writes

## Scope Boundary

The compatibility shell owns the Linear tool surface. The core should only see the generic Codex protocol, a tool spec list, and a tool handler.

The bridge should remain a leaf package:

- `internal/toolbridge/linear` may depend on `internal/config`, `internal/codex`, and the narrow Linear GraphQL client interface
- it must not depend on `internal/tracker`
- it must not depend on `internal/orchestrator`
- it must not depend on `internal/domain`
- it must not teach `internal/codex` any Linear-specific logic

The workflow/assembly layer may later inject the bridge into `codex.Config.DynamicTools` and `SessionOptions.ToolHandler`, but that assembly is not part of the core or orchestrator.

## Package / API Shape

Create a small `internal/toolbridge/linear` package with one exported bridge type.

Recommended public surface:

```go
type Client interface {
	GraphQL(context.Context, string, map[string]any) (map[string]any, error)
}

type Bridge struct {
	// settings and client
}

func New(settings config.ProviderSettings, client Client) (*Bridge, error)
func (b *Bridge) ToolSpecs() []codex.ToolSpec
func (b *Bridge) HandleTool(context.Context, codex.ToolCall) (codex.ToolResult, error)
func (b *Bridge) CreateComment(context.Context, issueID, body string) error
func (b *Bridge) UpdateIssueState(context.Context, issueID, stateName string) error
```

Implementation rules:

- `Bridge` itself should satisfy `codex.ToolHandler`
- `ToolSpecs()` should return a copy, not a shared slice
- `HandleTool()` should accept only `linear_graphql`
- unsupported tool names should return a Symphony-style failed tool result that includes `supportedTools`
- `HandleTool()` should only read `ToolCall.Name` and `ToolCall.Arguments`; raw protocol parsing stays in `internal/codex`

Constructor behavior:

- if `client` is nil, build the default HTTP-backed Linear client from `settings`
- API key is required when the bridge needs to construct its own client
- endpoint may default to the Linear GraphQL endpoint
- `Project` is not required for this bridge, because `linear_graphql` is a raw GraphQL tool and should not depend on workflow selection
- do not require `tracker` reads or any write API from core packages

This keeps the bridge aligned with the current reader-side transport shape while keeping the bridge independent from tracker runtime policy.

The comment and state methods are provider-specific compatibility helpers, not a new universal tracker write API. They mirror the old `SymphonyElixir.Linear.Adapter` write behavior so later workflow/integration code can call Linear writes without widening `internal/tracker`.

## Tool Spec

Expose exactly one Codex dynamic tool in this task:

`linear_graphql`

Schema requirements:

- object type
- `additionalProperties: false`
- required field: `query`
- optional field: `variables`
- `variables` may be a JSON object or `null`

The tool spec should preserve the current Symphony contract:

- query-only input is accepted through the object form
- raw string input is accepted when a caller constructs a `codex.ToolCall` with a string `Arguments` value
- query strings are trimmed before execution
- blank queries are rejected
- `operationName` is ignored
- no extra tool names are added in this slice

Do not add dedicated Codex tool names for comments or state transitions in this task. The old unattended workflow can already perform those mutations through raw `linear_graphql`; the package-level `CreateComment` and `UpdateIssueState` helpers exist for compatibility-shell runtime/workflow code, not for Codex tool advertisement.

Unsupported tool contract:

- unknown tool names return `success: false`
- the result contains one `inputText` content item
- that content item JSON-decodes to `{"error":{"message":"Unsupported dynamic tool: <tool>.","supportedTools":["linear_graphql"]}}`
- this is a bridge-local failed tool result, not `codex.ErrUnsupportedTool`, so the old DynamicTool payload shape is preserved when the bridge handles the call

## Execution Semantics

`linear_graphql` should behave like a raw GraphQL passthrough.

On success:

- send `query` and `variables` directly to the Linear GraphQL client
- preserve the returned JSON body
- format map/list payloads as pretty JSON text
- fall back to a stable string form for non-JSON-marshable values

On failure:

- reject missing or blank query strings with the existing Symphony message
- reject invalid arguments with the existing Symphony message
- reject non-object variables with the existing Symphony message
- surface missing auth with the existing Linear auth message
- classify non-200 responses separately from request failures
- classify GraphQL responses that contain a non-empty `errors` list as unsuccessful even when HTTP status is 200
- preserve the response body text in the result payload when possible
- encode unknown-tool failures with `supportedTools: ["linear_graphql"]`

Recommended failure strings to preserve:

- `` `linear_graphql` requires a non-empty `query` string. ``
- `` `linear_graphql` expects either a GraphQL query string or an object with `query` and optional `variables`. ``
- `` `linear_graphql.variables` must be a JSON object when provided. ``
- `Symphony is missing Linear auth. Set \`linear.api_key\` in \`WORKFLOW.md\` or export \`LINEAR_API_KEY\`.`
- `Linear GraphQL request failed with HTTP <status>.`
- `Linear GraphQL request failed before receiving a successful response.`
- `Linear GraphQL tool execution failed.`

## Linear Write Helpers

Implement the old Linear adapter write behavior in `internal/toolbridge/linear` as provider-specific helper methods:

- `CreateComment(ctx, issueID, body)` runs `commentCreate(input: {issueId, body})` and succeeds only when `data.commentCreate.success == true`
- `UpdateIssueState(ctx, issueID, stateName)` resolves the target state id through `issue(id).team.states(filter: {name: {eq: stateName}}, first: 1)` and then runs `issueUpdate(id: issueID, input: {stateId})`
- comment failure maps to `ErrCommentCreateFailed`
- missing state maps to `ErrStateNotFound`
- failed state mutation maps to `ErrIssueUpdateFailed`
- GraphQL transport/status/context errors bubble up without being collapsed into the helper-specific errors

These methods are intentionally not added to `internal/tracker.TrackerReader`. They are Linear compatibility-shell behavior used by later workflow/integration assembly.

## Codex Boundary Adjustment

Change `internal/codex` only where the existing generic tool boundary is too narrow for the compatibility contract.

Reason:

- Elixir returns dynamic tool responses as top-level `{success, contentItems}` payloads.
- Nesting `contentItems` under `ToolResult.Result` would serialize a different protocol result shape and risk hiding tool output from Codex.
- Elixir accepts raw string arguments in addition to object arguments. The current Go `codex.ToolCall.Arguments map[string]any` cannot represent that.

Recommended approach:

- add a generic `ToolContentItem` type in `internal/codex`
- extend `codex.ToolResult` with `ContentItems []ToolContentItem` tagged as `json:"contentItems,omitempty"`
- keep existing `Result` and `Error` fields so T09 behavior and tests remain compatible
- change `codex.ToolCall.Arguments` from `map[string]any` to `any`
- update `toolCallArguments` to preserve string arguments, clone object arguments, and default missing arguments to `map[string]any{}`
- return `linear_graphql` output as top-level `contentItems`, matching Symphony's app-server result shape

This remains a generic protocol-boundary change, not Linear logic in core. `internal/codex` still does not know about `linear_graphql`; it only gains enough shape to carry the tool result and arguments that the app-server already supports.

Framing detail to test explicitly:

- `Session.handleToolCall` still writes a JSON-RPC response shaped as `{"id": <call-id>, "result": <tool-result>}`
- `<tool-result>` must contain `contentItems` directly, not under a nested `result.contentItems`
- this matches Elixir, which sends `%{"id" => id, "result" => DynamicTool.execute(...)}`

## TDD Plan

Write the bridge tests before implementation and keep them package-scoped.

Required tests:

- `TestNewConstructsBridgeWithLinearClient`
  - proves the bridge can be built with a narrow client interface
  - proves the constructor validates the minimum settings it owns

- `TestToolSpecsExposeLinearGraphQL`
  - proves exactly one tool spec is exposed
  - proves the tool name stays `linear_graphql`
  - proves the schema shape is stable enough for Codex injection

- `TestHandleToolDispatchesLinearGraphQL`
  - proves `linear_graphql` calls the client with the expected query and variables
  - proves trimmed query handling
  - proves raw string arguments are preserved through the handler boundary
  - proves successful results are marked successful
  - proves the formatted body is returned as a top-level `contentItems` item

- `TestHandleToolRejectsUnknownTool`
  - proves unsupported tool names return a failed content-item result with `supportedTools`
  - keeps the bridge from becoming a catch-all dispatcher while preserving old dynamic-tool output

- `TestHandleToolRejectsBlankQueryAndInvalidVariables`
  - covers blank queries
  - covers non-object variables
  - covers malformed argument payloads

- `TestHandleToolMapsTransportStatusAndGraphQLErrorDetails`
  - covers request failures
  - covers non-200 status responses
  - covers GraphQL `errors` payloads
  - proves the error text is still visible to Codex

- `TestCodexToolCallPreservesRawStringArguments`
  - proves `item/tool/call` with string `params.arguments` reaches the handler as a string
  - protects the raw-query compatibility path

- `TestCodexToolResultSerializesContentItemsAtTopLevel`
  - proves the written `item/tool/call` response has `response.result.contentItems`
  - proves there is no nested `response.result.result.contentItems`
  - protects app-server protocol parity without adding Linear knowledge to `codex`

- `TestCreateCommentRunsLinearMutation`
  - proves `CreateComment` sends the old `commentCreate` mutation and expected variables
  - proves only `success == true` is accepted
  - proves failure maps to `ErrCommentCreateFailed`

- `TestUpdateIssueStateResolvesStateThenMutatesIssue`
  - proves state name lookup runs before `issueUpdate`
  - proves the resolved state id is used in the mutation variables
  - proves missing state maps to `ErrStateNotFound`
  - proves failed mutation maps to `ErrIssueUpdateFailed`

- `TestBridgeUsesNarrowClientInterface`
  - compile-time assertion that the bridge client dependency stays narrow
  - compile-time assertion that `internal/trackers/linear.HTTPClient` satisfies that dependency

- `TestBridgeDoesNotRequireCoreTrackerWrites`
  - run a dependency guard such as `go list -deps ./internal/toolbridge/linear` and assert it does not include `internal/tracker`, `internal/orchestrator`, or `internal/domain`
  - confirms the bridge stays in the compatibility shell and does not consume `tracker.TrackerReader`

Recommended TDD order:

1. tool spec shape
2. unsupported tool dispatch
3. success dispatch
4. blank query and argument validation
5. formatting and `contentItems` preservation
6. transport, HTTP, and GraphQL error mapping
7. Codex argument/result boundary regression tests
8. compile-time interface guards

## Verification Gates

Primary package gate:

```bash
go test ./internal/codex/... ./internal/toolbridge/...
```

Then run the broader repo gates before closing the task:

```bash
go test ./...
make build
make lint
make test-e2e
```

If an OpenSpec change is created from this plan, validate that change before implementation work is considered done.

## Non-Goals

This task does not:

- add a universal tracker write API
- expand `internal/tracker`
- move provider-specific write semantics into `domain`
- teach `orchestrator` about Linear writes
- add a generic workpad abstraction
- add any Lark-specific runtime behavior
- add workflow selection or `compat_linear_default`
- add more than one Codex dynamic tool in the first slice
- add Linear-specific logic to the Codex app-server protocol model

## Delivery Notes

This should land as a conservative compatibility-shell bridge:

- small public surface
- one explicit tool
- one narrow client interface
- no provider-specific core boundary expansion
- no hidden coupling to tracker writes or orchestrator state

That is the right size for T12. It preserves the Symphony `linear_graphql` contract, keeps future Linear write helpers local to the compatibility layer, and leaves the core packages provider-neutral.
