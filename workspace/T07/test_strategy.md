# T07 Test Strategy

## Goal

`T07 Workspace Lifecycle` must prove that `internal/workspace` becomes the lifecycle boundary for workspace naming, path safety, hook execution, and cleanup semantics without pulling in runner, Codex, or tracker ownership too early. The proof is not “the package compiles”; it is that the package enforces the Elixir-observable lifecycle rules closely enough that later runtime tasks can call into it with confidence.

## What Must Be Proven

1. Workspace paths are derived deterministically from the configured root and a normalized identifier, with `"issue"` as the empty / nil fallback.
2. Local workspace paths are validated safely before use: canonical root handling, root equality rejection, outside-root rejection, and symlink-escape rejection all behave as intended.
3. `Create(...)` preserves existing directories, replaces stale non-directory paths, and only runs `after_create` when the workspace is newly created.
4. `RunWithHooks(...)` enforces the run-phase ordering: `before_run` is fatal, the run body only executes after `before_run` succeeds, and `after_run` still executes on every exit path while remaining best-effort.
5. `Remove(...)` and `RemoveIssueWorkspaces(...)` enforce cleanup semantics: `before_remove` is best-effort, terminal cleanup is explicit, and host-aware fan-out is preserved when no explicit host is supplied.
6. The package remains isolated from later runtime ownership: no SSH transport implementation, runner policy, Codex protocol logic, or tracker mutation behavior is frozen by this task.
7. The repository still builds and lints cleanly after the new package lands.

## Verification Matrix

### 1. Package-scoped workspace contract tests

Command:

```bash
go test ./internal/workspace/...
```

Why this matters:

- This is the task-board gate for `T07`.
- `T07` is a behavior-freezing task, so package tests must prove the lifecycle contract directly rather than only compiling the package.
- These tests should cover the full path-and-hook matrix, including:
  - identifier normalization and empty-identifier fallback
  - deterministic path derivation from `root + identifier`
  - canonical root handling, symlinked-root handling, root equality rejection, outside-root rejection, and symlink-escape rejection
  - create reuse versus stale-path replacement
  - `after_create` failure and timeout are fatal
  - `before_run` failure and timeout are fatal
  - `after_run` executes even when `before_run` fails or the run body returns an error, and its own failure / timeout are ignored by the caller
  - `before_remove` failure and timeout are ignored by the caller
  - explicit cleanup removes the intended workspace path
  - non-terminal invalidation does not trigger implicit cleanup
  - host-addressed cleanup fan-out follows the configured worker-host list when no explicit host is supplied

### 2. Repository-wide compile safety

Command:

```bash
go test ./...
```

Why this matters:

- `T07` introduces the first real `internal/workspace` implementation and its exported boundary will be consumed by later runtime tasks.
- A green package suite is not enough if the new package collides with existing placeholders or breaks compilation elsewhere in the module.

### 3. Canonical build and lint gates

Commands:

```bash
make build
make lint
```

Why this matters:

- `make build` proves the repository still builds through the canonical entrypoint after `internal/workspace` stops being a placeholder.
- `make lint` catches API-shape and structural problems that focused behavior tests may miss.

### 4. E2E applicability check

Command:

```bash
make test-e2e
```

Why this matters:

- `T07` does not yet wire the full runtime path, so e2e is not the primary proof of lifecycle correctness.
- Running the standard e2e command still verifies the repository command contract remains healthy.
- If this command is only a command-contract check at this phase, that limitation must be recorded explicitly in `workspace/T07/todo.md` and the task board instead of being silently skipped.

## Deferred Checks

The following checks are intentionally not part of T07 acceptance because the full runtime is not wired yet:

- full runner / execution-host integration
- Codex app-server lifecycle and event handling
- tracker read/write behavior beyond the workspace lifecycle seam
- end-to-end runtime execution through `T14`

Those checks belong to later tasks, and T07 should avoid pretending to prove them indirectly.

## Acceptance Threshold

`T07` is ready to leave verification only if:

- `go test ./internal/workspace/...` proves the exact path, hook, and cleanup rules in `workspace/T07/final_impl.md`
- `go test ./...`, `make build`, and `make lint` pass
- `make test-e2e` either passes or its current applicability limit is recorded explicitly
- the observed behavior still keeps workspace lifecycle concerns inside `internal/workspace` and does not freeze later runner, Codex, or tracker ownership early
