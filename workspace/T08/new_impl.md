# T08 New Implementation Analysis

## Current Go-side State

The approved design already says `runner` owns local and SSH execution behavior, while `workspace` keeps path safety, lifecycle hooks, and terminal cleanup semantics. The current Go repo has not made that split yet.

What exists today:

- `internal/runner/` is only a placeholder package with `doc.go`.
- `internal/workspace/manager.go` still owns a private `transport` interface with three responsibilities mixed together:
  - `EnsureWorkspace(...)`
  - `RunCommand(...)`
  - `RemoveWorkspace(...)`
- `internal/workspace/manager.go` also owns the local implementation of that transport through `localTransport`, including shell execution via `sh -lc`.
- `internal/workspace/manager.go` already normalizes `settings.Worker.SSHHosts` into `workerHosts` and fans out `RemoveIssueWorkspaces(...)` across them when no explicit host is provided.
- `internal/orchestrator/state.go` already carries host-aware runtime facts:
  - `startRunRequest.PreferredHost`
  - `startRunResult.WorkerHost`, `WorkspacePath`, `SessionID`, and `Handle`
  - `runningEntry.WorkerHost` and `WorkspacePath`
  - `retryEntry.WorkerHost` and `WorkspacePath`
  - `stopRunRequest.CleanupWorkspace`
  - `serviceDeps.admitRun(...)` as the current private host-admission seam
- `internal/domain/types.go` already has the runtime vocabulary T08 needs, including `RunEventRunnerHostSelected`, `ActiveRun.WorkerHost`, `ActiveRun.WorkspacePath`, and the retry/snapshot types.
- `internal/config/settings.go` already has the host inputs that runner should eventually consume:
  - `Worker.SSHHosts`
  - `Worker.MaxConcurrentAgentsPerHost`

The current layout proves the repo already knows about hosts and cleanup intent, but it does not yet have a real execution-host boundary.

## Current Boundary Map

The repository is already split enough to show what T08 should and should not own.

Keep in `internal/workspace`:

- deterministic workspace naming
- safe identifier normalization
- local path validation
- create/reuse/remove lifecycle behavior
- `after_create`, `before_run`, `after_run`, and `before_remove` policy
- hostless cleanup fan-out for terminal sweeps

Move out of `internal/workspace`:

- shell invocation
- SSH transport details
- host admission and capacity policy
- remote command launch
- execution teardown for local versus SSH hosts

Keep in `internal/orchestrator`:

- the single owner of mutable scheduling state
- retry lineage
- stall recovery
- cleanup intent through `stopRunRequest.CleanupWorkspace`
- the private host-admission seam until a concrete runner adapter exists

The important point is that T07 already proved workspace lifecycle semantics. T08 should now remove execution mechanics from that package without reintroducing tracker or Codex concerns.

## What T08 Should Add

T08 should make `internal/runner` the real runtime contract for execution hosts.

That contract should cover:

1. Host selection or admission.
2. Local versus SSH execution behavior behind one interface.
3. Command launch and teardown.
4. Any host-level capacity gating that currently leaks into orchestration or workspace code.

The exact names can stay Go-native, but the contract should distinguish:

- choosing or admitting a host
- launching a command on that host
- reporting the selected host back to the runtime
- cleaning up execution resources without teaching `workspace` how SSH works

That means `internal/workspace` should stop being the place that knows how to run a hook command. It should keep hook ordering and policy, but delegate the actual execution to the runner boundary.

## Suggested Package Shape

The most plausible shape is a small execution-host package with a narrow set of types:

- a runner contract for host-aware command execution
- a local implementation for the non-SSH path
- an SSH implementation for remote execution
- a result type that carries host, handle, session, and workspace path information back to the orchestrator

The current code already gives a hint of the seams:

- `serviceDeps.admitRun(...)` in `internal/orchestrator/state.go`
- `startRunRequest.PreferredHost`
- `startRunResult.WorkerHost`
- `domain.RunEventRunnerHostSelected`

So T08 does not need to invent a new runtime vocabulary. It needs to harden the execution boundary that those fields already imply.

## Integration Points

`internal/orchestrator` is the main consumer of the future runner boundary.

- `admitRun(...)` is the current place where host admission can be wired to a concrete runner implementation.
- `startRun(...)` is where the selected host, workspace path, and session/handle data come back into orchestration state.
- `stopRun(...)` is where cleanup intent can turn into actual teardown.
- `RunEventRunnerHostSelected` is already the right event to report the selected host without exposing runner internals.

`internal/workspace` also needs to change, but more narrowly.

- `Create(...)` should remain the lifecycle entry point for workspace creation.
- `RunWithHooks(...)` should keep the hook ordering policy, but it should not own shell or SSH execution.
- `RemoveIssueWorkspaces(...)` should keep terminal cleanup fan-out, but the low-level remove implementation should not be tied to a private local shell transport.

`internal/config` is already ready enough for this task.

- `Workspace.Root` stays with workspace path logic.
- `Worker.SSHHosts` and `Worker.MaxConcurrentAgentsPerHost` are the inputs runner should use for host selection and capacity.
- Nothing in config needs a new provider-specific abstraction to support T08.

## Test Targets For `internal/runner`

The task board gate is `go test ./internal/runner/...`, so the runner package should be locked with package-scoped tests that prove the boundary instead of only compiling it.

High-value test areas:

- local execution path launches commands in the expected working directory
- SSH execution path stays separate from local shell behavior
- host admission honors configured host lists and per-host capacity
- preferred-host selection falls back predictably when needed
- cleanup/teardown preserves execution semantics without leaking into `workspace`
- result objects carry enough host and runtime metadata back to orchestration

It is also worth adding tests that prove `internal/workspace` no longer needs to know how to shell out directly once the runner boundary exists.

## Risks And Open Questions

- The biggest design choice is whether hook execution stays in `workspace` as policy only, or whether `runner` also owns the actual command invocation for hooks. The current code suggests policy belongs in workspace, but execution should move out.
- It is still unclear whether remote workspace creation/removal should live in runner itself or in a narrower transport helper under runner.
- `serviceDeps.admitRun(...)` may remain an orchestrator seam, or it may become a thin adapter over runner host admission. The repo does not need to decide that too early, but T08 should make the runner side concrete enough that the choice is no longer hypothetical.
- The current workspace package combines lifecycle and transport in one file. T08 should reduce that coupling rather than replace it with another broad abstraction.

## Bottom Line

The repo is ready for T08 because the surrounding contracts already exist: orchestrator owns runtime state, workspace owns lifecycle policy, and config already carries the SSH host inputs. What is missing is a real `internal/runner` boundary that takes over local/SSH execution mechanics so `internal/workspace` can stay a lifecycle package instead of a transport layer.
