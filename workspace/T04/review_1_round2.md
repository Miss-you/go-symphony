# T04 Review Round 2

Scores: 28/18/18/14/14
Total: 92/100

## High Severity Issues

None.

The two blockers from the first round are resolved in `workspace/T04/final_impl_v1.md`:

- startup is now explicitly fail-fast when typed normalization/validation fails after a successful raw load
- raw and typed config state now update as an atomic snapshot, so reloads cannot leave them out of sync

## Medium / Low Suggestions

1. The typed settings contract is now much clearer, but the plan still relies on `Settings.Provider` as the neutral output while accepting legacy `tracker.*` input. That is workable, but the implementation should keep the mapping one-way and avoid reintroducing `Workflow.Config` reads elsewhere.

2. The test matrix is stronger now because it names the concrete defaults, but it still needs to be implemented with exact assertions on representative fields like `Provider.Endpoint`, `Polling.IntervalMS`, `Workspace.Root`, `Agent.MaxTurns`, `Codex.Command`, and `Server.Host`. That is important to keep the port from drifting on the defaults that matter most.

3. The validation story is now specific enough to land, with `linear` and `memory` named explicitly. The only remaining risk is accidental duplication between typed validation and later runtime consumers, so the implementation should keep provider checks centralized in `internal/config`.
