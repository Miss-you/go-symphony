# T03 Review 2

Scores
- Design fit: 26/30
- Go-native scope: 17/20
- T04 compatibility: 16/20
- Testability: 13/15
- Clarity / traceability: 13/15

Total: 85/100

High Severity Issues
- None. The draft stays within `internal/config` and does not leak the loader into orchestrator or workflow selection, so there is no blocking architectural violation.

Medium / Low Suggestions
- The spec still overreaches on prompt fallback for `T03`. `workspace/T03/new_impl.md:20-26` and `docs/plans/2026-04-10-go-symphony-design.md:198-200` both frame prompt rendering / default prompt fallback as deferred or later-layer work, but `workspace/T03/final_impl_v1.md:47-48` introduces a helper that returns Symphony’s built-in default prompt template. That is a compatibility concern, but it should stay out of the raw loader boundary for `T03` unless the implementation doc also makes the later prompt-building boundary explicit.
- `workspace/T03/final_impl_v1.md:115-176` is stronger than the task board asks for a one-task loader. The store API now includes path mutation, polling internals, stamp tracking, and direct `Current()` reload behavior. That is plausible, but it raises the bar for `T03` and risks pulling test effort away from the core loader/parser semantics that `docs/plans/2026-04-10-go-symphony-design-task.md:36-37` expects to land first. I would trim the spec to the minimum needed to prove load/reload and last-known-good semantics.
- Testability will be much better if the cache/reload code is designed around injectable filesystem and clock seams. `workspace/T03/final_impl_v1.md:51-57` and `117-135` require 1-second polling and content-hash stamp checks, but the document does not say how tests will avoid sleeping or depending on real file timestamps. Add explicit seams for time, stat/hash, and logging so the `internal/config/...` tests can verify reload behavior deterministically.
