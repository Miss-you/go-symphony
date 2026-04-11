# T11 Review 2

## Score

- Symphony alignment and source fidelity: 21/30
- Go-native simplicity and maintainability: 17/20
- No overdesign / clean boundaries: 18/20
- Implementation clarity and testability: 11/15
- Verification coverage and landing safety: 9/15

Total: 76/100

## Findings

1. High severity: `fetch_issues_by_states` parity is under-specified, which risks breaking the cleanup-oriented reader contract.

   The Elixir reference makes `fetch_issues_by_states/1` a distinct read path with three behaviors that matter here: it normalizes state names, returns `{:ok, []}` for empty input, and fetches by project plus states without applying assignee routing (`workspace/T11/original_impl.md:69-76`, `workspace/T11/original_impl.md:106-114`). The Go baseline already freezes `ListByStates` in `TrackerReader` (`workspace/T11/new_impl.md:7-11`), but `final_impl_v1.md` only says "`ListByStates` should use normalized state-name filtering against the same Linear data source" and does not state the project scope, empty-input no-op, or no-routing behavior (`workspace/T11/final_impl_v1.md:91-95`). That leaves enough ambiguity for an implementation to satisfy the letter of the plan while regressing startup cleanup parity.

   Required change: explicitly specify that `ListByStates` is project-scoped, returns an empty slice for empty normalized input, and does not apply assignee routing. If the adapter reuses candidate-query helpers, the plan should say so only for shared GraphQL plumbing, not for routing semantics.

## Blocking Status

Blocking. The plan is close, but T11 should not move past review until the `ListByStates` contract is pinned down with the same precision as candidate fetch and refresh-by-ID behavior.
