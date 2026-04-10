# T05 Review 1

Scores: 27/18/18/14/14
Total: 91/100

## High Severity Issues

None.

The revised `final_impl_v1.md` now covers the main parity risks correctly:

- `WorkItem` keeps the prompt-relevant fields that current Symphony already renders
- blocker and retry semantics are first-class instead of being left implicit
- `Routable` is no longer forced to a false zero-value default
- snapshot and worker-event contracts are explicit without copying orchestrator-private refs into the core

## Medium / Low Suggestions

1. `ActiveRun` deliberately excludes process refs and timer refs, which is correct, but the eventual T06 implementation should keep that line strict. If any goroutine/process bookkeeping starts leaking back into `internal/domain`, the package will lose the “projection-only” role this plan is trying to preserve.

2. The test strategy is on the right track with reflection-based contract tests. It will be stronger if those tests assert the specific prompt-facing `WorkItem` fields that later prompt/template work depends on, especially `Description`, `Labels`, `CreatedAt`, and `UpdatedAt`.

3. `RateLimits` is typed enough for current parity, but the implementation should keep it clearly Codex-scoped. It should not become a generic quota abstraction that later tracker adapters feel pressured to use.
