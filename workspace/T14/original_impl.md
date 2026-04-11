# T14 Original Symphony Flow

## Scope

This note captures how the Elixir Symphony repository wires an end-to-end run together for T14, from tracker polling through issue selection, workspace lifecycle, Codex session execution, tool calls, retries, cleanup, and observability.

The focus is runtime behavior that Go T14 needs to preserve. It does not cover UI styling, unrelated CLI surface, or speculative design.

## Source Files Inspected

- [`/Users/apple/Documents/Github/symphony/elixir/lib/symphony_elixir.ex`](/Users/apple/Documents/Github/symphony/elixir/lib/symphony_elixir.ex) - application entrypoint and OTP supervision tree
- [`/Users/apple/Documents/Github/symphony/elixir/lib/symphony_elixir/orchestrator.ex`](/Users/apple/Documents/Github/symphony/elixir/lib/symphony_elixir/orchestrator.ex) - polling, dispatch, retries, reconciliation, snapshot state
- [`/Users/apple/Documents/Github/symphony/elixir/lib/symphony_elixir/tracker.ex`](/Users/apple/Documents/Github/symphony/elixir/lib/symphony_elixir/tracker.ex) - tracker adapter boundary
- [`/Users/apple/Documents/Github/symphony/elixir/lib/symphony_elixir/tracker/memory.ex`](/Users/apple/Documents/Github/symphony/elixir/lib/symphony_elixir/tracker/memory.ex) - in-memory tracker implementation
- [`/Users/apple/Documents/Github/symphony/elixir/lib/symphony_elixir/linear/client.ex`](/Users/apple/Documents/Github/symphony/elixir/lib/symphony_elixir/linear/client.ex) - Linear GraphQL polling and normalization
- [`/Users/apple/Documents/Github/symphony/elixir/lib/symphony_elixir/linear/adapter.ex`](/Users/apple/Documents/Github/symphony/elixir/lib/symphony_elixir/linear/adapter.ex) - tracker writes for comments and issue state updates
- [`/Users/apple/Documents/Github/symphony/elixir/lib/symphony_elixir/linear/issue.ex`](/Users/apple/Documents/Github/symphony/elixir/lib/symphony_elixir/linear/issue.ex) - normalized issue struct
- [`/Users/apple/Documents/Github/symphony/elixir/lib/symphony_elixir/workspace.ex`](/Users/apple/Documents/Github/symphony/elixir/lib/symphony_elixir/workspace.ex) - workspace creation, hooks, removal
- [`/Users/apple/Documents/Github/symphony/elixir/lib/symphony_elixir/agent_runner.ex`](/Users/apple/Documents/Github/symphony/elixir/lib/symphony_elixir/agent_runner.ex) - per-issue execution loop
- [`/Users/apple/Documents/Github/symphony/elixir/lib/symphony_elixir/codex/app_server.ex`](/Users/apple/Documents/Github/symphony/elixir/lib/symphony_elixir/codex/app_server.ex) - Codex stdio JSON-RPC session management
- [`/Users/apple/Documents/Github/symphony/elixir/lib/symphony_elixir/codex/dynamic_tool.ex`](/Users/apple/Documents/Github/symphony/elixir/lib/symphony_elixir/codex/dynamic_tool.ex) - client-side `linear_graphql` tool bridge
- [`/Users/apple/Documents/Github/symphony/elixir/lib/symphony_elixir/workflow.ex`](/Users/apple/Documents/Github/symphony/elixir/lib/symphony_elixir/workflow.ex) - `WORKFLOW.md` loading
- [`/Users/apple/Documents/Github/symphony/elixir/lib/symphony_elixir/workflow_store.ex`](/Users/apple/Documents/Github/symphony/elixir/lib/symphony_elixir/workflow_store.ex) - workflow cache and file polling
- [`/Users/apple/Documents/Github/symphony/elixir/lib/symphony_elixir/prompt_builder.ex`](/Users/apple/Documents/Github/symphony/elixir/lib/symphony_elixir/prompt_builder.ex) - prompt rendering
- [`/Users/apple/Documents/Github/symphony/elixir/lib/symphony_elixir/config.ex`](/Users/apple/Documents/Github/symphony/elixir/lib/symphony_elixir/config.ex) - runtime config access and validation
- [`/Users/apple/Documents/Github/symphony/elixir/lib/symphony_elixir/config/schema.ex`](/Users/apple/Documents/Github/symphony/elixir/lib/symphony_elixir/config/schema.ex) - defaults and config schema
- [`/Users/apple/Documents/Github/symphony/elixir/lib/symphony_elixir/log_file.ex`](/Users/apple/Documents/Github/symphony/elixir/lib/symphony_elixir/log_file.ex) - rotating log handler
- [`/Users/apple/Documents/Github/symphony/elixir/lib/symphony_elixir/status_dashboard.ex`](/Users/apple/Documents/Github/symphony/elixir/lib/symphony_elixir/status_dashboard.ex) - terminal observability render path
- [`/Users/apple/Documents/Github/symphony/elixir/lib/symphony_elixir_web/observability_pubsub.ex`](/Users/apple/Documents/Github/symphony/elixir/lib/symphony_elixir_web/observability_pubsub.ex) - dashboard update pubsub
- [`/Users/apple/Documents/Github/symphony/elixir/lib/symphony_elixir_web/presenter.ex`](/Users/apple/Documents/Github/symphony/elixir/lib/symphony_elixir_web/presenter.ex) - snapshot projections for API and dashboard
- [`/Users/apple/Documents/Github/symphony/elixir/lib/symphony_elixir_web/controllers/observability_api_controller.ex`](/Users/apple/Documents/Github/symphony/elixir/lib/symphony_elixir_web/controllers/observability_api_controller.ex) - JSON observability API
- [`/Users/apple/Documents/Github/symphony/elixir/lib/symphony_elixir_web/live/dashboard_live.ex`](/Users/apple/Documents/Github/symphony/elixir/lib/symphony_elixir_web/live/dashboard_live.ex) - live dashboard subscriber
- [`/Users/apple/Documents/Github/symphony/elixir/lib/symphony_elixir_web/router.ex`](/Users/apple/Documents/Github/symphony/elixir/lib/symphony_elixir_web/router.ex) - observability routes
- Supporting tests used as behavior evidence:
  - [`/Users/apple/Documents/Github/symphony/elixir/test/symphony_elixir/app_server_test.exs`](/Users/apple/Documents/Github/symphony/elixir/test/symphony_elixir/app_server_test.exs)
  - [`/Users/apple/Documents/Github/symphony/elixir/test/symphony_elixir/orchestrator_status_test.exs`](/Users/apple/Documents/Github/symphony/elixir/test/symphony_elixir/orchestrator_status_test.exs)
  - [`/Users/apple/Documents/Github/symphony/elixir/test/symphony_elixir/workspace_and_config_test.exs`](/Users/apple/Documents/Github/symphony/elixir/test/symphony_elixir/workspace_and_config_test.exs)
  - [`/Users/apple/Documents/Github/symphony/elixir/test/symphony_elixir/dynamic_tool_test.exs`](/Users/apple/Documents/Github/symphony/elixir/test/symphony_elixir/dynamic_tool_test.exs)
  - [`/Users/apple/Documents/Github/symphony/elixir/test/symphony_elixir/observability_pubsub_test.exs`](/Users/apple/Documents/Github/symphony/elixir/test/symphony_elixir/observability_pubsub_test.exs)

## Runtime Flow

At startup, [`symphony_elixir.ex`](/Users/apple/Documents/Github/symphony/elixir/lib/symphony_elixir.ex) wires the system as:

```
Phoenix.PubSub
Task.Supervisor
WorkflowStore
Orchestrator
HttpServer
StatusDashboard
```

The important part for T14 is the orchestrator and its dependencies:

1. `WorkflowStore` loads `WORKFLOW.md` and keeps re-reading it on a one-second poll so config and prompt changes can take effect without a restart.
2. `Orchestrator.init/1` does a startup cleanup pass for terminal issues, then schedules the first poll tick immediately.
3. Each poll tick marks polling as in progress, emits a dashboard update, and starts the actual poll cycle after a short delay so the UI can render the transition.
4. `maybe_dispatch/1` validates config, asks the tracker for candidate issues, and dispatches as many as slots allow.
5. Each dispatched issue is revalidated by ID before work starts. If it is no longer active, visible, or routable, dispatch is skipped.
6. A dispatched issue runs in a `Task.Supervisor` child through `AgentRunner.run/3`, while the orchestrator keeps the running entry in memory and monitors the task PID.
7. The runner creates the workspace, runs hooks, starts a Codex app-server session, loops over turns, and always runs `after_run` on exit.
8. Codex stream messages are forwarded back to the orchestrator as `{:codex_worker_update, issue_id, message}`.
9. The orchestrator projects those updates into running state, token totals, rate-limit snapshots, and retry queue entries.
10. When the task exits normally or abnormally, the orchestrator decides whether to continue, retry, or clean up.

### Tracker polling and refresh

`Orchestrator` fetches candidate issues via `Tracker.fetch_candidate_issues/0`, which is Linear-backed by default and memory-backed in test/local mode.

Selection rules in [`orchestrator.ex`](/Users/apple/Documents/Github/symphony/elixir/lib/symphony_elixir/orchestrator.ex) are:

- issue must have binary `id`, `identifier`, `title`, and `state`
- issue must be routed to this worker (`assigned_to_worker` true, unless the tracker leaves that unset)
- issue state must be in configured active states
- issue state must not be terminal
- `todo` issues are blocked if any blocker is still non-terminal
- the orchestrator must still have global and per-state capacity

Before each dispatch, the issue is refreshed again by ID. This avoids starting stale work after the tracker changed between the poll and the launch.

### Linear issue selection and routing

[`linear/client.ex`](/Users/apple/Documents/Github/symphony/elixir/lib/symphony_elixir/linear/client.ex) builds the polling query around:

- configured `tracker.project_slug`
- configured active states
- an assignee filter, if configured

Assignee routing supports three modes:

- no assignee configured, which accepts all issues
- a specific assignee ID or other configured match value
- `me`, which resolves the current Linear viewer ID first

The client also normalizes each issue into [`Linear.Issue`](/Users/apple/Documents/Github/symphony/elixir/lib/symphony_elixir/linear/issue.ex) with:

- labels lowercased
- blockers extracted from `inverseRelations` of type `blocks`
- `assigned_to_worker` derived from assignee matching
- `created_at` and `updated_at` parsed into `DateTime`

Paging matters here: the client fetches issues in pages of 50 and merges them back into stable order.

### Workspace setup and cleanup

[`workspace.ex`](/Users/apple/Documents/Github/symphony/elixir/lib/symphony_elixir/workspace.ex) creates one workspace per issue identifier under the configured workspace root.

Important behavior:

- path is canonicalized and must stay inside the configured workspace root
- the workspace root itself is rejected
- symlink escapes are rejected
- an existing directory is reused instead of being destroyed
- only temporary artifacts like `.elixir_ls` and `tmp` are cleaned on reuse
- if the path exists as a file, it is removed and replaced with a directory
- `after_create` runs only on first creation
- `before_run` is hard-failing
- `after_run` is best-effort and failures are ignored
- `before_remove` is run before deletion, but its failure is ignored

Cleanup happens in two places:

- startup terminal cleanup removes workspaces for all terminal issues
- terminal issue reconciliation and terminal retry completion remove the issue workspace again if needed

### Runner and executor

[`agent_runner.ex`](/Users/apple/Documents/Github/symphony/elixir/lib/symphony_elixir/agent_runner.ex) is the single-issue worker.

Its flow is:

1. create the workspace
2. run `before_run`
3. start a Codex session
4. run turns until Codex says the turn is complete or an error happens
5. run `after_run` in an `after` block

The runner keeps re-entering Codex turns while the issue is still active and the configured `agent.max_turns` has not been reached.

Continuation turns are special:

- first turn uses the rendered workflow prompt
- later turns use a hardcoded continuation prompt that tells Codex to resume from the current workspace and not restart from scratch

After each normal turn, the runner refreshes issue state by ID. If the issue is still active, it continues; if it is no longer active, the runner stops and hands control back to the orchestrator.

### Codex app-server sessions and events

[`codex/app_server.ex`](/Users/apple/Documents/Github/symphony/elixir/lib/symphony_elixir/codex/app_server.ex) talks to the Codex app-server over stdio as JSON-RPC 2.0.

Session bootstrap:

- validate the workspace is inside the configured root
- start the configured Codex command through `bash -lc`
- send `initialize`
- send `initialized`
- start a thread with `thread/start`
- pass the configured `approvalPolicy`, `thread_sandbox`, `cwd`, and the dynamic tool list

Per turn:

- send `turn/start` with the thread ID, prompt, cwd, title, approval policy, and turn sandbox policy
- wait for response lines from the port
- emit `session_started` when the turn starts

The stream parser handles these event families:

- `turn/completed`
- `turn/failed`
- `turn/cancelled`
- tool and approval request methods under `item/*`
- generic notifications and unknown JSON payloads
- malformed non-JSON stream lines

Tool and approval handling is important for T14:

- `item/tool/call` is executed client-side through the dynamic tool executor
- `item/tool/requestUserInput` is auto-answered when possible, otherwise the turn fails with `turn_input_required`
- session command approvals and file-change approvals are auto-approved when approval policy is `never`
- otherwise approval requests surface as `approval_required`
- if Codex requests a tool the bridge does not understand, the turn emits `unsupported_tool_call`

The runner forwards every emitted message to the orchestrator, including the port PID and any usage metadata it can extract.

### Linear GraphQL tool bridge

[`codex/dynamic_tool.ex`](/Users/apple/Documents/Github/symphony/elixir/lib/symphony_elixir/codex/dynamic_tool.ex) exposes exactly one client-side tool: `linear_graphql`.

Behavior:

- accepts either a raw GraphQL string or an object with `query` and optional `variables`
- trims and validates the query
- forwards the call to `Linear.Client.graphql/3`
- returns the raw Linear response JSON encoded inside `contentItems`
- marks the response as failed if the GraphQL payload contains a non-empty `errors` list
- returns structured error payloads for invalid arguments, missing query, invalid variables, or Linear API failures

This is a thin bridge, not a higher-level Linear SDK.

### Workflow prompt

[`workflow.ex`](/Users/apple/Documents/Github/symphony/elixir/lib/symphony_elixir/workflow.ex) loads `WORKFLOW.md` from the current working directory unless an override path is set.

[`workflow_store.ex`](/Users/apple/Documents/Github/symphony/elixir/lib/symphony_elixir/workflow_store.ex) caches the last known good workflow and reloads it when the file changes.

[`prompt_builder.ex`](/Users/apple/Documents/Github/symphony/elixir/lib/symphony_elixir/prompt_builder.ex) renders the prompt using `Solid` with the issue struct as template data.

Important details:

- empty prompt text falls back to `Config.workflow_prompt()`
- empty workflow prompt falls back to a default issue-centric prompt template
- DateTime values are converted to ISO8601 before rendering
- continuation turns do not re-render the workflow prompt; they use the hardcoded continuation guidance instead

### Observable state and events

The observable runtime is built from the orchestrator snapshot and the codex updates it receives.

[`orchestrator.ex`](/Users/apple/Documents/Github/symphony/elixir/lib/symphony_elixir/orchestrator.ex) keeps these public state projections:

- running issues
- retrying issues
- codex token totals
- rate limits
- polling status (`checking?`, next poll ETA, poll interval)

[`presenter.ex`](/Users/apple/Documents/Github/symphony/elixir/lib/symphony_elixir_web/presenter.ex) turns that into JSON for the API and dashboard.

[`observability_pubsub.ex`](/Users/apple/Documents/Github/symphony/elixir/lib/symphony_elixir_web/observability_pubsub.ex) broadcasts update events on the `observability:dashboard` topic, and [`dashboard_live.ex`](/Users/apple/Documents/Github/symphony/elixir/lib/symphony_elixir_web/live/dashboard_live.ex) subscribes to those updates.

Observed codex message state includes:

- `session_started`
- `turn_completed`
- `turn_failed`
- `turn_cancelled`
- `notification`
- `approval_auto_approved`
- `approval_required`
- `tool_call_completed`
- `tool_call_failed`
- `unsupported_tool_call`
- `tool_input_auto_answered`
- `turn_input_required`
- `startup_failed`
- `other_message`
- `malformed`

Token accounting is incremental, not a simple overwrite:

- the orchestrator tracks the last reported totals from Codex
- deltas are added only when a new total is greater than or equal to the previous reported total
- turn-completed payloads can also contribute usage if they include it directly

The dashboard and API therefore show both the current running state and the queued backoff state, not just the latest task result.

## Retry/Cleanup Semantics

Normal completion does not mean final completion.

When a worker process exits normally, the orchestrator:

1. records session completion totals
2. marks the issue as completed internally
3. schedules a continuation retry after 1 second
4. waits to see whether the issue is still active on the next poll

If the issue is still active on that continuation retry, it is dispatched again with the next turn. If it is no longer active, the orchestrator releases the claim instead.

Failure retry behavior is exponential backoff:

- base delay is 10 seconds
- delay doubles by attempt number
- delay is capped by `agent.max_retry_backoff_ms`
- retry metadata preserves issue identifier and last error across reschedules

Stall behavior is separate from worker exit behavior:

- a stalled worker is one with no Codex activity past `codex.stall_timeout_ms`
- the orchestrator terminates the task and schedules a retry with backoff
- last activity time comes from the latest Codex event or the turn start time

Cleanup rules:

- terminal issue state during reconciliation removes the workspace and stops the worker
- issue disappearing from the tracker releases the claim without terminal cleanup
- non-terminal state changes stop the worker but do not remove the workspace
- startup cleanup removes workspaces for all terminal issues before the first dispatch loop

Workspace removal itself is guarded:

- the path must still validate inside the workspace root
- `before_remove` runs only when the path is a directory
- delete failures are not escalated through the cleanup hook path

## Linear/Memory Implications

The tracker contract is intentionally tiny:

- `fetch_candidate_issues/0`
- `fetch_issues_by_states/1`
- `fetch_issue_states_by_ids/1`
- `create_comment/2`
- `update_issue_state/2`

Linear mode is the real integration. Memory mode is a test/local adapter that keeps the same contract but uses application env instead of HTTP:

- issues come from `:memory_tracker_issues`
- write operations emit test events to `:memory_tracker_recipient`
- it is useful for replaying selection and reconciliation logic without Linear

This means Go T14 should keep the adapter boundary clean. The orchestrator should not hardcode Linear assumptions into the core dispatch loop.

## Parity Requirements for Go T14

The original flow suggests these must stay aligned in Go:

- poll on a cadence, then reconcile running issues before selecting new ones
- refresh tracker state by issue ID before dispatch and during continuation checks
- preserve active-state vs terminal-state routing logic
- preserve assignee routing, including `me` resolution semantics
- preserve `todo` blocker filtering
- preserve per-state concurrency limits plus the global agent limit
- preserve deterministic issue ordering by priority, then creation time, then identifier
- preserve workspace path safety, reuse behavior, and hook semantics
- preserve the distinction between `before_run` hard failure and `after_run` best-effort cleanup
- preserve the Codex app-server session lifecycle: initialize, thread start, turn start, receive stream, close session
- preserve the event vocabulary surfaced to the orchestrator and dashboard
- preserve `linear_graphql` as the only client-side dynamic tool in this path
- preserve continuation turns after normal completion while the issue remains active
- preserve exponential backoff and stalled-run restart behavior
- preserve startup cleanup of terminal issue workspaces
- preserve public observability shape: running, retrying, token totals, rate limits, polling
- preserve the pubsub refresh path used by the dashboard and API

For T14 specifically, the riskiest parity gap would be a runtime that can start a session but loses the restart/refresh contract, or one that can poll Linear but cannot keep the running entry and observability state coherent across turns.

## Risks/Unknowns

- The Codex app-server event parser is broad but still version-sensitive. Future Codex releases may introduce methods that need new branches.
- I did not exhaustively inspect every dashboard formatting helper, so exact text rendering is not the main source of parity risk here.
- The dynamic tool bridge currently exposes only `linear_graphql`. Any extra tool support would need a new contract, not just a new query.
- `WORKFLOW.md` is live data. The Go side needs a clear reload story if it wants the same no-restart behavior as `WorkflowStore`.
- The memory tracker is intentionally minimal. If Go T14 relies on it, the test harness must preserve the same adapter semantics even though it does not hit Linear.
