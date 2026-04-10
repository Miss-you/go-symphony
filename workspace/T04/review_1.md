# T04 Review

Scores: 24/20/20/12/10
Total: 86/100

## High Severity Issues

None.

## Medium / Low Suggestions

1. The typed settings boundary is still underspecified. `final_impl_v1.md` says the typed model should be the downstream contract and that `Store` may cache derived settings, but it never defines an explicit retrieval API for those settings. As written, later packages can still drift back to `Workflow.Config` and reparse raw YAML, which weakens the point of T04. See [workspace/T04/final_impl_v1.md:33](/Users/lihui/Documents/GitHub/go-symphony/.worktrees/t04-internal-config-model/workspace/T04/final_impl_v1.md#L33) and [workspace/T04/final_impl_v1.md:79](/Users/lihui/Documents/GitHub/go-symphony/.worktrees/t04-internal-config-model/workspace/T04/final_impl_v1.md#L79).

2. The test strategy is too abstract for the most brittle compatibility values. It mentions defaults and env fallbacks, but it does not name the concrete defaults that came from the Elixir schema (`tracker.endpoint`, `polling.interval_ms`, `workspace.root`, `agent.max_turns`, `codex.command`, `server.host`). That leaves room for an incomplete port that still passes the written tests. See [workspace/T04/final_impl_v1.md:86](/Users/lihui/Documents/GitHub/go-symphony/.worktrees/t04-internal-config-model/workspace/T04/final_impl_v1.md#L86).

3. The validation section should be sharper about the current provider matrix. "Accept only the provider kinds supported by the current product scope" is vague enough that an implementation could either overfit to today’s Linear/memory split or postpone provider validation entirely. T04 needs a concrete choice here because the rest of the runtime will read the normalized model. See [workspace/T04/final_impl_v1.md:58](/Users/lihui/Documents/GitHub/go-symphony/.worktrees/t04-internal-config-model/workspace/T04/final_impl_v1.md#L58).
