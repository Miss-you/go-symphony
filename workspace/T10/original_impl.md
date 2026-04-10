# T10 Original Implementation Notes

## Scope

T10 needs to freeze the tracker surface that the runtime actually depends on and preserve the memory-backed local/test path that lets Symphony run without Linear.

The important boundary question is not "what tracker methods exist anywhere in the app", but "which reads does the orchestrator and worker lifecycle actually need". The reference implementation already answers that:

- orchestration is driven by tracker reads
- tracker writes still exist in the compatibility boundary
- the memory adapter is a local/test adapter, not just a fake stub

## Source Evidence

- `elixir/lib/symphony_elixir/tracker.ex:8-37` defines the adapter boundary. It exposes three read callbacks plus two write callbacks: `fetch_candidate_issues/0`, `fetch_issues_by_states/1`, `fetch_issue_states_by_ids/1`, `create_comment/2`, and `update_issue_state/2`.
- `elixir/lib/symphony_elixir/orchestrator.ex:224-295` shows the main poll loop. It fetches candidate issues for dispatch and fetches running issue states by ID for reconcile.
- `elixir/lib/symphony_elixir/orchestrator.ex:660-880` shows additional runtime use of tracker reads for dispatch revalidation, retry polling, and startup terminal-workspace cleanup.
- `elixir/lib/symphony_elixir/agent_runner.ex:81-131` shows worker-side continuation checks. The runner defaults its state refresh dependency to `Tracker.fetch_issue_states_by_ids/1` and uses that refresh to decide whether to continue another turn.
- `elixir/lib/symphony_elixir/linear/client.ex:12-20,57-93,448-573` shows the normalized Linear issue shape and how it is decoded from GraphQL.
- `elixir/lib/symphony_elixir/linear/adapter.ex:40-90` shows that the Linear adapter forwards the read methods to the client and keeps the write mutations separate.
- `elixir/lib/symphony_elixir/tracker/memory.ex:10-47` shows the memory adapter behavior used for local development and tests.
- `elixir/test/symphony_elixir/extensions_test.exs:184-205` and `elixir/test/symphony_elixir/core_test.exs:353-515` pin down the runtime behaviors that matter for parity: adapter selection, memory-side effects, reconcile, and running-item cleanup.

## Current Symphony Model

### Tracker boundary

The Elixir `Tracker` module is broader than the Go core needs. It is a compatibility boundary that includes both reads and writes, but the orchestrator itself only consumes the read side.

The read surface currently used by runtime code is:

- `fetch_candidate_issues/0` for normal poll/dispatch cycles
- `fetch_issue_states_by_ids/1` for reconcile and dispatch revalidation
- `fetch_issues_by_states/1` for startup cleanup of terminal issue workspaces

The write side remains in the same boundary in Elixir, but that is a compatibility concern, not a requirement of the orchestrator's read path.

### Normalized issue shape

`SymphonyElixir.Linear.Issue` is the normalized runtime item that the orchestrator works with. It carries:

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

The client constructs that shape from Linear GraphQL responses. A few details are operationally important:

- `labels` are lowercased
- `blocked_by` is derived only from `inverseRelations` entries whose relation type is `blocks`
- `assigned_to_worker` is computed from the configured assignee filter, with `nil` meaning unrestricted routing
- `fetch_issue_states_by_ids/1` preserves the requested ID order in its response
- both `fetch_issues_by_states/1` and `fetch_issue_states_by_ids/1` return `{:ok, []}` for empty input

### Memory adapter behavior

`SymphonyElixir.Tracker.Memory` is the local/test path.

Its reads are simple and deterministic:

- `fetch_candidate_issues/0` returns the configured issues from `:memory_tracker_issues`
- `fetch_issues_by_states/1` normalizes incoming state names, matches case-insensitively, and filters the configured issues
- `fetch_issue_states_by_ids/1` filters the configured issues by exact ID match
- non-`Issue` values in `:memory_tracker_issues` are ignored

Its write methods are not no-ops in the abstract sense. They send notifications to `:memory_tracker_recipient` when configured:

- `create_comment/2` emits `{:memory_tracker_comment, issue_id, body}`
- `update_issue_state/2` emits `{:memory_tracker_state_update, issue_id, state_name}`

That matters because the memory adapter is a test/local integration point, not just a mock that answers reads.

## Parity Constraints

- The orchestrator-facing tracker surface must stay read-focused, because the runtime only depends on reads to poll, reconcile, revalidate, and clean up.
- `fetch_candidate_issues/0` is the poll entrypoint. If it is missing or behaves differently, the dispatch loop changes.
- `fetch_issue_states_by_ids/1` is required both for running-item reconcile and for worker continuation checks. It is not just a convenience API.
- `fetch_issues_by_states/1` is required for startup cleanup of terminal workspaces. Omitting it would lose existing cleanup behavior.
- The normalized issue shape must preserve `blocked_by` and `assigned_to_worker`, because the orchestrator uses those semantics to decide whether an item can run.
- State normalization must remain forgiving. The memory adapter and Linear client both accept state lists and compare them in a normalized way.
- Empty-input behavior is part of the contract. The Elixir client returns empty lists rather than errors for empty ID/state queries.
- The memory adapter must remain able to drive tests and local closure without Linear, including the side effects that tests currently observe through `memory_tracker_recipient`.

## Risks For Go Port

- If the Go core copies the Elixir `Tracker` module shape too literally, it will inherit write methods into a boundary that the runtime does not need.
- If the Go normalized item omits `blocked_by` or `assigned_to_worker`, dispatch gating will diverge from current Symphony behavior.
- If the Go memory adapter is too minimal, the local/test path will no longer exercise the same runtime flows the Elixir suite uses today.
- If `fetch_issues_by_states/1` is treated as optional, startup terminal cleanup parity will slip.
- If empty-input handling or state normalization changes, the retry/reconcile paths will become more brittle than the reference implementation.
- The current Elixir model mixes tracker reads and writes in one compatibility module. The Go port needs to separate what the core truly depends on from what remains a provider/toolbridge concern, or T10 will freeze the wrong abstraction.

