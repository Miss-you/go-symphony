# T07 Final Implementation

## Review Gate

`final_impl_v1.md` passed the required second-round rubric review after one correction round.

Round-two review results:

- `review_3.md`: 89 / 100, no high-severity issues
- `review_4.md`: 92 / 100, no high-severity issues
- average: 90.5 / 100

Key review corrections accepted into this final plan:

- require `after_run` to execute on every post-create exit path, including `before_run` failure and run-body failure
- make canonical root handling and symlink-escape checks explicit inside `internal/workspace`, not just in config preprocessing
- preserve host-aware terminal cleanup fan-out as part of the T07 lifecycle contract, while still deferring concrete SSH transport to T08
- tighten the API shape so `RunWithHooks(...)` is the concrete run-phase entry point and `RemoveIssueWorkspaces(...)` is the shared cleanup entry point for both startup sweeps and runtime terminal cleanup

Acceptance decision:

- average score exceeds the `>= 80` threshold
- no round-two reviewer reported a remaining high-severity issue
- medium/low review notes are incorporated below

## Task Goal

`T07` implements the workspace lifecycle boundary for `go-symphony`.

The package should own:

- deterministic workspace naming
- safe identifier normalization
- workspace path validation
- lifecycle hook execution
- create / reuse / remove semantics
- terminal-state cleanup semantics when the orchestrator asks for cleanup

The package should not own:

- runner host selection
- SSH transport policy
- Codex protocol handling
- tracker reads or writes
- observability projection

## Scope

`T07` should land a single lifecycle package under `internal/workspace` with a small API surface and private helpers.

Recommended shape:

- one narrow lifecycle service that takes the already-normalized config values from `internal/config`
- one path helper that derives a workspace path from the configured root and a normalized identifier
- one concrete run-phase helper, `RunWithHooks(...)`, that locks the `before_run` / run body / `after_run` ordering
- one shared cleanup entry point, `RemoveIssueWorkspaces(...)`, used by both startup terminal sweeps and runtime terminal cleanup
- internal seams for transport, timing, and directory mutation so tests can control timeout and host-addressed behavior without baking in later runner abstractions

The public surface should be minimal. The package should expose only what later orchestrator / runner integration needs, not a generic filesystem framework.

## Non-Goals

Do not add in T07:

- SSH host selection or remote command policy
- runner execution orchestration
- Codex session lifecycle or protocol parsing
- tracker mutations or `linear_graphql`
- a provider-agnostic workflow system
- broad filesystem abstraction layers

Do not try to solve T08, T09, or T10 inside workspace.

## Package And API Shape

Use one lifecycle boundary with private implementation details.

Recommended API responsibilities:

- `PathForIdentifier(root, identifier, workerHost)` returns the workspace path derived from the configured root and the normalized identifier, using strict local validation rules and narrower host-addressed validation rules when a later caller supplies `workerHost`.
- `Create(...)` ensures the workspace exists, preserves an existing directory, replaces stale non-directory paths, and runs `after_create` only when the workspace is newly created.
- `RunWithHooks(...)` owns the run-phase ordering: once workspace creation has succeeded, it runs `before_run`, executes the run body only if `before_run` succeeds, and still invokes `after_run` in a `defer`/`finally`-equivalent on every exit path.
- `Remove(...)` validates and removes a workspace path, running `before_remove` first but not letting hook failure block cleanup.
- `RemoveIssueWorkspaces(...)` removes the workspace(s) for a closed identifier when the orchestrator requests terminal-state cleanup, and when no explicit host is supplied it must preserve the Elixir fan-out behavior across configured worker hosts.

Implementation should use a single private transport seam for filesystem / command operations, plus single private path-validation and hook-policy helpers. That keeps tests precise without exporting a premature runner abstraction, while still letting T07 freeze host-aware lifecycle semantics before T08 plugs in real SSH transport.

## Path Safety And Identifier Rules

Match the Elixir-safe naming behavior where it matters:

- normalize identifiers by replacing every character outside `[a-zA-Z0-9._-]` with `_`
- use `"issue"` when the identifier is empty or nil-equivalent
- derive paths only from the configured workspace root and the normalized identifier
- canonicalize the local path before accepting it
- reject the workspace root itself
- reject paths outside the root
- reject symlink escapes

Keep the config layer responsible for env-var and `~` expansion. `internal/workspace` should assume it receives the resolved root and should only validate the derived path.

Even with config-level expansion, `internal/workspace` must still canonicalize both the configured root and the derived local workspace path with an Elixir-equivalent safety helper before accepting them. The package, not the config layer, owns the safety decision for:

- canonical root versus canonical workspace comparison
- root-equality rejection
- outside-root rejection
- symlink-escape rejection when the expanded path still points outside the canonical root

The path rules should be stricter for local filesystem paths than for any later transport-aware path handling. If a remote execution path is added later, do not weaken the local validation rule to make the remote case feel cleaner.

## Hook Semantics

Preserve the Elixir hook asymmetry.

- `after_create` is blocking and fatal for workspace creation
- `before_run` is blocking and fatal for run start
- `after_run` is best-effort and must not fail the run
- `before_remove` is best-effort and must not block cleanup

Use the configured `hooks.timeout_ms` for all lifecycle hooks.

The hook runner should:

- execute commands in the workspace directory
- capture command output for error reporting
- time out deterministically
- surface a structured error when a hook fails or times out

The error policy should be phase-aware:

- create failures from `after_create` fail the create call
- `before_run` failures fail the run start path
- `after_run` must still execute after every post-create run attempt, including `before_run` failure, run-body failure, timeout, or early return; its own failures are logged and ignored by the caller
- `before_remove` failures are logged and ignored by the caller

## Local Vs Remote Boundary

T07 should keep the lifecycle contract transport-neutral, but it should not absorb the runner implementation.

Recommended decision:

- freeze workspace lifecycle semantics for both local paths and host-addressed paths, but keep the transport machinery behind a package-private seam
- do not add host selection, capacity, or remote topology policy here
- require T07 tests to cover host-addressed create/remove/hook behavior with fakes, while leaving the real SSH-backed transport implementation to T08

This keeps T07 narrow while still preserving the Elixir-observable host-aware cleanup and hook semantics that users can see today. T08 then owns the concrete execution-host implementation, not the lifecycle contract itself.

## Cleanup And Terminal-State Semantics

Terminal cleanup is orchestrator-driven, not workspace-driven.

The workspace package should honor a cleanup request by removing the workspace path and running `before_remove` as a best-effort hook first.

Behavior to preserve:

- success path: workspace remains available after `after_run`, unless the orchestrator explicitly requests cleanup
- failure path: workspace removal is not automatic unless the orchestrator requests cleanup
- timeout path: timeout alone does not imply cleanup unless the orchestrator marks the item terminal or explicitly asks for it
- terminal refresh invalidation: cleanup request means the workspace should be removed
- startup terminal cleanup must preserve the Elixir fan-out rule: if configured worker hosts are present and no explicit host is supplied, cleanup iterates all configured hosts for the identifier instead of silently doing local-only removal

The orchestrator already has the cleanup intent in `CleanupWorkspace`. T07 should make that intent observable in the workspace API without inventing a second policy engine, and `RemoveIssueWorkspaces(...)` should be the shared cleanup entry point that later orchestrator startup/runtime paths call.

## Error Model

Use structured errors that keep the phase, path, and cause visible.

Recommended error categories:

- invalid / unreadable workspace path
- workspace path outside root
- workspace root collision
- symlink escape
- hook timeout
- hook command failure
- workspace prepare failure
- workspace remove failure

Prefer wrapping the underlying cause instead of flattening everything into strings. The caller needs enough structure to decide whether to retry, log, or ignore.

## Test Plan

`go test ./internal/workspace/...` should prove the lifecycle contract end to end.

Tests should cover:

- identifier normalization and empty-identifier fallback
- deterministic path derivation from root + identifier
- root canonicalization, symlinked-root handling, root collision, outside-root rejection, and symlink-escape rejection
- create reuses an existing directory without deleting content
- create replaces a stale file path with a directory
- `after_create` failure and timeout are fatal
- `before_run` failure and timeout are fatal
- `after_run` runs even when `before_run` fails or the run body returns an error, and its own failure / timeout are ignored by the caller
- `before_remove` failure and timeout are ignored by the caller
- cleanup removes the intended workspace path when requested
- cleanup does not happen implicitly for non-terminal invalidation
- host-addressed cleanup fan-out follows the configured worker-host list when no explicit host is supplied

Add table-driven tests where possible so the path and hook matrix stays readable.

The tests should use fakes for command execution, transport, and timing. Avoid depending on real SSH or real tracker data in `T07`.

## Integration Points Deferred To Later Tasks

Keep these boundaries out of T07:

- `T08` will own runner and execution-host policy
- `T09` will own Codex app-server protocol behavior
- `T10` will own the tracker read boundary
- `T14` will wire workspace lifecycle into the full run path

`T07` should expose only enough surface for those later tasks to call into the lifecycle package without changing the lifecycle contract. The host-aware cleanup and hook semantics are part of that contract now; the real SSH transport is what remains deferred.

## Acceptance Criteria

`T07` is complete when all of the following are true:

- the workspace package exists and owns path, hook, and cleanup semantics
- local workspace paths are deterministic and safe
- hook behavior matches the phase-specific fatal / best-effort split from Elixir
- terminal cleanup is honored when orchestrator cleanup intent says to remove the workspace
- tests document and lock the compatibility rules above
- `go test ./internal/workspace/...` passes

## Summary Decision

The right T07 shape is a small lifecycle package with strict path safety, asymmetric hook semantics, and cleanup behavior driven by orchestrator intent. That gives the Go port parity where users can observe it, while keeping transport and execution-host policy where they belong in later tasks.
