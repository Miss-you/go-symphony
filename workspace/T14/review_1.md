# T14 Review: `final_impl_v1`

## Score Table

| Criterion | Score | Notes |
| --- | ---:| --- |
| Symphony alignment and source faithfulness | 22/30 | Good parity intent, but the memory path is still underspecified against the current Linear-only bundle shape and config reload semantics are not carried through explicitly. |
| Go-native simplicity and maintainability | 16/20 | The bootstrap/core split is clean, but the proposed exported orchestration surface is a little broader than needed. |
| Avoiding overdesign / clean boundaries | 16/20 | The plan mostly respects the core boundary, but it should be more explicit about not turning `internal/cli` into a second state owner. |
| Implementation clarity and testability | 11/15 | The TDD order is usable, but it leaves the actual process entrypoint and memory-mode client injection too implicit. |
| Verification coverage and rollout safety | 9/15 | Package verification is covered, but the plan does not fully pin bootstrap/entrypoint behavior or memory-path isolation. |
| **Total** | **74/100** |  |

## High-Severity Issues

1. **Memory mode is not actually guaranteed to be local/self-contained.**  
   The plan says memory mode should use `workflow.CompatLinearDefaultBundle` directly ([`workspace/T14/final_impl_v1.md:52-53`](/Users/apple/Documents/Github/go-symphony/.worktrees/t14-end-to-end-run-integration/workspace/T14/final_impl_v1.md#L52-L53), [`workspace/T14/final_impl_v1.md:99-101`](/Users/apple/Documents/Github/go-symphony/.worktrees/t14-end-to-end-run-integration/workspace/T14/final_impl_v1.md#L99-L101), [`workspace/T14/final_impl_v1.md:151-157`](/Users/apple/Documents/Github/go-symphony/.worktrees/t14-end-to-end-run-integration/workspace/T14/final_impl_v1.md#L151-L157)).  
   But the current bundle constructor builds a real Linear HTTP client when no client is injected, so a memory-backed run can still depend on live Linear credentials. That makes the promised memory end-to-end path fragile and not truly isolated.

## Medium / Low Issues

1. **The bootstrap / entrypoint boundary is not tested explicitly.**  
   The plan introduces `internal/cli` as the assembly layer ([`workspace/T14/final_impl_v1.md:62-89`](/Users/apple/Documents/Github/go-symphony/.worktrees/t14-end-to-end-run-integration/workspace/T14/final_impl_v1.md#L62-L89)), but the TDD plan never says to prove that `cmd/symphony/main.go` actually calls it or that shutdown/cancellation is wired through the executable. With `cmd/symphony/main.go` still empty in the current repo, this is a real rollout gap.

2. **Config hot reload / last-known-good behavior is not preserved explicitly in the runtime flow.**  
   The design still targets `WORKFLOW.md` loading semantics and last-known-good retention ([`docs/plans/2026-04-10-go-symphony-design.md:10-19`](/Users/apple/Documents/Github/go-symphony/.worktrees/t14-end-to-end-run-integration/docs/plans/2026-04-10-go-symphony-design.md#L10-L19), [`docs/plans/2026-04-10-go-symphony-design.md:51-53`](/Users/apple/Documents/Github/go-symphony/.worktrees/t14-end-to-end-run-integration/docs/plans/2026-04-10-go-symphony-design.md#L51-L53), [`docs/plans/2026-04-10-go-symphony-design.md:93-95`](/Users/apple/Documents/Github/go-symphony/.worktrees/t14-end-to-end-run-integration/docs/plans/2026-04-10-go-symphony-design.md#L93-L95)).  
   The plan says "load config and workflow text" but does not say whether the bootstrap keeps a `config.Store` alive or whether this is intentionally a one-shot load for T14. Without that explicit statement, the implementation could silently drop a previously approved compatibility behavior.

3. **The proposed exported orchestrator surface is broader than the plan needs.**  
   `Snapshot`, `RequestRefresh`, `ApplyRunEvent`, and `Close` are all listed as exported methods ([`workspace/T14/final_impl_v1.md:85-87`](/Users/apple/Documents/Github/go-symphony/.worktrees/t14-end-to-end-run-integration/workspace/T14/final_impl_v1.md#L85-L87)).  
   This is still compatible with the design, but it should be tightened to the smallest API that the bootstrap layer actually needs, otherwise the core will start looking service-oriented instead of state-owned.

## Required Changes Before Acceptance

1. Make memory mode explicitly injectable so it can run without live Linear auth. The plan needs to say how the bootstrap supplies a fake or no-network Linear client in memory runs, or it should stop claiming memory mode is fully local.
2. Add an explicit bootstrap/entrypoint test gate that proves `main` and `internal/cli` build the runtime, perform startup cleanup before dispatch, and shut the process down idempotently.
3. State clearly whether T14 preserves `config.Store` hot reload semantics. If yes, add it to the runtime flow and tests; if no, mark it as a deliberate defer in the task artifacts.

## Verdict

Not ready for acceptance. The architecture direction is mostly correct, but the memory path and the actual executable bootstrap need sharper, testable boundaries before implementation starts.
