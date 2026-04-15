# T19 Final Implementation Review 1

Score: 86/100

| Dimension | Score |
| --- | ---: |
| Symphony alignment/source fidelity | 26/30 |
| Go-native maintainability | 17/20 |
| No overdesign/boundaries | 16/20 |
| Implementation clarity/testability | 13/15 |
| Verification coverage/safety | 14/15 |

High-severity issues: none.

Recommended fixes:

1. Move the `--only-issue` control out of the primary `cmd/symphony` launcher, or make it an explicit verification subcommand instead of a permanent runtime flag. In `workspace/T19/final_impl_v1.md:54-73`, the plan adds a second runtime personality to the normal daemon path. That is workable, but it blurs the operator-verification edge and makes the default CLI semantics harder to reason about. A sibling verification command keeps the smoke path isolated and preserves the production launcher unchanged.

2. Load Linear probe settings through the same store semantics the runtime uses, or document why strict `LoadSettings` parsing is intentional. `workspace/T19/final_impl_v1.md:36-45` says the probe should use `config.LoadSettings`, but the actual runtime path uses `config.NewStore` with last-known-good behavior. If the probe parses the file directly, it can report a config state the runtime would not run under after a failed reload. The safer choice is to reuse the same store path, or at minimum make the probe's divergence explicit and test for it.

3. Add one boundary test that proves the probe binary never reaches runtime startup seams. The plan already says the probe must not import or call `internal/cli.StartRuntime`, `workspace`, `orchestrator`, or `codex` (`workspace/T19/final_impl_v1.md:52`), but the test list only checks argument parsing and provider rejection. A compile-time dependency guard or package-level test would make the isolation requirement enforceable instead of advisory.
