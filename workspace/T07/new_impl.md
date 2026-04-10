# T07 New Implementation Analysis

## Current Go-side State

The Go repository already has the configuration inputs that T07 needs, but it does not yet have a workspace lifecycle implementation.

- `internal/config/settings.go` already normalizes `workspace.root` into `Settings.Workspace.Root` and carries the hook command strings plus timeout under `Settings.Hooks`:
  - `after_create`
  - `before_run`
  - `after_run`
  - `before_remove`
  - `timeout_ms`
- The same file already validates the relevant limits and defaults:
  - default workspace root is `${TMPDIR}/symphony_workspaces`
  - default hooks timeout is `60_000`
  - workspace root supports the same path normalization already used by the config layer
- `internal/workspace/` is still just a placeholder package. It contains only `doc.go`, so there is no lifecycle code, no path helper, no hook runner, and no tests yet.
- `internal/orchestrator/state.go` already carries workspace-related runtime facts through the scheduler:
  - running entries retain `WorkspacePath`
  - retry entries retain `WorkspacePath`
  - `stopRunRequest` already includes `CleanupWorkspace bool`
  - `invalidateRun` sets cleanup intent to `true` only when a refreshed item is terminal, and `false` otherwise
- `workspace/T06/final_impl.md` explicitly defers workspace creation/removal behavior to `T07`, so T06 already established the cleanup intent but not the actual lifecycle behavior.

## What T07 Needs To Add

T07 should turn the placeholder package into the narrow workspace lifecycle boundary that the approved design expects.

The missing behavior is:

1. Workspace naming and path construction from the configured workspace root.
2. Safe identifier handling so workspace names do not directly depend on raw tracker values.
3. Hook execution for `after_create`, `before_run`, `after_run`, and `before_remove`.
4. Cleanup behavior for success, failure, timeout, and terminal-state invalidation.
5. Tests that pin the lifecycle contract without dragging in runner, Codex, or tracker write semantics.

The important split is that T07 owns workspace lifecycle semantics, not run execution or protocol behavior. The package should stay provider-neutral and should not invent tracker writes or Codex-specific behavior to make the lifecycle look complete.

## Likely Package Boundary

The cleanest boundary is for `internal/workspace` to own the path and hook semantics, while the orchestrator continues to own the decision about when cleanup is requested.

That implies `internal/workspace` should likely contain:

- workspace root resolution helpers
- workspace name / identifier sanitization
- create/remove lifecycle helpers
- hook invocation with the configured timeout
- cleanup helpers that can be called when orchestration requests removal

What should stay out of this package in V1:

- tracker read/write logic
- runner host selection or SSH policy
- Codex session lifecycle or protocol handling
- observability projection
- any generic provider-agnostic write API

## Missing Behavior In The Current Codebase

The current repository does not yet show how a workspace path is derived from an item identifier or how invalid path characters are handled. It also does not show how hooks are launched, whether failures are fatal, or how hook timeout is enforced.

The current cleanup intent in `internal/orchestrator/state.go` is also only a signal. The package has enough state to decide whether removal should clean up the workspace, but the actual file-system work and hook ordering still do not exist.

In other words: T06 already knows when cleanup should happen; T07 needs to define what cleanup actually does.

## Recommended Non-goals

Keep T07 narrow and do not absorb later tasks:

- Do not add SSH/local execution policy or host-capacity logic; that belongs to `T08`.
- Do not add Codex app-server session or event parsing; that belongs to `T09`.
- Do not add tracker reader/write abstractions beyond whatever the workspace package itself needs internally; `T10` is the tracker boundary task.
- Do not add provider-specific workflow behavior or Linear write semantics.
- Do not widen `internal/workspace` into a generic runtime abstraction layer.

## Risks And Open Questions

- Hook failure policy is not yet explicit. The remaining decision is whether `after_create`, `before_run`, `after_run`, and `before_remove` failures should fail the whole run, be best-effort, or differ by phase.
- The workspace naming contract is still underspecified in Go. We need to decide how much of the Elixir path shape is required versus a simpler Go-native sanitization rule that preserves observable compatibility.
- Timeout semantics for hooks need to be pinned down. The config already provides one `hooks.timeout_ms` value, but the lifecycle code still has to decide whether all hooks share it and how to report timeout failures.
- Cleanup ordering is still only implied by orchestration intent. We need to decide whether cleanup is synchronous in the caller, delegated to a helper, or wrapped in a separate command runner abstraction.

## Summary

The repository is ready for T07 at the boundary level: config already supplies workspace and hook inputs, the orchestrator already emits cleanup intent, and the workspace package is still empty. T07 should fill only the lifecycle gap, keep the package provider-neutral, and avoid freezing later execution or protocol concerns.
