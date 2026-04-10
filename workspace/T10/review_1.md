# T10 Review 1

High severity:
- `workspace/T10/final_impl_v1.md:16-17,129-167` asks T10 to make an internal `internal/orchestrator` wiring change, but the task board gate for T10 is still only `go test ./internal/tracker/... ./internal/trackers/memory/...` in `docs/plans/2026-04-10-go-symphony-design-task.md:43`. That leaves the package the plan wants to modify uncompiled and untested under the declared gate. Either keep T10 strictly to the tracker/memory packages or expand the formal verification gate to include the orchestrator package before this is implementation-ready.

Medium severity:
- `workspace/T10/final_impl_v1.md:116-123` says the memory reader returns fresh copies and cannot leak shared mutable state, but `domain.WorkItem` contains slices and pointers. The draft never states deep-copy semantics for `Labels`, `BlockedBy`, or the pointer fields, so a shallow copy would still let callers mutate adapter-internal seed data. Tighten the contract and add tests that prove nested fields are isolated.

Scores:
- Symphony alignment and source fidelity: 24/30
- Go-native simplicity and maintainability: 16/20
- Avoiding overdesign / clean boundaries: 15/20
- Implementation clarity and testability: 11/15
- Verification coverage and landing safety: 9/15
- Total: 75/100

Gate:
- Does not pass. The draft is directionally right, but the orchestrator wiring/verification mismatch is a blocking planning issue, and the copy semantics need to be made explicit before implementation.
