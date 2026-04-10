# T06 Original Implementation

## Reference Scope

This summary is based on direct inspection of:

- `/Users/lihui/Documents/GitHub/symphony/elixir/lib/symphony_elixir/orchestrator.ex`
- `/Users/lihui/Documents/GitHub/symphony/elixir/test/symphony_elixir/core_test.exs`
- `/Users/lihui/Documents/GitHub/symphony/elixir/test/symphony_elixir/orchestrator_status_test.exs`

The goal here is to capture the runtime behavior `T06` must preserve, not to copy Elixir process wiring into Go.

## Runtime State Owned By The Elixir Orchestrator

`SymphonyElixir.Orchestrator.State` owns all mutable scheduling state in one place:

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

That matches the approved Go design: the orchestrator is the sole runtime state owner, while workers report facts back.

## Polling And Refresh Semantics

Startup behavior:

- `init/1` reads config, sets `next_poll_due_at_ms` to `now`, runs terminal-workspace cleanup, and schedules an immediate tick.
- `handle_info({:tick, token}, ...)` and `handle_info(:tick, ...)` mark polling as in progress, clear the next due time, notify the dashboard, and schedule `:run_poll_cycle`.
- `handle_info(:run_poll_cycle, ...)` refreshes config, reconciles + dispatches, schedules the next tick for `poll_interval_ms`, clears `poll_check_in_progress`, and notifies the dashboard again.

Two source-faithful details matter:

1. The orchestrator deliberately exposes a short “checking now” transition before the main poll body finishes. The tests assert that snapshots can show `checking?: true` shortly after startup.
2. Manual refresh is coalesced. `handle_call(:request_refresh, ...)` queues an immediate poll only when a poll is not already in progress and no already-due tick exists. Superseded tick tokens are ignored.

## Dispatch Ordering And Gating

Candidate selection is not first-come-first-served.

`sort_issues_for_dispatch/1` sorts by:

1. priority rank (`1..4`, then missing/unknown last)
2. `created_at` oldest first
3. identifier/id as stable tie-breaker

`should_dispatch_issue?/4` only allows dispatch when all of the following are true:

- the issue has `id`, `identifier`, `title`, and `state`
- it is routed to this worker (`assigned_to_worker` true or absent)
- its state is active and not terminal
- a `Todo` issue is not blocked by any non-terminal blocker
- it is not already claimed
- it is not already running
- a global slot is available
- the per-state concurrency limit allows it
- a worker host slot is available

Before spawn, the orchestrator re-fetches the issue by id. If the refreshed issue is no longer active/routable/visible, the stale dispatch is skipped.

## Reconcile Behavior

Every poll begins with reconciliation:

1. restart stalled running issues
2. refresh currently running issue states from the tracker
3. stop any running issue that is no longer valid

The stop rules are explicit:

- terminal state: stop the run and clean up the workspace
- no longer routed to this worker: stop the run without cleanup
- active state: keep the run, but replace the embedded issue state with the refreshed issue
- non-active non-terminal state: stop the run without cleanup
- missing from the refresh response entirely: stop the run without cleanup

The tests in `core_test.exs` pin two critical parity rules:

- active refreshed issues keep their claim and update the embedded runtime issue state
- reassigned issues are removed from both `running` and `claimed`, and the worker process is stopped

## Retry And Continuation Rules

The Elixir orchestrator distinguishes completion, failure, and retry delivery clearly.

### Worker Exit

When a worker process exits normally:

- the run is removed from `running`
- the item is added to `completed`
- a continuation retry is scheduled with attempt `1`
- the continuation delay is fixed at `1_000ms`

This is not treated as failure. It is an active-state continuation check.

When a worker exits abnormally:

- the next retry attempt comes from the running entry's previous retry attempt plus one, or falls back to first failure
- the retry error text records the exit reason
- the delay is exponential backoff starting at `10_000ms`
- the delay is capped by `config.agent.max_retry_backoff_ms`

The tests pin concrete expectations:

- first abnormal exit retries in roughly `10s`
- a previous attempt of `2` leads to attempt `3` and roughly `40s` delay

### Retry Queue Ownership

`retry_attempts` is owned by the orchestrator, not workers.

Each retry entry stores:

- `attempt`
- `timer_ref`
- `retry_token`
- `due_at_ms`
- `identifier`
- `error`
- `worker_host`
- `workspace_path`

`retry_token` is important because stale timer deliveries are ignored. Tests explicitly verify that an old retry timer message must not consume a newer retry entry.

### Retry Delivery

When a retry fires:

- the orchestrator removes the matching retry entry only if the retry token matches
- it fetches current candidate issues again
- if the issue is terminal, it cleans up workspace and releases the claim
- if the issue is still active/routable and slots are available, it dispatches again
- otherwise it reschedules another retry with updated error context

## Stall Recovery

Stall handling is part of orchestrator ownership, not a dashboard concern.

- `reconcile_stalled_running_issues/1` compares the last Codex timestamp, or the run start time when no Codex timestamp exists, against `config.codex.stall_timeout_ms`
- a stalled run is terminated and moved back into retry scheduling
- the retry error records the stall duration

The status tests assert that stale activity older than the configured timeout causes:

- the worker to stop
- the run to leave `running`
- a retry entry with attempt `1`
- failure-style backoff timing

## Snapshot Contract

`handle_call(:snapshot, ...)` is the source of truth for observability.

It returns:

- `running`
- `retrying`
- `codex_totals`
- `rate_limits`
- `polling`

### Running Entries

Each running snapshot entry includes:

- `issue_id`
- `identifier`
- `state`
- `worker_host`
- `workspace_path`
- `session_id`
- `codex_app_server_pid`
- `codex_input_tokens`
- `codex_output_tokens`
- `codex_total_tokens`
- `turn_count`
- `started_at`
- `last_codex_timestamp`
- `last_codex_message`
- `last_codex_event`
- `runtime_seconds`

### Retry Entries

Each retry snapshot entry includes:

- `issue_id`
- `attempt`
- `due_in_ms`
- `identifier`
- `error`
- `worker_host`
- `workspace_path`

### Polling Projection

The snapshot exposes:

- `checking?`
- `next_poll_in_ms`
- `poll_interval_ms`

Tests assert both the waiting state and the active “checking now” state.

## Source-Faithful Porting Implications

The Go port needs to preserve these semantics:

- single-owner mutable scheduling state
- source-faithful dispatch ordering and gating
- explicit reconcile before new dispatch
- distinct continuation retry versus failure retry behavior
- stale retry delivery protection
- stall recovery as orchestrator logic
- snapshot as the only observability truth source

It should not blindly copy these Elixir-specific mechanisms:

- GenServer message handling
- process refs and monitor refs in the public runtime contract
- direct dependency on Linear-shaped issue structs
- workspace cleanup implementation details that belong to later Go tasks
