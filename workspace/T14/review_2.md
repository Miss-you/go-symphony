# T14 Review 2

## Score Table

| Rubric | Score | Max | Notes |
| --- | ---: | ---: | --- |
| Symphony alignment and source faithfulness | 23 | 30 | Mostly tracks the Elixir flow, but max-turn exhaustion and event normalization are still too loose. |
| Go-native simplicity and maintainability | 16 | 20 | Thin bootstrap boundary is sensible and avoids a new runtime framework. |
| Avoiding overdesign / clean boundaries | 17 | 20 | Boundaries are disciplined, but `startRun` / `stopRun` ownership still needs sharper behavior wording. |
| Implementation clarity and testability | 9 | 15 | The plan is not yet precise enough to drive tests for the full turn loop. |
| Verification coverage and rollout safety | 8 | 15 | Verification covers broad package scope, but not the behavior-level assertions that matter most here. |

**Total: 73 / 100**

## High-Severity Issues

1. **Max-turn handling is underspecified and can leak run state.**  
   In [workspace/T14/final_impl_v1.md](file:///Users/apple/Documents/Github/go-symphony/.worktrees/t14-end-to-end-run-integration/workspace/T14/final_impl_v1.md#L117), the plan says the runner should continue until the refreshed item is inactive or `agent.max_turns` is reached, but it never says what happens at the cap. The design requires the loop to keep refreshing after turns and only continue while the item is still active; once the cap is hit, the plan needs to say whether the worker exits as a normal completion, a retry, or a terminal stop, and whether the orchestrator releases the claim or leaves it queued. Without that, a complete run can terminate without a defined state transition or cleanup path.

2. **Event normalization is too vague to prove or implement safely.**  
   In [workspace/T14/final_impl_v1.md](file:///Users/apple/Documents/Github/go-symphony/.worktrees/t14-end-to-end-run-integration/workspace/T14/final_impl_v1.md#L118-L133), the plan only names a few coarse event buckets. The design explicitly puts event normalization in `codex` and the runtime vocabulary in `domain.RunEvent`; T14 needs a complete, testable mapping for session start, turn completion/failure/cancel, approval handling, tool calls, unsupported tools, malformed payloads, and how totals/rate limits are projected. As written, the plan leaves the most important observability and retry signals ambiguous, which makes the end-to-end behavior untestable.

## Medium / Low Issues

1. **The test strategy does not explicitly prove the post-turn refresh-by-ID checkpoint.**  
   The plan mentions continuation after a refreshed item stays active, but it does not require a test that proves refresh happens after each completed turn before redispatch. That check is central to avoiding stale continuation loops.

2. **The verification plan is broad, but it does not call out direct assertions for the runtime event vocabulary.**  
   The package sweep can pass while the worker still drops or mislabels `domain.RunEvent` kinds. T14 should include a focused regression that asserts the normalized event stream, not just a successful turn transcript.

## Required Changes Before Acceptance

1. Define the exact `agent.max_turns` exit behavior and state transition, including whether the run is retried, released, or marked terminal.
2. Add a complete event-normalization matrix from `codex.Event` to `domain.RunEvent`, including tool calls, approvals, malformed messages, and the end-of-turn / end-of-run split.
3. Add a dedicated test case that proves refresh-by-ID runs after each completed turn before the next continuation decision.
4. Add a targeted regression test for normalized runtime events and totals/rate-limit projection, so the acceptance gate proves more than compile-time wiring.

## Verdict

**Not accepted yet.** The plan is close on structure, but the turn-loop edge cases that decide correctness are still too vague for T14. Fix the max-turn and event-normalization wording first, then re-review.
