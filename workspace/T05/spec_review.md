# T05 Spec Review

## High Severity Issues

None.

`workspace/T05/final_impl.md`, the `domain-model` OpenSpec change, and `workspace/T05/test_strategy.md` now align on the core requirements:

- the exported helper types that remain in scope (`ActiveRun`, `CodexTotals`, and the rate-limit structs) are concrete runtime projection shapes backed by contract tests rather than speculative future-proofing
- `WorkItem` keeps current prompt-visible and orchestration fields for compatibility, while still excluding provider config and write semantics
- blocker, retry, polling, snapshot, and worker-event semantics are captured explicitly without turning `internal/domain` into an orchestrator-private state dump
- the test strategy explains how package-level contract tests, repo-wide compile safety, and canonical build/lint gates each prove a distinct part of the task

## Medium / Low Suggestions

1. If later tasks widen `internal/domain` beyond the currently landed exported helpers, they should do it in the same change as the corresponding contract-test updates rather than letting the surface grow incidentally.

2. `Routable` should stay documented as an adapter-computed eligibility hint with explicit nil semantics. The orchestrator still owns the full dispatch decision based on state, blockers, and concurrency.
