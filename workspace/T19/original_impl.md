# T19 Original Symphony Implementation Notes

Research-only notes for the original Elixir/OTP implementation in `/Users/lihui/Documents/GitHub/symphony`.

## 1. What exists today for Linear fetching / operational verification

The original implementation does **not** have a standalone `probe` command, but it does have a very clear Linear read path and a real live-e2e verification path.

- The main Linear read client lives in [`elixir/lib/symphony_elixir/linear/client.ex`](/Users/lihui/Documents/GitHub/symphony/elixir/lib/symphony_elixir/linear/client.ex#L1).
  - `fetch_candidate_issues/0` polls active work for the configured project slug and assignee filter.
  - `fetch_issues_by_states/1` and `fetch_issue_states_by_ids/1` are the two other read entrypoints used for reconciliation and cleanup.
  - The GraphQL selection set already includes the fields the runtime needs for eligibility and prompt rendering: `id`, `identifier`, `title`, `description`, `priority`, `state.name`, `branchName`, `url`, `assignee.id`, `labels`, `inverseRelations`, `createdAt`, and `updatedAt`.
- The normalized issue model is [`elixir/lib/symphony_elixir/linear/issue.ex`](/Users/lihui/Documents/GitHub/symphony/elixir/lib/symphony_elixir/linear/issue.ex#L1).
  - It preserves the runtime fields Symphony actually reasons about: `identifier`, `state`, `labels`, `blocked_by`, `assigned_to_worker`, and timestamps.
- The dynamic `linear_graphql` tool is implemented in [`elixir/lib/symphony_elixir/codex/dynamic_tool.ex`](/Users/lihui/Documents/GitHub/symphony/elixir/lib/symphony_elixir/codex/dynamic_tool.ex#L1).
  - This is the operator-facing escape hatch for raw Linear GraphQL during Codex sessions.
  - It accepts either a raw query string or `{query, variables}` map and returns a JSON-encoded success/failure payload.
- The live end-to-end test in [`elixir/test/symphony_elixir/live_e2e_test.exs`](/Users/lihui/Documents/GitHub/symphony/elixir/test/symphony_elixir/live_e2e_test.exs#L1) is the closest thing to an operational verification harness.
  - It creates a real Linear project and issue.
  - It runs a real Codex session.
  - It verifies a workspace side effect, a Linear comment write, and a terminal issue state transition.

Practical takeaway: if you want a first-stage “Linear read probe” in go-symphony, the original repo already shows the exact read surfaces to expose or reuse. The missing piece is a separate, operator-friendly entrypoint that isolates those reads from the full runtime.

## 2. Normal runtime flow from Linear issue to Codex app-server work

The production flow is orchestrated end-to-end in a few layers:

1. CLI startup.
   - [`elixir/lib/symphony_elixir/cli.ex`](/Users/lihui/Documents/GitHub/symphony/elixir/lib/symphony_elixir/cli.ex#L1) parses the guardrail acknowledgement flag, optional `--logs-root`, optional `--port`, and an optional workflow path.
   - It defaults to `WORKFLOW.md` when no path is passed and refuses to boot if the file is missing or startup fails.
2. Workflow loading and config parsing.
   - [`elixir/lib/symphony_elixir/workflow.ex`](/Users/lihui/Documents/GitHub/symphony/elixir/lib/symphony_elixir/workflow.ex#L1) loads `WORKFLOW.md`, splits YAML front matter from the Markdown body, and hot-reloads the current store when the file changes.
   - [`elixir/lib/symphony_elixir/config.ex`](/Users/lihui/Documents/GitHub/symphony/elixir/lib/symphony_elixir/config.ex#L1) and [`elixir/lib/symphony_elixir/config/schema.ex`](/Users/lihui/Documents/GitHub/symphony/elixir/lib/symphony_elixir/config/schema.ex#L1) provide typed config defaults, environment indirection, and semantic validation.
3. Orchestration.
   - [`elixir/lib/symphony_elixir/orchestrator.ex`](/Users/lihui/Documents/GitHub/symphony/elixir/lib/symphony_elixir/orchestrator.ex#L1) owns the mutable runtime state.
   - It polls Linear, reconciles running issues, sorts candidates, applies active/terminal state rules, and enforces concurrency.
   - On dispatch it revalidates the issue, creates/retains a workspace, and starts an agent worker.
   - When a Codex turn finishes normally, it checks whether the issue is still active and may continue for another turn until `agent.max_turns` is reached.
4. Workspace lifecycle.
   - [`elixir/lib/symphony_elixir/workspace.ex`](/Users/lihui/Documents/GitHub/symphony/elixir/lib/symphony_elixir/workspace.ex#L1) maps issue identifiers to deterministic workspace paths, runs `after_create` / `before_run` / `after_run` / `before_remove` hooks, and removes terminal-issue workspaces.
5. Agent execution.
   - [`elixir/lib/symphony_elixir/agent_runner.ex`](/Users/lihui/Documents/GitHub/symphony/elixir/lib/symphony_elixir/agent_runner.ex#L1) is the one-issue execution loop.
   - It creates the workspace, runs hooks, starts Codex app-server, runs turns, and reports updates back to the orchestrator.
   - Prompt rendering is handled by [`elixir/lib/symphony_elixir/prompt_builder.ex`](/Users/lihui/Documents/GitHub/symphony/elixir/lib/symphony_elixir/prompt_builder.ex#L1), which injects the issue struct into the `WORKFLOW.md` prompt template.
6. Codex app-server protocol.
   - [`elixir/lib/symphony_elixir/codex/app_server.ex`](/Users/lihui/Documents/GitHub/symphony/elixir/lib/symphony_elixir/codex/app_server.ex#L1) starts `codex app-server` over stdio, sends `initialize`, `thread/start`, and `turn/start`, then waits for `turn/completed`, `turn/failed`, or approval/input events.
   - It enforces the workspace CWD guardrail, passes through sandbox policy, and emits tool-call events into a dynamic tool executor.
7. Linear tool bridge during Codex turns.
   - The only built-in dynamic tool is `linear_graphql`, defined in [`elixir/lib/symphony_elixir/codex/dynamic_tool.ex`](/Users/lihui/Documents/GitHub/symphony/elixir/lib/symphony_elixir/codex/dynamic_tool.ex#L1).
   - That is what allows repo skills to do arbitrary Linear reads/writes from within a Codex session.
8. Operator visibility.
   - [`elixir/lib/symphony_elixir/status_dashboard.ex`](/Users/lihui/Documents/GitHub/symphony/elixir/lib/symphony_elixir/status_dashboard.ex#L1) renders a terminal status surface.
   - [`elixir/lib/symphony_elixir/http_server.ex`](/Users/lihui/Documents/GitHub/symphony/elixir/lib/symphony_elixir/http_server.ex#L1) starts the optional Phoenix observability endpoint.

Useful mental model:

```text
CLI -> Workflow/Config -> Orchestrator -> Workspace -> AgentRunner -> Codex app-server
                                              |                         |
                                              v                         v
                                         Linear.Client          DynamicTool.linear_graphql
```

## 3. CLI / workflow config examples operators actually use

The repo already documents the operator-facing contract in two places:

- [`elixir/README.md`](/Users/lihui/Documents/GitHub/symphony/elixir/README.md#L14) explains the run model, a minimal `WORKFLOW.md`, and the supported workflow knobs.
  - It shows the basic startup sequence with `mise install`, `mix build`, and `./bin/symphony ./WORKFLOW.md`.
  - It documents `tracker.kind: linear`, `workspace.root`, `hooks.after_create`, `agent.max_concurrent_agents`, `agent.max_turns`, and `codex.command`.
  - It also documents env-backed values such as `tracker.api_key: $LINEAR_API_KEY`, `workspace.root: $SYMPHONY_WORKSPACE_ROOT`, and `codex.command: "$CODEX_BIN app-server --model gpt-5.3-codex"`.
  - It calls out the optional dashboard/API startup via `--port`.
  - It documents the live e2e workflow and the fact that it creates disposable Linear resources and launches a real `codex app-server` session.
- [`elixir/WORKFLOW.md`](/Users/lihui/Documents/GitHub/symphony/elixir/WORKFLOW.md#L1) is the canonical in-repo workflow contract used by real runs.
  - It sets `tracker.kind: linear`, a concrete project slug, active states, terminal states, polling interval, workspace root, lifecycle hooks, agent concurrency, and the Codex command.
  - The prompt body tells the agent how to behave on a Linear ticket and explicitly requires a `linear_graphql` tool.
  - It contains the operational rules for status transitions, workpad use, blocked paths, and the handoff states (`Todo`, `In Progress`, `Human Review`, `Merging`, `Rework`, `Done`).

## 4. What go-symphony should preserve vs. what can be Go-native

### Preserve

- The repository-owned `WORKFLOW.md` contract as the user-visible policy surface.
- Workspace-per-issue isolation and deterministic identifier-to-path mapping.
- The distinction between tracker reads, orchestration, workspace lifecycle, and Codex app-server execution.
- The `linear_graphql` capability inside Codex sessions for raw tracker operations.
- Active-state / terminal-state semantics, including stopping active work when the issue becomes ineligible.
- The “keep working until done or blocked” turn continuation behavior.
- Operator-visible observability and the ability to run a live end-to-end flow against real Linear + real Codex.

### Can be Go-native

- A dedicated `linear probe` / `verify linear config` command that isolates read-only fetching and shape validation.
- A clearer CLI split between “probe Linear data” and “run the full daemon.”
- A more explicit typed config / validation layer if that makes the Go implementation easier to reason about.
- A Go-native test harness for the two-stage verification story:
  - stage 1: read-only Linear fetch / issue shape inspection
  - stage 2: full issue-to-Codex end-to-end execution

In the original repo, the second stage already exists as a live e2e test, but the first stage is implicit inside the tracker client rather than exposed as a dedicated operator command. That is the main gap a Go-native implementation can fill cleanly without changing the underlying behavior.
