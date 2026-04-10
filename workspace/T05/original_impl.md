# T05 Original Implementation Notes

## Scope

This task needs a provider-neutral domain model that can carry the runtime facts Symphony already depends on:

- one dispatchable work item shape
- runtime bookkeeping for active runs, retries, completion, and polling
- snapshot data for observability and dashboards
- a small blocker representation for dispatch gating

The reference implementation is still Linear-shaped in several places, but the orchestrator already treats the runtime as a generic scheduling loop once the issue has been normalized.

## Relevant Source Evidence

- `elixir/lib/symphony_elixir/linear/client.ex`
- `elixir/lib/symphony_elixir/linear/issue.ex`
- `elixir/lib/symphony_elixir/orchestrator.ex`
- `elixir/lib/symphony_elixir/agent_runner.ex`
- `elixir/lib/symphony_elixir/codex/app_server.ex`
- `elixir/lib/symphony_elixir/workflow_store.ex`
- `elixir/lib/symphony_elixir/status_dashboard.ex`
- `elixir/lib/symphony_elixir_web/presenter.ex`
- `elixir/test/symphony_elixir/workspace_and_config_test.exs`
- `elixir/test/symphony_elixir/core_test.exs`
- `elixir/test/symphony_elixir/orchestrator_status_test.exs`

The tests are important because they pin down behavior that is easy to miss from the code alone: dispatch eligibility, blocker handling, snapshot contents, retry bookkeeping, and prompt input rendering.

## Current Symphony Model

### Work item shape

The normalized issue type is `SymphonyElixir.Linear.Issue`. It currently carries:

- `id`
- `identifier`
- `title`
- `description`
- `priority`
- `state`
- `branch_name`
- `url`
- `assignee_id`
- `blocked_by`
- `labels`
- `assigned_to_worker`
- `created_at`
- `updated_at`

`labels` are normalized to lowercase strings. `blocked_by` is a list of lightweight blocker maps with `id`, `identifier`, and `state`. The module doc explicitly says this is the normalized Linear issue representation used by the orchestrator.

### Linear-specific mix-ins

`linear/client.ex` is where provider details leak into the normalized work item:

- GraphQL fields are Linear-specific: `priority`, `branchName`, `createdAt`, `updatedAt`, `inverseRelations`, `assignee`, `labels`
- `assigned_to_worker` is derived from the Linear assignee filter and the viewer identity
- `blocked_by` is extracted only from `inverseRelations` entries whose relation type is `blocks`
- the client is still responsible for GraphQL pagination, error classification, and assignee routing

This means the work item shape is already mostly generic, but the acquisition path is not.

### Orchestrator state

`Orchestrator.State` is the real mutable runtime owner. Its fields show the minimum runtime model Symphony needs:

- `poll_interval_ms`
- `max_concurrent_agents`
- `next_poll_due_at_ms`
- `poll_check_in_progress`
- `tick_timer_ref`
- `tick_token`
- `running`
- `completed`
- `claimed`
- `retry_attempts`
- `codex_totals`
- `codex_rate_limits`

The `running` map stores per-item runtime metadata, not just the work item itself. In the current implementation each running entry can contain:

- `pid`
- `ref`
- `identifier`
- `issue`
- `worker_host`
- `workspace_path`
- `session_id`
- `last_codex_message`
- `last_codex_timestamp`
- `last_codex_event`
- Codex token counters and last-reported counters
- `turn_count`
- `retry_attempt`
- `started_at`
- `codex_app_server_pid`

`retry_attempts` stores scheduled retries with `attempt`, `timer_ref`, `retry_token`, `due_at_ms`, `identifier`, `error`, `worker_host`, and `workspace_path`.

### Snapshot model

`Orchestrator.handle_call(:snapshot, ...)` is the canonical projection point. The snapshot returned to observability contains:

- `running`
- `retrying`
- `codex_totals`
- `rate_limits`
- `polling`

Each running entry in the snapshot exposes:

- `issue_id`
- `identifier`
- `state`
- `worker_host`
- `workspace_path`
- `session_id`
- `codex_app_server_pid`
- Codex token counters
- `turn_count`
- `started_at`
- `last_codex_timestamp`
- `last_codex_message`
- `last_codex_event`
- `runtime_seconds`

Each retry entry exposes:

- `issue_id`
- `attempt`
- `due_in_ms`
- `identifier`
- `error`
- `worker_host`
- `workspace_path`

`presenter.ex` then turns that snapshot into `/api/v1/state` and `/api/v1/:issue_identifier` payloads. So the snapshot is not just a debug view; it is the source of truth for the user-facing observability surfaces.

### Retry and completion semantics

The orchestrator distinguishes several runtime outcomes:

- normal worker exit schedules a continuation retry when the issue is still active
- failure exit schedules a backoff retry
- stalled runs are restarted after `codex.stall_timeout_ms`
- terminal issues clean up their workspaces
- missing issues release claims and stop active workers

That means the runtime model needs both a current running view and a future retry queue, not just a flat list of active items.

### Polling and reconciliation

The runtime loop is explicitly polling-driven:

- `poll_check_in_progress` tracks the transition state while a poll cycle starts
- `next_poll_due_at_ms` controls the next cycle
- `reconcile_running_issues/1` refreshes running work items and removes claims when the issue disappears, becomes terminal, or is no longer routed to the worker

Dispatch eligibility depends on more than state:

- active states and terminal states come from config
- `Todo` items are blocked if any blocker is still non-terminal
- `assigned_to_worker` can exclude an issue from dispatch
- concurrency is limited globally and per state
- priority and creation time determine dispatch ordering

### Observability and workflow boundaries

`WorkflowStore` keeps the last-known-good workflow and only replaces it on a successful reload. That is important because the runtime assumes config and workflow text move together.

`StatusDashboard` and `Presenter` are projections only. They read snapshot data from the orchestrator instead of owning any independent runtime state.

## Parity Constraints

- Preserve `issue`-level compatibility at the edges, but keep the core runtime type provider-neutral.
- Keep the runtime owner singular: the orchestrator owns mutable scheduling state, and workers only report events back.
- Preserve the dispatch filters that matter today: active vs terminal states, blocker gating for `Todo`, worker assignment routing, and concurrency limits.
- Preserve the sort order: priority first, then oldest `created_at`, then identifier fallback.
- Preserve snapshot shape stability because both the HTTP API and terminal/web dashboards depend on it.
- Preserve the last-known-good config semantics; a failed reload must not partially mutate runtime-visible state.
- Keep provider-specific graph/query fields out of the new core model even if the current Linear adapter still needs them.

## Risks For Go Port

- The current `Issue` struct mixes core runtime facts with Linear acquisition details. If the Go `WorkItem` copies all of that blindly, the core will stay provider-shaped.
- `blocked_by` is semantically important even though it is only a lightweight list of maps today. Dropping it would break dispatch gating for blocked `Todo` items.
- Snapshot consumers expect token accounting and polling metadata, not just “active items”. If the new domain model is too small, observability parity will slip.
- Retry handling is stateful and tokenized. A naive “retry list” without identifiers, due times, and workspace metadata will not support the current behavior.
- The orchestrator currently uses `assigned_to_worker` as a hard dispatch gate. The Go model needs either a neutral equivalent or an explicit compatibility bridge.
- The current reference code still exposes Linear field names in the normalization path. T05 should separate those names from the core model, but it must not lose the semantics they carry.
