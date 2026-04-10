# T10 Spec Review

## High Severity Issues

None.

`workspace/T10/final_impl.md`, the `tracker-reader-memory-adapter` OpenSpec change, and `workspace/T10/test_strategy.md` align on the core `T10` requirements:

- the frozen core tracker surface is read-only and limited to candidate listing, state-based listing, and refresh-by-id reads
- `ListByStates` is included for Symphony contract fidelity, but runtime adoption remains explicitly deferred to later workspace/runtime integration tasks
- the memory adapter scope stays narrow and deterministic, with deep-copy isolation written into both the plan and the spec
- the test strategy explains what the package gate proves, what broader repo verification proves, and why `make test-e2e` is only a command-contract check at this stage
- the change artifacts do not widen scope into `internal/orchestrator`, tracker writes, or provider-specific Linear behavior

## Medium / Low Suggestions

1. Keep the deferred-runtime-adoption note explicit when implementation lands, so later code changes do not quietly pull `internal/orchestrator` work into `T10` without updating the task scope and verification.

2. When the package tests land, make the deep-copy assertions concrete by mutating returned `Labels`, `BlockedBy`, `Priority`, `Routable`, `CreatedAt`, and `UpdatedAt` values individually rather than relying on only one mutation path.
