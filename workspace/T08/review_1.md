# T08 Review: final_impl_v1

## Findings

1. [high] The runner admission API is underspecified for the capacity rules the plan says runner should own. `ExecutionHost.Admit(preferredHost string)` at [lines 48-53](/Users/lihui/Documents/GitHub/go-symphony/.worktrees/t08-runner-executionhost/workspace/T08/final_impl_v1.md#L48) has no input for current host occupancy, total concurrency, or per-state limits, yet the plan says runner should own host admission and per-host capacity checks at [lines 15-21](/Users/lihui/Documents/GitHub/go-symphony/.worktrees/t08-runner-executionhost/workspace/T08/final_impl_v1.md#L15) and [lines 31-35](/Users/lihui/Documents/GitHub/go-symphony/.worktrees/t08-runner-executionhost/workspace/T08/final_impl_v1.md#L31). In the current code, those facts live in orchestrator state (`serviceDeps.admitRun`, `schedulerState.running`, `maxConcurrent`, and `stateLimits`) at [internal/orchestrator/state.go:36-42](/Users/lihui/Documents/GitHub/go-symphony/.worktrees/t08-runner-executionhost/internal/orchestrator/state.go#L36) and [internal/orchestrator/state.go:102-123](/Users/lihui/Documents/GitHub/go-symphony/.worktrees/t08-runner-executionhost/internal/orchestrator/state.go#L102). As written, the plan either forces hidden global state into runner or leaves the required capacity behavior impossible to implement deterministically. That is a blocking gap.

2. [medium] The plan pushes workspace create/remove mechanics into runner more broadly than the approved design calls for. `ExecutionHost` includes `EnsureWorkspace(...)` and `RemoveWorkspace(...)` at [lines 48-53](/Users/lihui/Documents/GitHub/go-symphony/.worktrees/t08-runner-executionhost/workspace/T08/final_impl_v1.md#L48), and Task 2 says workspace should call runner for “actual command and remote workspace operations” at [lines 135-148](/Users/lihui/Documents/GitHub/go-symphony/.worktrees/t08-runner-executionhost/workspace/T08/final_impl_v1.md#L135). But the source design keeps `workspace` as the owner of workspace path rules, hooks, cleanup, and terminal-state cleanup semantics at [docs/plans/2026-04-10-go-symphony-design.md:109-115](/Users/lihui/Documents/GitHub/go-symphony/.worktrees/t08-runner-executionhost/docs/plans/2026-04-10-go-symphony-design.md#L109), and the current Go code already centralizes create/reuse/remove policy in `internal/workspace/manager.go` at [lines 121-219](/Users/lihui/Documents/GitHub/go-symphony/.worktrees/t08-runner-executionhost/internal/workspace/manager.go#L121). The plan should narrow runner to transport/process execution and keep workspace CRUD policy in `internal/workspace`, otherwise the boundary becomes less clean than the design intends.

## Score

- Symphony alignment and source fidelity: 20/30
- Go-native simplicity and maintainability: 14/20
- No overdesign / clean boundaries: 13/20
- Implementation clarity and testability: 11/15
- Verification coverage and rollout safety: 11/15

Total: 69/100

## Verdict

Block on the host-admission contract until the plan defines where current host occupancy and concurrency state come from. After that, tighten the runner/workspace split so runner owns transport and execution, while workspace keeps lifecycle policy and workspace CRUD semantics.
