# T10 Review 2

Scores: 24/17/15/12/10
Total: 78/100

## High Severity Issues

1. The draft expands T10 beyond the tracker/memory packages, but it does not raise the verification gate to cover the touched runtime path. `final_impl_v1.md` makes the orchestrator wiring adjustment part of scope (`lines 16, 131-138`), while the formal gate still only requires `go test ./internal/tracker/... ./internal/trackers/memory/...` (`lines 154-167` and the task board at `docs/plans/2026-04-10-go-symphony-design-task.md:43`). Since `internal/orchestrator/service.go` currently owns the private scheduler seams (`listCandidates`, `refreshItems`) that this change would alter, the plan can land with green tracker tests and still break the runtime bridge. Either remove the orchestrator change from T10 or add an orchestrator compile/test gate that proves the bridge still works.

## Medium / Low Suggestions

1. `ListByStates` is defensible from the Symphony spec, but it is broader than the current T10 task-board acceptance criteria and pulls cleanup-oriented API surface into this task earlier than the board asks (`final_impl_v1.md:37-50, 75-85`). That is a scope-risk tradeoff worth calling out explicitly in the task doc so later work does not treat it as settled T10 behavior by accident.
