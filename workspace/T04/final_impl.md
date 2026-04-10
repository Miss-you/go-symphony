# T04 Final Implementation

## Review Gate

`final_impl_v1.md` required two review rounds.

Round 1 findings:

- high severity: startup fail-fast behavior for semantically invalid typed config was not explicit
- high severity: raw workflow state and typed settings state could drift because the store update contract was not atomic

Round 2 outcome after revision:

- `review_1_round2.md`: 92 / 100, no high-severity issues
- `review_2_round2.md`: 91 / 100, no high-severity issues
- average: 91.5 / 100

Acceptance decision:

- average score exceeds the `>= 80` threshold
- no reviewer reports a remaining high-severity issue
- remaining notes are implementation-discipline items, not design blockers

## Final Scope

`T04` introduces a typed, provider-neutral settings layer in `internal/config` on top of the raw `Workflow` loader/store landed in `T03`.

It must provide:

- a typed `Settings` model that downstream runtime packages use instead of reparsing `Workflow.Config`
- compatibility parsing from legacy external `tracker.*` YAML into neutral `Settings.Provider` fields
- normalization for defaults, env fallbacks, path handling, and nested Codex policy maps
- semantic validation for supported provider kinds and required values
- fail-fast startup behavior when typed config is semantically invalid
- atomic last-known-good snapshot behavior for raw workflow plus typed settings during reload
- typed retrieval APIs alongside the existing raw workflow API

It must not provide:

- changes to the raw `Workflow` file-loading contract from `T03`
- prompt rendering or template strictness
- workflow-bundle selection
- orchestrator, runner, tracker, or CLI integration beyond consuming `Settings`
- a universal tracker write API or broader provider abstraction
- any Lark-specific runtime behavior

## Final Design

### Typed boundary

Keep the raw loader result separate from the typed runtime settings:

```go
type Workflow struct {
	Path           string
	Config         map[string]any
	Prompt         string
	PromptTemplate string
}

type Settings struct {
	Provider      ProviderSettings
	Polling       PollingSettings
	Workspace     WorkspaceSettings
	Worker        WorkerSettings
	Agent         AgentSettings
	Codex         CodexSettings
	Hooks         HookSettings
	Observability ObservabilitySettings
	Server        ServerSettings
}
```

`Settings` is the only typed config contract that downstream runtime packages should consume.

Freeze the `internal/config` typed API in `T04`:

- `ParseSettings(workflow Workflow) (Settings, error)`
- `LoadSettings(path string) (Settings, error)`
- `(*Store).Current() (Workflow, error)` remains the raw workflow accessor
- `(*Store).CurrentSettings() (Settings, error)` becomes the typed accessor

No downstream package should read `Workflow.Config` directly after `T04`.

### Compatibility mapping

The external workflow format remains source-compatible with Symphony:

- accept `tracker.kind`, `tracker.endpoint`, `tracker.project_slug`, `tracker.assignee`, `tracker.active_states`, and `tracker.terminal_states`
- normalize those fields one-way into `Settings.Provider`
- keep legacy `tracker.*` names confined to the compatibility parser so the typed model does not grow a second naming dialect

Supported provider kinds in `T04` are explicit:

- `linear`
- `memory`

`linear` requires credentials/project information. `memory` does not.

### Normalization and validation

Normalize `Workflow.Config` into `Settings` with these steps:

1. Canonicalize keys to string form.
2. Drop nil values so absent values can use typed defaults.
3. Cast into typed compatibility input structures.
4. Apply typed defaults centrally in `internal/config`.
5. Resolve env-backed secrets and path tokens.
6. Normalize Codex policy maps recursively.
7. Validate semantics and return a typed error on failure.

Validation lives in `internal/config` and must cover:

- provider kind is present and supported
- Linear requires API key and project slug
- positive integer bounds for polling, concurrency, retries, hooks, and Codex timeouts
- non-blank state names and positive per-state limits
- unsupported shapes remain configuration errors, not silent fallback

### Path, env, and defaults

Keep the raw workflow path semantics from `T03`: explicit workflow path first, otherwise `<cwd>/WORKFLOW.md`. Do not add new normalization of the workflow file path itself.

For typed settings:

- `workspace.root` resolves `$VAR` before path handling
- local `workspace.root` expands `~`
- missing or empty env-backed workspace root falls back to the default workspace root
- `tracker.api_key` falls back to `LINEAR_API_KEY` when omitted or env-referenced
- `tracker.assignee` falls back to `LINEAR_ASSIGNEE` when omitted or env-referenced

Pin the concrete defaults inherited from the Symphony reference:

- `Provider.Endpoint = "https://api.linear.app/graphql"`
- `Provider.ActiveStates = ["Todo", "In Progress"]`
- `Provider.TerminalStates = ["Closed", "Cancelled", "Canceled", "Duplicate", "Done"]`
- `Polling.IntervalMS = 30000`
- `Workspace.Root = filepath.Join(os.TempDir(), "symphony_workspaces")`
- `Agent.MaxConcurrentAgents = 10`
- `Agent.MaxTurns = 20`
- `Agent.MaxRetryBackoffMS = 300000`
- `Codex.Command = "codex app-server"`
- `Codex.ThreadSandbox = "workspace-write"`
- `Codex.TurnTimeoutMS = 3600000`
- `Codex.ReadTimeoutMS = 5000`
- `Codex.StallTimeoutMS = 300000`
- `Hooks.TimeoutMS = 60000`
- `Observability.DashboardEnabled = true`
- `Observability.RefreshMS = 1000`
- `Observability.RenderIntervalMS = 16`
- `Server.Host = "127.0.0.1"`

### Store contract

`Store` becomes the owner of an atomic last-known-good snapshot that contains both raw `Workflow` and typed `Settings`.

Rules:

- `NewStore` must raw-load, normalize, validate, and only then return a running store
- if any of those startup steps fail, startup fails
- reload success requires raw parse plus typed normalize/validate to all succeed
- if raw parsing succeeds but typed normalization fails, the reload still fails and the previous snapshot remains active
- snapshot replacement is atomic, so raw and typed state cannot drift apart
- path-switch retry semantics from `T03` remain unchanged: the last known good snapshot stays active while future polls keep retrying the desired path

## Test Focus

`go test ./internal/config/...` must prove:

- all `T03` raw workflow loader/store behavior still passes unchanged
- `LoadSettings` and `ParseSettings` return typed `Settings` without callers reparsing `Workflow.Config`
- representative workflow input produces the exact defaults listed above
- env fallback behavior works for `LINEAR_API_KEY`, `LINEAR_ASSIGNEE`, and `$VAR` path values
- `workspace.root` covers `~` expansion and missing-vs-empty env token behavior
- `linear` and `memory` normalize successfully, while unsupported provider kinds fail validation
- missing Linear requirements and invalid numeric bounds fail validation
- startup fails when raw workflow is parseable but typed config is semantically invalid
- reload keeps the last-known-good raw workflow and typed settings together on invalid changes
- raw and typed config state never drift because snapshot replacement is atomic

## Deferred To Later Tasks

- prompt rendering and template strictness move to later workflow/prompt tasks
- `internal/workflow` remains the place for workflow-bundle selection, including the Linear compatibility bundle
- any broader provider abstraction beyond current config needs should wait for evidence from later tasks
