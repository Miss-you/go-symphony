# T11 Review 1

## Score

- Symphony alignment and source fidelity: 23 / 30
- Go-native simplicity and maintainability: 16 / 20
- No overdesign / clean boundaries: 17 / 20
- Implementation clarity and testability: 12 / 15
- Verification coverage and landing safety: 11 / 15

Total: 79 / 100

## Findings

1. High severity: `ListByStates` is under-specified in a way that can break cleanup parity. The Elixir source uses this path as a project-scoped, state-only read, returns `[]` for empty normalized input, and does not apply assignee routing. In `final_impl_v1.md`, the state-based read section only says to "use normalized state-name filtering against the same Linear data source" and then explains why the method exists, but it never pins down project scoping, empty-input no-op behavior, or the explicit no-routing rule. That leaves enough room for an implementation to query globally by state or to filter by `Routable`, both of which would diverge from the source contract and can break terminal-workspace cleanup. See `workspace/T11/final_impl_v1.md:91-95`, `workspace/T11/original_impl.md:69-90`, and `workspace/T11/new_impl.md:77-95`.

2. Medium severity: the `Routable` contract is still phrased as a recommendation instead of a requirement. The plan says "Recommended mapping" for `Routable` at `workspace/T11/final_impl_v1.md:168-180`, but the runtime dispatch gate already consumes `domain.WorkItem.Routable` directly. This should be written as mandatory behavior with explicit outcomes for no routing filter, exact match, mismatch, and `me` resolution so the implementation and tests cannot treat it as advisory.

## Required Changes

- Make `ListByStates` explicit: project-scoped, state-only, empty input returns `[]`, and assignee routing is not applied there.
- Upgrade `Routable` from "recommended mapping" to a hard contract with exact outcomes for `nil`, exact-assignee match, mismatch, and `me`.
- Add a test requirement for the `ListByStates` empty-input and no-routing cases so the package gate proves the cleanup path is preserved.

## Blocking Status

Blocking. The current draft leaves enough room for a parity-breaking `ListByStates` implementation.
