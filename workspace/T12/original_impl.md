# T12 Linear ToolBridge: Old Symphony Implementation

## Scope

The Elixir implementation exposes exactly one Codex dynamic tool for Linear: `linear_graphql`. It is registered in the Codex app-server session, executed client-side by Symphony, and backed by the Linear GraphQL client. Separate tracker write APIs (`commentCreate`, `issueUpdate`) exist in the Linear adapter, but they are not a second dynamic tool; they are used by the tracker boundary and live E2E prompt flow.

## Dynamic tool shape and injection

- `SymphonyElixir.Codex.DynamicTool.tool_specs/0` returns a one-element list containing `name`, `description`, and `inputSchema` for `linear_graphql` ([`dynamic_tool.ex`](/Users/apple/Documents/Github/symphony/elixir/lib/symphony_elixir/codex/dynamic_tool.ex#L8)).
- The schema is an object with `additionalProperties: false`, required `query`, and optional `variables` that may be an object or `null` ([`dynamic_tool.ex`](/Users/apple/Documents/Github/symphony/elixir/lib/symphony_elixir/codex/dynamic_tool.ex#L12)).
- `AppServer.start_thread/3` injects `DynamicTool.tool_specs()` into the Codex `thread/start` payload under `params.dynamicTools` ([`app_server.ex`](/Users/apple/Documents/Github/symphony/elixir/lib/symphony_elixir/codex/app_server.ex#L239)).
- Tool calls are handled in the app-server by `maybe_handle_approval_request/8` for `item/tool/call`; it accepts either `params.tool` or `params.name`, plus `params.arguments` ([`app_server.ex`](/Users/apple/Documents/Github/symphony/elixir/lib/symphony_elixir/codex/app_server.ex#L505) and [`app_server.ex`](/Users/apple/Documents/Github/symphony/elixir/lib/symphony_elixir/codex/app_server.ex#L941)).
- The default `tool_executor` in `AppServer.run_turn/4` delegates directly to `DynamicTool.execute/3` ([`app_server.ex`](/Users/apple/Documents/Github/symphony/elixir/lib/symphony_elixir/codex/app_server.ex#L80)).

## `linear_graphql` execution semantics

- Supported tool names: only `linear_graphql`. Unknown tools return a failure payload that includes `supportedTools: ["linear_graphql"]` ([`dynamic_tool.ex`](/Users/apple/Documents/Github/symphony/elixir/lib/symphony_elixir/codex/dynamic_tool.ex#L30)).
- Inputs accepted:
  - raw query string, trimmed; blank strings are rejected
  - map/object with `query` and optional `variables`
- Legacy `operationName` in the tool input is ignored by the dynamic tool. The tool forwards only `query` and `variables` to the Linear client ([`dynamic_tool_test.exs`](/Users/apple/Documents/Github/symphony/elixir/test/symphony_elixir/dynamic_tool_test.exs#L92)).
- `variables` must be a JSON object when present; non-maps fail before any Linear call.
- On success, the tool returns `{success: true, contentItems: [%{type: "inputText", text: ...}]}`.
- Map/list responses are JSON-encoded with pretty formatting; non-JSON payloads fall back to `inspect/1`.
- A GraphQL response is marked unsuccessful when it contains a non-empty `errors` list, even if HTTP 200 was returned. The body is still preserved in `contentItems`.

## Error/result formatting to preserve

`DynamicTool.execute/3` emits structured tool failures rather than raising:

- missing query -> `` `linear_graphql` requires a non-empty `query` string. ``
- invalid arguments -> `` `linear_graphql` expects either a GraphQL query string or an object with `query` and optional `variables`. ``
- invalid variables -> `` `linear_graphql.variables` must be a JSON object when provided. ``
- missing auth -> `Symphony is missing Linear auth. Set \`linear.api_key\` in \`WORKFLOW.md\` or export \`LINEAR_API_KEY\`.`
- non-200 response -> `Linear GraphQL request failed with HTTP <status>.` plus `status`
- request failure -> `Linear GraphQL request failed before receiving a successful response.` plus `reason`
- unexpected client error -> `Linear GraphQL tool execution failed.` plus `reason`

Every failure is returned as a tool result with `success: false` and a single `inputText` content item containing JSON.

## Linear client behavior behind the tool

`SymphonyElixir.Linear.Client.graphql/3` is the transport the tool uses by default ([`linear/client.ex`](/Users/apple/Documents/Github/symphony/elixir/lib/symphony_elixir/linear/client.ex#L163)):

- builds payload as `{"query", "variables"}` and optionally `operationName` when a nonblank `:operation_name` opt is supplied
- sends `Authorization` and `Content-Type: application/json` headers
- posts to `Config.settings!().tracker.endpoint`
- uses a 30s connect timeout
- returns `{:ok, body}` only on HTTP 200; otherwise maps the response to `{:error, {:linear_api_status, status}}`
- request transport errors become `{:error, {:linear_api_request, reason}}`
- non-200 logs include the operation name when present and a summarized/truncated response body

The client’s Linear-specific decode path treats `%{"errors" => errors}` as an error tuple so the dynamic tool can flag the turn as unsuccessful while still surfacing the GraphQL body ([`linear/client.ex`](/Users/apple/Documents/Github/symphony/elixir/lib/symphony_elixir/linear/client.ex#L405)).

## Linear write behavior adjacent to the bridge

The tracker adapter is the separate write path for Linear issue mutations:

- `Tracker` defines `create_comment/2` and `update_issue_state/2` as tracker writes ([`tracker.ex`](/Users/apple/Documents/Github/symphony/elixir/lib/symphony_elixir/tracker.ex#L8)).
- `Linear.Adapter.create_comment/2` runs a `commentCreate` mutation and expects `data.commentCreate.success == true`; otherwise it returns `{:error, :comment_create_failed}` ([`linear/adapter.ex`](/Users/apple/Documents/Github/symphony/elixir/lib/symphony_elixir/linear/adapter.ex#L49)).
- `Linear.Adapter.update_issue_state/2` first resolves the state id via `issue.team.states(first: 1)` and then runs `issueUpdate`; failures map to `:state_not_found` or `:issue_update_failed` as appropriate ([`linear/adapter.ex`](/Users/apple/Documents/Github/symphony/elixir/lib/symphony_elixir/linear/adapter.ex#L61)).
- The adapter gets its client module from `Application.get_env(:symphony_elixir, :linear_client_module, Client)`, which makes the write path testable without the real API ([`linear/adapter.ex`](/Users/apple/Documents/Github/symphony/elixir/lib/symphony_elixir/linear/adapter.ex#L76)).

The live E2E prompt uses the same `linear_graphql` tool for the read/write sequence: query issue context, post a comment with `commentCreate`, move the issue with `issueUpdate`, then verify both outcomes with a final query ([`live_e2e_test.exs`](/Users/apple/Documents/Github/symphony/elixir/test/symphony_elixir/live_e2e_test.exs#L390)).

## Tests and proofs

- [`dynamic_tool_test.exs`](/Users/apple/Documents/Github/symphony/elixir/test/symphony_elixir/dynamic_tool_test.exs) covers tool registration, unknown-tool failures, raw string input, map input, blank query rejection, invalid variables, GraphQL errors, transport/auth failures, unexpected client failures, and non-JSON payload fallback.
- [`app_server_test.exs`](/Users/apple/Documents/Github/symphony/elixir/test/symphony_elixir/app_server_test.exs#L420) proves `thread/start` includes `dynamicTools` with the `linear_graphql` schema, and later tests prove `item/tool/call` accepts both `name` and `tool`, forwards arguments unchanged, surfaces success/failure distinctly, and rejects unsupported tool calls without stalling.
- [`extensions_test.exs`](/Users/apple/Documents/Github/symphony/elixir/test/symphony_elixir/extensions_test.exs#L220) proves the Linear adapter’s comment/state write semantics and error mapping.
- [`live_e2e_test.exs`](/Users/apple/Documents/Github/symphony/elixir/test/symphony_elixir/live_e2e_test.exs#L390) documents the intended real-world write flow through `linear_graphql`.

## Go must preserve

1. One dynamic tool, named `linear_graphql`, injected into Codex session startup.
2. Raw GraphQL passthrough with trimmed query strings and optional variables.
3. Tool-level JSON result wrapping with stable success/failure semantics.
4. Distinct error strings for bad inputs, missing Linear auth, HTTP errors, request errors, and unexpected client errors.
5. Acceptance of both `tool` and `name` in incoming Codex tool calls.
6. Linear client behavior that returns bodies on HTTP 200, surfaces GraphQL `errors`, and logs non-200s with summarized bodies.
7. The separate Linear write adapter for comment creation and state changes, including state-id lookup.
