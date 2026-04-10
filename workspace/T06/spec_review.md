# T06 Spec Review

## High Severity Issues

None.

`workspace/T06/final_impl.md`, the `orchestrator-core` OpenSpec change, and `workspace/T06/test_strategy.md` now align on the core `T06` requirements:

- `internal/orchestrator` is the sole owner of mutable scheduling state and workers report facts only through `domain.RunEvent`
- polling, refresh coalescing, deterministic candidate ordering, retry lineage, claim retention, reconcile, and stall behavior are all specified precisely enough to implement and test
- aggregate `CodexTotals`, latest `RateLimits`, and snapshot ordering semantics are explicit instead of being left to implementation guesswork
- package-private collaborator seams are kept local to `internal/orchestrator`, so `T06` does not prematurely freeze `T07` to `T10` interfaces
- the test strategy explains what `go test ./internal/orchestrator/...`, broader repo verification, and the e2e applicability check each prove

## Medium / Low Suggestions

1. When the package tests land, keep the startup checking/coalescing assertions at the service level instead of letting them collapse into only reducer-level tests. That behavior is part of the accepted runtime contract now.

2. If completion bookkeeping ends up unnecessary once the implementation and tests are green, remove it rather than keeping a private field with no behavioral proof.
