# T07 Original Implementation Analysis

## What The Elixir Side Actually Does

The workspace lifecycle in the Elixir implementation is centered on `lib/symphony_elixir/workspace.ex`, with orchestration-triggered cleanup in `lib/symphony_elixir/orchestrator.ex` and lifecycle hook execution from `lib/symphony_elixir/agent_runner.ex`.

The key behavior is:

- workspaces are derived from a configured root plus a sanitized issue identifier
- local workspace paths are canonicalized and checked against the configured root
- remote workspaces are prepared and removed through SSH
- `after_create`, `before_run`, `after_run`, and `before_remove` have different failure policies
- terminal-state cleanup is driven by the orchestrator, not by the workspace module itself

## Workspace Root And Name Rules

The configured root comes from `Config.settings!().workspace.root`, which defaults to `${TMPDIR}/symphony_workspaces` in `lib/symphony_elixir/config/schema.ex`.

In `Workspace.create_for_issue/2`, the identifier is normalized by `safe_identifier/1`, which replaces every character outside `[a-zA-Z0-9._-]` with `_`. `nil` becomes `"issue"`.

Path construction then splits by worker host:

- local path: `workspace_path_for_issue/2` joins the configured root with the safe identifier and canonicalizes it with `PathSafety.canonicalize/1`
- remote path: `workspace_path_for_issue/2` returns `Path.join(root, safe_id)` without canonicalization, because the remote host creates and validates it through shell commands

Local workspace validation is stricter than remote validation:

- local `validate_workspace_path/2` rejects the root itself, paths outside the root, and symlink escapes under the root
- remote `validate_workspace_path/2` only rejects empty strings and paths containing newline, carriage return, or NUL

Relevant files:

- `lib/symphony_elixir/workspace.ex`
- `lib/symphony_elixir/config/schema.ex`
- `lib/symphony_elixir/config.ex`

## Create Flow

`Workspace.create_for_issue/2` accepts a map, string, or `nil`, derives an `issue_context`, and then:

1. computes the workspace path
2. validates the path
3. creates or reuses the directory
4. runs `after_create` only when the directory was newly created

The local create path has these semantics:

- if the target is already a directory, it is reused and local changes are preserved
- if the target exists but is not a directory, it is removed first and recreated
- if the target does not exist, it is created

The remote create path shells into the worker host, creates or reuses the directory there, and prints a marker line in the form `__SYMPHONY_WORKSPACE__\t<created>\t<pwd -P>`. The caller parses that marker in `parse_remote_workspace_output/1`.

The create path rescues `ArgumentError`, `ErlangError`, and `File.Error`, logs the failure, and returns `{:error, error}`.

Relevant tests:

- `workspace path is deterministic per issue identifier`
- `workspace reuses existing issue directory without deleting local changes`
- `workspace replaces stale non-directory paths`
- `workspace rejects symlink escapes under the configured root`
- `workspace canonicalizes symlinked workspace roots before creating issue directories`
- `workspace surfaces after_create hook failures`
- `workspace surfaces after_create hook timeouts`
- `workspace creates an empty directory when no bootstrap hook is configured`
- `remote workspace lifecycle uses ssh host aliases from worker config`

## Remove Flow

`Workspace.remove/2` and `Workspace.remove_issue_workspaces/2` implement cleanup.

Local removal:

- if the path exists, it validates the path first
- if validation succeeds, it runs `before_remove` and then `File.rm_rf/1`
- if the path does not exist, it still calls `File.rm_rf/1` and does not run hooks

Remote removal:

- `before_remove` is attempted first
- the workspace is then removed via SSH with `rm -rf "$workspace"`
- failures from the hook are ignored by design, but remote removal command failures are returned

`remove_issue_workspaces/2` also fans out across all configured SSH hosts when called without a host argument and `worker.ssh_hosts` is non-empty. That is how terminal cleanup can remove remote copies even when the caller does not know which host originally ran the issue.

Relevant tests:

- `workspace remove rejects the workspace root itself with a distinct error`
- `workspace cleanup handles missing workspace root`
- `workspace cleanup ignores non-binary identifier`
- `workspace removes all workspaces for a closed issue identifier`
- `workspace remove returns error information for missing directory`

## Hook Semantics

The hook boundaries are implemented in `Workspace.run_before_run_hook/3`, `Workspace.run_after_run_hook/3`, `maybe_run_after_create_hook/4`, and `maybe_run_before_remove_hook/2`.

Observed behavior:

- `after_create` is fatal: if the hook fails or times out, workspace creation returns an error
- `before_run` is fatal: if the hook fails or times out, the agent run should not proceed
- `after_run` is best-effort: failures and timeouts are swallowed and the call still returns `:ok`
- `before_remove` is best-effort: failures and timeouts are swallowed so cleanup can continue

Hook execution uses the configured `hooks.timeout_ms` value.

Local hooks run through `System.cmd("sh", ["-lc", command], cd: workspace, stderr_to_stdout: true)` inside a `Task`, so the timeout is enforced with `Task.yield/2` and `Task.shutdown/2`.

Remote hooks run through SSH with `cd <workspace> && <command>`.

Relevant tests:

- `workspace hooks support multiline YAML scripts and run at lifecycle boundaries`
- `workspace remove continues when before_remove hook fails`
- `workspace remove continues when before_remove hook fails with large output`
- `workspace remove continues when before_remove hook times out`

## Cleanup Across Success, Failure, Timeout, And Terminal Refresh

The workspace module itself does not decide when cleanup should happen. That decision is owned by the orchestrator.

In `agent_runner.ex`, the call chain is:

1. `Workspace.create_for_issue/2`
2. `Workspace.run_before_run_hook/3`
3. Codex turns
4. `Workspace.run_after_run_hook/3` in an `after` block

So `after_run` happens even when the Codex work fails, but its own failure is not allowed to fail the run.

In `orchestrator.ex`, cleanup intent is driven by refreshed tracker state:

- terminal state refresh => `cleanup_workspace = true`
- non-terminal invalidation => `cleanup_workspace = false`
- missing or unrouted running issues => `cleanup_workspace = false`

`terminate_running_issue/3` uses that flag to decide whether to call `cleanup_issue_workspace/2`, which delegates to `Workspace.remove_issue_workspaces/2`.

Other relevant cleanup paths:

- startup calls `run_terminal_workspace_cleanup/0`, which fetches all terminal issues and removes their workspaces
- a normal worker process exit schedules a continuation retry without cleanup
- an abnormal worker process exit schedules a retry without cleanup
- stalled runs are terminated and rescheduled without cleanup

So the Elixir parity point is: cleanup is explicitly terminal-state driven, not a generic response to every failure or timeout.

Relevant files:

- `lib/symphony_elixir/agent_runner.ex`
- `lib/symphony_elixir/orchestrator.ex`
- `lib/symphony_elixir/workspace.ex`

## Risks And Open Questions For Go T07

- The Elixir code uses different strictness for local vs remote path validation. Go T07 needs to preserve that asymmetry or consciously narrow it with a documented compatibility tradeoff.
- Hook failure policy is intentionally asymmetric. `after_create` and `before_run` are blocking, while `after_run` and `before_remove` are best-effort.
- Remote workspace creation/removal depends on SSH output parsing and a special marker line, so the Go implementation needs a comparable way to know whether the directory was newly created.
- Startup cleanup removes all terminal issues across configured SSH hosts, not just the current host. That behavior matters for remote parity.

## Bottom Line

The Elixir implementation makes `workspace.ex` the lifecycle boundary, but it does not own runtime policy. It owns path derivation, hook execution, creation/removal, and cleanup primitives, while `agent_runner.ex` and `orchestrator.ex` decide when those primitives are used.
