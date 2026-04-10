# T05 Review 2

Scores: 28/19/17/14/13
Total: 91/100

## High Severity Issues

None.

The current plan is faithful to the approved design: it keeps the core provider-neutral, treats the orchestrator as the sole mutable runtime owner, and freezes the domain types that later orchestration and observability work will need.

## Medium / Low Suggestions

1. The plan is right to use `time.Time` and `time.Duration` in the core, but T06 and later compatibility-shell tasks should be disciplined about converting those values only at presentation boundaries. If milliseconds leak back into `internal/domain`, the contract will drift toward transport concerns.

2. `RunEvent` is intentionally small, which is good, but T09 should keep Codex event normalization centralized in `internal/codex` rather than spreading event-shape interpretation across orchestrator and dashboard code.

3. The proposed `WorkItem` shape is appropriately narrow for V1, but later tasks should resist adding provider identity or raw metadata bags just because another adapter wants convenience fields. The current exclusions section should be treated as a hard boundary, not a soft preference.
