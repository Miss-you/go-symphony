# T04 New Implementation Review

Current `go-symphony` config handling is still a loader/store pair, not an internal runtime config model. The package entry point in [/Users/lihui/Documents/GitHub/go-symphony/.worktrees/t04-internal-config-model/internal/config/doc.go] only says the package will hold external and internal runtime configuration, but the actual code in [/Users/lihui/Documents/GitHub/go-symphony/.worktrees/t04-internal-config-model/internal/config/workflow.go] and [/Users/lihui/Documents/GitHub/go-symphony/.worktrees/t04-internal-config-model/internal/config/store.go] only parses `WORKFLOW.md`, caches the raw result, and hot-reloads it.

## What Exists Now

- [`internal/config/workflow.go`](/Users/lihui/Documents/GitHub/go-symphony/.worktrees/t04-internal-config-model/internal/config/workflow.go) loads `WORKFLOW.md`, splits front matter from prompt text, normalizes YAML maps recursively, and returns a `Workflow` struct with `Path`, `Config`, `Prompt`, and `PromptTemplate`.
- [`internal/config/store.go`](/Users/lihui/Documents/GitHub/go-symphony/.worktrees/t04-internal-config-model/internal/config/store.go) wraps that raw `Workflow` in a last-known-good cache with tick-based reloads and path switching.
- Tests in [`internal/config/workflow_test.go`](/Users/lihui/Documents/GitHub/go-symphony/.worktrees/t04-internal-config-model/internal/config/workflow_test.go) and [`internal/config/store_test.go`](/Users/lihui/Documents/GitHub/go-symphony/.worktrees/t04-internal-config-model/internal/config/store_test.go) prove only parsing, fallback, and reload behavior.

## What Is Missing For T04

- There is no typed internal config model yet. `Workflow.Config` is still a `map[string]any`, so downstream code would have to keep re-parsing raw YAML shape.
- There is no provider-neutral normalization layer. The current prompt template still says `Linear issue`, and nothing converts the raw workflow config into a stable Go-native shape with neutral names and defaults.
- There is no validation boundary for semantics like tracker kind, polling interval, workspace root, worker limits, codex settings, hooks, observability, or server settings.
- There is no explicit compatibility seam between raw workflow input and runtime settings. `WorkflowStore`-style caching exists, but not a `settings` object that core packages can depend on without reaching into provider-specific YAML.

## Likely Extension Points

- Keep `Workflow` as the raw file loader result, but add a separate normalized config type in `internal/config` that represents runtime settings, defaults, and validation.
- Add a conversion step from `Workflow.Config` into typed settings so the rest of the runtime can consume a stable model instead of a generic map.
- Preserve `Workflow` hot reload and last-known-good behavior in `Store`, but make `Store` cache both the raw workflow and the normalized settings result.
- Treat provider-specific tracker values as compatibility input, then normalize them into neutral fields that later core packages can read without Linear naming.

## T04 Risks

- If the config model stays map-based, `internal/orchestrator`, `internal/runner`, and future `internal/domain` work will keep depending on ad hoc YAML keys.
- The current `defaultPromptTemplate` in [`internal/config/workflow.go`](/Users/lihui/Documents/GitHub/go-symphony/.worktrees/t04-internal-config-model/internal/config/workflow.go) is still Linear-flavored; that is acceptable only if it stays isolated from core runtime logic.
- The Elixir reference implementation in [/Users/lihui/Documents/GitHub/symphony/elixir/lib/symphony_elixir/config.ex](/Users/lihui/Documents/GitHub/symphony/elixir/lib/symphony_elixir/config.ex) and [/Users/lihui/Documents/GitHub/symphony/elixir/lib/symphony_elixir/config/schema.ex](/Users/lihui/Documents/GitHub/symphony/elixir/lib/symphony_elixir/config/schema.ex) shows the target behavior more clearly: raw workflow input is parsed into structured settings with defaults, env fallbacks, and validation. The Go side does not have that second layer yet.

## Bottom Line

T04 is not blocked by parsing or reload mechanics. It is blocked by the absence of a real internal settings model that sits between `WORKFLOW.md` and the rest of the runtime. The safest next step is to introduce that typed normalization layer without changing the raw file loader contract yet.
