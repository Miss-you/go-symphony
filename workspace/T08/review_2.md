# T08 Review 2

## Findings

1. **High**: `ExecutionHost.Admit(preferredHost)` is too underspecified to preserve the current capacity-aware host selection contract. The plan says `runner` should own host admission and per-host capacity checks, but the proposed API takes only a preferred host and does not accept the live occupancy data that current orchestration uses to make the decision. In the current code, capacity is derived from orchestrator-owned runtime state (`serviceDeps.admitRun` over `state.running`), so this contract either forces runner to reach back into orchestrator state or silently degrades selection to a stateless preference chooser. Either outcome breaks the stated ownership boundary and risks shipping a runner boundary that cannot enforce `maxConcurrentAgents` / per-host limits correctly. This should block acceptance until the admission data flow is made explicit.

   References: [/Users/lihui/Documents/GitHub/go-symphony/.worktrees/t08-runner-executionhost/workspace/T08/final_impl_v1.md:48-53](file:///Users/lihui/Documents/GitHub/go-symphony/.worktrees/t08-runner-executionhost/workspace/T08/final_impl_v1.md#L48), [/Users/lihui/Documents/GitHub/go-symphony/.worktrees/t08-runner-executionhost/workspace/T08/final_impl_v1.md:165-177](file:///Users/lihui/Documents/GitHub/go-symphony/.worktrees/t08-runner-executionhost/workspace/T08/final_impl_v1.md#L165), [/Users/lihui/Documents/GitHub/go-symphony/.worktrees/t08-runner-executionhost/internal/orchestrator/state.go:36-41](file:///Users/lihui/Documents/GitHub/go-symphony/.worktrees/t08-runner-executionhost/internal/orchestrator/state.go#L36), [/Users/lihui/Documents/GitHub/go-symphony/.worktrees/t08-runner-executionhost/docs/plans/2026-04-10-go-symphony-design.md:112-114](file:///Users/lihui/Documents/GitHub/go-symphony/.worktrees/t08-runner-executionhost/docs/plans/2026-04-10-go-symphony-design.md#L112)

2. **Medium**: The proposed runner boundary is wider than the task needs because it pulls workspace lifecycle mutation into `runner` via `EnsureWorkspace(...)` and `RemoveWorkspace(...)`. The design keeps `workspace` as the owner of path rules, hooks, cleanup, and terminal-state cleanup semantics, and the current `workspace` package already contains those responsibilities. Moving filesystem lifecycle operations into `runner` risks creating a second lifecycle layer rather than a transport boundary, which makes the eventual wiring harder to reason about and increases the chance of duplicate policy between `workspace` and `runner`.

   References: [/Users/lihui/Documents/GitHub/go-symphony/.worktrees/t08-runner-executionhost/workspace/T08/final_impl_v1.md:51-52](file:///Users/lihui/Documents/GitHub/go-symphony/.worktrees/t08-runner-executionhost/workspace/T08/final_impl_v1.md#L51), [/Users/lihui/Documents/GitHub/go-symphony/.worktrees/t08-runner-executionhost/workspace/T08/final_impl_v1.md:84-85](file:///Users/lihui/Documents/GitHub/go-symphony/.worktrees/t08-runner-executionhost/workspace/T08/final_impl_v1.md#L84), [/Users/lihui/Documents/GitHub/go-symphony/.worktrees/t08-runner-executionhost/workspace/T08/final_impl_v1.md:135-148](file:///Users/lihui/Documents/GitHub/go-symphony/.worktrees/t08-runner-executionhost/workspace/T08/final_impl_v1.md#L135), [/Users/lihui/Documents/GitHub/go-symphony/.worktrees/t08-runner-executionhost/docs/plans/2026-04-10-go-symphony-design.md:109-114](file:///Users/lihui/Documents/GitHub/go-symphony/.worktrees/t08-runner-executionhost/docs/plans/2026-04-10-go-symphony-design.md#L109), [/Users/lihui/Documents/GitHub/go-symphony/.worktrees/t08-runner-executionhost/internal/workspace/manager.go:130-147](file:///Users/lihui/Documents/GitHub/go-symphony/.worktrees/t08-runner-executionhost/internal/workspace/manager.go#L130), [/Users/lihui/Documents/GitHub/go-symphony/.worktrees/t08-runner-executionhost/internal/workspace/manager.go:171-210](file:///Users/lihui/Documents/GitHub/go-symphony/.worktrees/t08-runner-executionhost/internal/workspace/manager.go#L171)

## Score

- Symphony alignment and source fidelity: 21/30
- Go-native simplicity and maintainability: 13/20
- No overdesign / clean boundaries: 9/20
- Implementation clarity and testability: 13/15
- Verification coverage and rollout safety: 12/15

**Total: 68/100**

## Verdict

Do not accept `final_impl_v1.md` as-is. The admission contract needs to be made explicit, and the runner/workspace boundary should stay narrower than the current draft.
