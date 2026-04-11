# T08 Original Implementation Notes

## What The Elixir Code Actually Does

The original Symphony implementation does not have a standalone "runner" package in the Elixir codebase. The runtime split is:

- `AgentRunner` owns the per-issue execution lifecycle.
- `Workspace` owns workspace creation, validation, hooks, and removal.
- `SSH` is a thin transport wrapper.
- `Codex.AppServer` owns Codex process/session startup and turn execution.
- `Orchestrator` owns dispatch, host selection, retry, cleanup, and runtime state.

That split matters for T08 because the Go version should separate local/SSH execution concerns from workspace lifecycle, but the Elixir code already keeps the SSH transport and the Codex session boundary outside the workspace module.

## Execution Flow

The top-level flow for one issue is:

1. `Orchestrator` selects an issue and a worker host.
2. `AgentRunner.run/3` starts the run and keeps one worker lifetime on one host.
3. `Workspace.create_for_issue/2` creates or reuses the issue workspace.
4. `Workspace.run_before_run_hook/3` runs any configured pre-run hook.
5. `Codex.AppServer.start_session/2` opens the Codex app-server process.
6. `Codex.AppServer.run_turn/4` runs one or more Codex turns, with continuation turns when the issue is still active.
7. `Workspace.run_after_run_hook/3` always runs in an `after` block.

Source refs:

- `elixir/lib/symphony_elixir/agent_runner.ex:12-42`
- `elixir/lib/symphony_elixir/codex/app_server.ex:28-35`
- `elixir/lib/symphony_elixir/codex/app_server.ex:39-67`
- `elixir/lib/symphony_elixir/codex/app_server.ex:69-145`

## Local Execution Path

When no worker host is selected, local execution is used.

### Codex process launch

`Codex.AppServer.start_port/2` starts `bash -lc <codex.command>` directly with `Port.open/2`, using:

- `cd: workspace`
- `:stderr_to_stdout`
- `:binary`
- `:exit_status`
- line buffering set to `@port_line_bytes`

The local launch path validates the workspace root first and rejects:

- the workspace root itself
- paths outside the workspace root
- symlink escapes under the workspace root

Source refs:

- `elixir/lib/symphony_elixir/codex/app_server.ex:147-173`
- `elixir/lib/symphony_elixir/codex/app_server.ex:189-210`

### Local workspace creation and cleanup

For local runs, `Workspace.create_for_issue/2`:

- sanitizes the issue identifier into a safe directory name
- canonicalizes the workspace path under `Config.settings!().workspace.root`
- creates the directory locally
- runs `after_create` only when the workspace was newly created

Cleanup is local filesystem cleanup:

- `Workspace.remove/2` validates the path before deletion
- `before_remove` runs locally if configured
- `Workspace.remove_issue_workspaces/2` removes the issue workspace for the local root, or across all configured SSH hosts when no worker host is provided

Source refs:

- `elixir/lib/symphony_elixir/workspace.ex:13-45`
- `elixir/lib/symphony_elixir/workspace.ex:87-160`
- `elixir/lib/symphony_elixir/workspace.ex:166-345`
- `elixir/lib/symphony_elixir/workspace.ex:358-365`

## SSH Execution Path

When a worker host is selected, the Elixir code keeps SSH as a transport detail, not a workspace concern.

### SSH transport

`SSH.run/3` and `SSH.start_port/3` both:

- find the `ssh` executable
- add `-T`
- add `-F <config>` if `SYMPHONY_SSH_CONFIG` is set
- parse `host:port` into `ssh -p <port> <host>`
- preserve user prefixes such as `user@host:port`
- treat bracketed IPv6 specially
- wrap the remote command as `bash -lc '<command>'`

Source refs:

- `elixir/lib/symphony_elixir/ssh.ex:4-25`
- `elixir/lib/symphony_elixir/ssh.ex:29-99`
- `elixir/test/symphony_elixir/ssh_test.exs:6-163`

### Remote workspace creation

For SSH workers, `Workspace.create_for_issue/2` does not create a local directory. Instead it sends a remote shell script that:

- defines the workspace path safely in the remote shell
- creates or reuses the workspace directory
- removes a non-directory path at that location if needed
- `cd`s into the workspace
- prints a marker line containing whether the workspace was created and the canonical remote path

That marker is parsed back into the Elixir process, and `after_create` is still only invoked when the workspace was newly created.

Source refs:

- `elixir/lib/symphony_elixir/workspace.ex:48-79`
- `elixir/lib/symphony_elixir/workspace.ex:196-225`
- `elixir/lib/symphony_elixir/workspace.ex:358-365`

### Remote hooks and removal

For SSH workers, hooks run remotely too:

- `before_run` executes through `run_remote_command/3`
- `after_run` executes through `run_remote_command/3`
- `before_remove` executes in the remote workspace if it exists
- remote removal uses `rm -rf "$workspace"`

Timeouts are enforced with `Task.yield/2` and `Task.shutdown/2` for local hooks, and with the same timeout budget for remote commands.

Source refs:

- `elixir/lib/symphony_elixir/workspace.ex:253-333`
- `elixir/lib/symphony_elixir/workspace.ex:346-355`

## Worker Host Selection And Capacity

The orchestrator is the owner of host selection.

Rules in the original implementation:

- if `worker.ssh_hosts` is empty, the run is local
- if hosts exist, only hosts under `worker.max_concurrent_agents_per_host` are eligible
- a preferred host is honored if it is eligible
- otherwise the least-loaded eligible host is chosen
- if no host has capacity, the run is not dispatched
- total dispatch capacity is also capped by `agent.max_concurrent_agents`

The orchestrator tracks host usage from `state.running`, so host capacity is enforced before a worker is started.

Source refs:

- `elixir/lib/symphony_elixir/orchestrator.ex:973-1033`
- `elixir/lib/symphony_elixir/orchestrator.ex:1061-1067`
- `elixir/lib/symphony_elixir/orchestrator.ex:554-567`
- `elixir/lib/symphony_elixir/config/schema.ex:103-119`
- `elixir/lib/symphony_elixir/config/schema.ex:122-150`

`AgentRunner.run/3` also normalizes the selected host once more for the worker lifetime, which means one run does not hop between machines after it starts.

Source ref:

- `elixir/lib/symphony_elixir/agent_runner.ex:12-18`

## Process Lifecycle And Cleanup

The original lifecycle is broader than just "start a command and wait":

- `AgentRunner` sends `{:worker_runtime_info, ...}` to the orchestrator with the worker host and workspace path.
- `AgentRunner` starts a Codex session, runs turns, and always stops the session in an `after` block.
- `Codex.AppServer.run/4` also wraps the whole session in `start_session/2` and `stop_session/1`.
- `Codex.AppServer` emits structured events for session start, turn completion/failure, tool approval, and input-required cases.
- `Orchestrator` records those events in its running-state snapshot.

The orchestrator also has several cleanup paths:

- startup cleanup removes terminal-state workspaces for all terminal Linear issues
- when an issue moves to a terminal state, its workspace is removed
- when a run exits normally, the orchestrator schedules a continuation retry check
- when a run exits abnormally, it schedules a backoff retry
- retry bookkeeping preserves worker host and workspace path across attempts

Source refs:

- `elixir/lib/symphony_elixir/agent_runner.ex:32-87`
- `elixir/lib/symphony_elixir/codex/app_server.ex:28-145`
- `elixir/lib/symphony_elixir/codex/app_server.ex:329-497`
- `elixir/lib/symphony_elixir/codex/app_server.ex:526-880`
- `elixir/lib/symphony_elixir/codex/app_server.ex:993-1012`
- `elixir/lib/symphony_elixir/orchestrator.ex:119-180`
- `elixir/lib/symphony_elixir/orchestrator.ex:773-920`
- `elixir/lib/symphony_elixir/orchestrator.ex:882-897`

## Runtime Config That Shapes Runner Behavior

The relevant config surface is:

- `workspace.root`
- `worker.ssh_hosts`
- `worker.max_concurrent_agents_per_host`
- `agent.max_concurrent_agents`
- `agent.max_turns`
- `agent.max_retry_backoff_ms`
- `codex.command`
- `codex.turn_timeout_ms`
- `codex.read_timeout_ms`
- `codex.stall_timeout_ms`
- `hooks.before_run`, `hooks.after_run`, `hooks.before_remove`, `hooks.after_create`

The default turn sandbox policy is derived from the workspace root, and remote sessions use `remote: true` so the sandbox policy keeps the remote workspace path as-is.

Source refs:

- `elixir/lib/symphony_elixir/config/schema.ex:86-119`
- `elixir/lib/symphony_elixir/config/schema.ex:122-199`
- `elixir/lib/symphony_elixir/config/schema.ex:292-317`
- `elixir/lib/symphony_elixir/config/schema.ex:368-506`
- `elixir/lib/symphony_elixir/config.ex:39-99`

## T08 Compatibility Takeaway

For the Go runner/execution-host boundary, the Elixir behavior to preserve is:

1. workspace lifecycle is separate from command transport
2. SSH is only a transport/launch detail
3. local and remote runs share the same higher-level Codex session contract
4. host selection and capacity are orchestrator-owned
5. cleanup happens in both startup and terminal-state paths
6. runtime events must expose worker host and workspace path back to the orchestrator

That is the shape T08 should match.
