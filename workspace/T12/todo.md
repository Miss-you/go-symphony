# T12 Residual Notes

## Review Disposition

No blocking code review issues remain.

## Residual Risks

- Linear client error classification in `internal/toolbridge/linear` intentionally avoids importing `internal/trackers/linear` so the bridge does not inherit read-adapter dependencies. The current implementation recognizes status/request failures through generic shape checks; if the Linear client error shape changes later, this mapping should be revisited.
- Codex tests pin raw string arguments and `response.result.contentItems`. They do not add a separate mixed-shape assertion for a tool result that sets both `Result` and `ContentItems`; no current T12 behavior requires that mixed shape.
- The dependency guard proves the current `internal/toolbridge/linear` package graph does not include core tracker/domain/orchestrator packages. Future helpers added under this package should keep the same guard meaningful.

## Deferred Scope

- Runtime assembly that injects the Linear bridge into workflow-selected Codex sessions belongs to T13/T14.
- Full Linear live e2e behavior remains outside T12 because this task lands the bridge and protocol boundary, not the end-to-end runner wiring.
