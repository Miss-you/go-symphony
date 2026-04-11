# T13 Spec Readiness Review

## High-Severity Spec Issues

None found.

The task board, `final_impl.md`, OpenSpec artifacts, and `test_strategy.md` are aligned on the same narrow goal: add a provider-bound workflow selector and the first `compat_linear_default` bundle without turning `internal/workflow` into a generic runtime layer.

## Scope Mismatches

None found.

The scope boundaries are consistent across the materials:

- `T13` on the task board is still the workflow-bundle task with the isolated worktree and `go test ./internal/workflow/...` gate.
- `final_impl.md` keeps the package leaf-like, Linear-only, and explicitly non-generic.
- The OpenSpec change keeps the same constraints, including explicit unsupported-provider behavior and no new workflow registry.
- `tasks.md` and the spec both require a dependency guard or equivalent check that keeps `internal/workflow` out of orchestrator/tracker/workspace/runner/domain.

## Test Strategy Assessment

`test_strategy.md` does prove the key functionality instead of just listing commands.

It ties each gate to a concrete claim:

- `go test ./internal/workflow/...` proves bundle selection, prompt fallback, and Linear tool wiring.
- the Codex compile-shape check proves the bundle can flow directly into `codex.SessionOptions` without an adapter layer.
- `go test ./...`, `make build`, and `make lint` are framed as repo-level safety gates, not as substitutes for the package-level evidence.
- `openspec validate --type change linear-workflow-bundle` proves the artifact set is complete and internally consistent.

That is the right shape for readiness: the strategy explains what each check proves and why it matters for this task.

## Verdict

Pass.
