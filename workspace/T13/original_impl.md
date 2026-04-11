# T13 Original Implementation Notes

Source of truth: original Symphony Elixir implementation in `/Users/apple/Documents/Github/symphony/elixir`.

## Inspected Paths

- `/Users/apple/Documents/Github/symphony/elixir/WORKFLOW.md`
- `/Users/apple/Documents/Github/symphony/elixir/lib/symphony_elixir/workflow.ex`
- `/Users/apple/Documents/Github/symphony/elixir/lib/symphony_elixir/workflow_store.ex`
- `/Users/apple/Documents/Github/symphony/elixir/lib/symphony_elixir/config.ex`
- `/Users/apple/Documents/Github/symphony/elixir/lib/symphony_elixir/config/schema.ex`
- `/Users/apple/Documents/Github/symphony/elixir/lib/symphony_elixir/prompt_builder.ex`
- `/Users/apple/Documents/Github/symphony/elixir/lib/symphony_elixir/cli.ex`
- `/Users/apple/Documents/Github/symphony/elixir/lib/symphony_elixir/codex/app_server.ex`
- `/Users/apple/Documents/Github/symphony/elixir/lib/symphony_elixir/codex/dynamic_tool.ex`
- `/Users/apple/Documents/Github/symphony/elixir/lib/symphony_elixir/linear/client.ex`
- `/Users/apple/Documents/Github/symphony/elixir/lib/symphony_elixir/linear/adapter.ex`
- `/Users/apple/Documents/Github/symphony/elixir/lib/symphony_elixir/linear/issue.ex`
- `/Users/apple/Documents/Github/symphony/elixir/test/symphony_elixir/core_test.exs`
- `/Users/apple/Documents/Github/symphony/elixir/test/symphony_elixir/dynamic_tool_test.exs`
- `/Users/apple/Documents/Github/symphony/elixir/test/symphony_elixir/cli_test.exs`
- `/Users/apple/Documents/Github/symphony/elixir/test/symphony_elixir/app_server_test.exs`
- `/Users/apple/Documents/Github/symphony/elixir/test/symphony_elixir/live_e2e_test.exs`

## Behavior Summary

### Workflow loading and selection

`SymphonyElixir.Workflow` is the primary loader. It reads a single `WORKFLOW.md` file, splits optional YAML front matter from the markdown body, and returns:

- `config`: parsed front matter map
- `prompt`: the markdown body after trimming
- `prompt_template`: same body, used later for rendering

Relevant behavior:

- Default path is `WORKFLOW.md` in the current working directory when no override is set.
- `Application` env `:workflow_file_path` wins over the CWD default.
- CLI startup also defaults to `WORKFLOW.md` when no explicit path is passed.
- `WorkflowStore` keeps the last known good workflow in memory, polls for file changes every 1 second, and reloads on path change or stamp change.
- If a reload fails, the store keeps serving the previous valid workflow instead of crashing the session.
- `Workflow.load/1` accepts prompt-only files with no front matter, and accepts unterminated front matter by treating the body as empty.
- Non-map YAML front matter is rejected with `:workflow_front_matter_not_a_map`.

Relevant functions:

- `SymphonyElixir.Workflow.workflow_file_path/0`
- `SymphonyElixir.Workflow.load/0`
- `SymphonyElixir.Workflow.load/1`
- `SymphonyElixir.WorkflowStore.current/0`
- `SymphonyElixir.WorkflowStore.force_reload/0`
- `SymphonyElixir.WorkflowStore.handle_info/2`

### Default Linear workflow bundle

The bundled `/Users/apple/Documents/Github/symphony/elixir/WORKFLOW.md` is a Linear-oriented unattended orchestration workflow, not a generic blank template.

Front matter config in that file sets:

- `tracker.kind: linear`
- `tracker.project_slug: "symphony-0c79b11b75ea"`
- `tracker.active_states: [Todo, In Progress, Merging, Rework]`
- `tracker.terminal_states: [Closed, Cancelled, Canceled, Duplicate, Done]`
- `polling.interval_ms: 5000`
- `workspace.root: ~/code/symphony-workspaces`
- `hooks.after_create`: clone the Symphony repo and bootstrap deps
- `hooks.before_remove`: run `mix workspace.before_remove`
- `agent.max_concurrent_agents: 10`
- `agent.max_turns: 20`
- `codex.command`: `codex --config shell_environment_policy.inherit=all --config model_reasoning_effort=xhigh --model gpt-5.3-codex app-server`
- `codex.approval_policy: never`
- `codex.thread_sandbox: workspace-write`
- `codex.turn_sandbox_policy.type: workspaceWrite`

The runtime `Config.Schema` layer fills in defaults when values are omitted:

- `workspace.root` falls back to a tmpdir-based `symphony_workspaces` path
- `codex.approval_policy` defaults to the reject/sandbox approval policy map
- `codex.thread_sandbox` defaults to `workspace-write`
- `codex.turn_sandbox_policy` defaults to a workspace-write policy rooted at the configured workspace root

### Prompt and workflow text

`SymphonyElixir.Config.workflow_prompt/0` returns the markdown body from `WORKFLOW.md`, or a built-in fallback template when the body is blank or missing.

The built-in fallback prompt is the short Linear issue template:

- `You are working on a Linear issue.`
- `Identifier: {{ issue.identifier }}`
- `Title: {{ issue.title }}`
- `Body:` with a `No description provided.` fallback

`SymphonyElixir.PromptBuilder` then renders the chosen prompt with strict Solid templating. It supplies:

- `issue` as a normalized map
- `attempt` when retrying a session

Important user-visible semantics in the bundled workflow body:

- the session is explicitly described as unattended
- no human follow-up should be requested
- only true blockers should stop the run early
- final messaging should report completed actions and blockers only
- retry attempts should resume from current workspace state rather than restarting

Prompt builder behavior to preserve:

- strict variable resolution
- template parse failures surface as `template_parse_error: ...`
- missing workflow file surfaces as `workflow_unavailable: ...`
- date/time and nested struct values are normalized before rendering

### Dynamic tool expectations

The Codex app-server advertises dynamic tools from `SymphonyElixir.Codex.DynamicTool.tool_specs/0`.

Current surface area:

- exactly one dynamic tool is exposed: `linear_graphql`
- `codex/app_server.ex` passes that tool list into `thread/start` as `dynamicTools`
- the tool is documented as a raw GraphQL execution path against Linear using Symphony auth

Tool contract details to preserve:

- accepts either a raw GraphQL query string or an object with `query` and optional `variables`
- blank queries are rejected
- non-map `variables` are rejected
- legacy `operationName` input is ignored
- multi-operation documents are forwarded unchanged
- GraphQL error bodies still return the tool result body, but `success` becomes `false`
- unsupported tool names return a failure payload that includes `supportedTools: ["linear_graphql"]`

Linear-specific execution path:

- `linear_graphql` delegates to `SymphonyElixir.Linear.Client.graphql/3`
- Linear transport/auth failures are translated into stable JSON error payloads
- the missing-auth message tells the user to set `linear.api_key` in `WORKFLOW.md` or export `LINEAR_API_KEY`

### Config inputs and validation

`SymphonyElixir.Config.Schema` is the config parser/validator for `WORKFLOW.md` front matter. Relevant fields:

- `tracker.kind`
- `tracker.endpoint`
- `tracker.api_key`
- `tracker.project_slug`
- `tracker.assignee`
- `tracker.active_states`
- `tracker.terminal_states`
- `polling.interval_ms`
- `workspace.root`
- `agent.max_concurrent_agents`
- `agent.max_turns`
- `agent.max_retry_backoff_ms`
- `agent.max_concurrent_agents_by_state`
- `codex.command`
- `codex.approval_policy`
- `codex.thread_sandbox`
- `codex.turn_sandbox_policy`
- `codex.turn_timeout_ms`
- `codex.read_timeout_ms`
- `codex.stall_timeout_ms`
- `hooks.*`
- `observability.*`
- `server.*`

Validation and runtime resolution semantics:

- `tracker.kind` must be `linear` or `memory`
- linear mode requires a Linear API token and a project slug
- `codex.command` is required
- `LINEAR_API_KEY` and `LINEAR_ASSIGNEE` can fill missing tracker values
- `tracker.active_states` and `tracker.terminal_states` are normalized through config parsing
- workspace roots are canonicalized and path-safe before use

### User-visible compatibility points T13 must preserve

- default workflow lookup should still resolve to a `WORKFLOW.md` file at the repo/workspace root unless explicitly overridden
- CLI behavior should still accept an explicit workflow path and still default to `WORKFLOW.md`
- the bundled Linear workflow should remain unattended by default
- the default workflow text should continue to communicate no-human-follow-up behavior, blocker-only early exit, and retry-resume semantics
- `linear_graphql` should remain the single dynamic tool surface unless a separate compatibility decision adds more tools
- the tool contract should stay strict about arguments and stable about error shapes
- the existing missing-auth wording and config key names should remain recognizable to users
- blank workflow prompt bodies should still fall back to the short Linear issue template, not an empty prompt
- reload-on-change should keep the last good workflow if a new `WORKFLOW.md` is temporarily invalid

## Boundaries

### Linear compatibility shell

Keep these Linear-specific details outside the provider-neutral core:

- `linear_graphql` dynamic tool name, description, schema, and error wording
- Linear auth and transport handling
- Linear-specific issue/project/project-slug semantics
- any prompt text that explicitly names Linear, the unattended orchestration posture, or Linear workflow instructions
- the bundled `WORKFLOW.md` content for the Linear default bundle
- CLI messaging that references the guardrails acknowledgement and the workflow bundle

### Core

Keep these provider-neutral or reusable:

- workflow file path resolution and reload mechanics
- front matter parsing and markdown body extraction
- prompt rendering and retry interpolation
- config schema parsing, normalization, and validation infrastructure
- workspace root canonicalization and sandbox defaults
- workflow caching and last-known-good behavior

## Risks / Unknowns for Go

- The original behavior is byte-sensitive in several places. The bundled `WORKFLOW.md` body, default prompt template, and dynamic-tool error messages are likely compatibility surfaces.
- `Workflow.load/1` accepts prompt-only files and unterminated front matter. It is not safe to assume the Go version can require a fully closed YAML block.
- The original runtime keeps the last valid workflow after a reload failure. A Go port that eagerly fails hard on parse errors would change operational behavior.
- The bundled workflow uses `approval_policy: never` plus `thread_sandbox: workspace-write` and a workspace-write turn sandbox. That posture likely matters for unattended execution and should not be loosened accidentally.
- It is not yet clear whether Go should preserve the exact `linear_graphql` tool-only surface or whether future compatibility layers will add more tools later. For T13, the safe assumption is to keep the one-tool contract.
- The original product distinguishes workflow/config concerns from orchestration concerns, but the `WORKFLOW.md` file carries both prompt text and runtime config. Go should preserve that split rather than inventing a second configuration source.
