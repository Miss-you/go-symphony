# T04 Final Implementation v1

## Goal

Introduce a typed, provider-neutral internal config model in `internal/config` so the rest of `go-symphony` can consume stable settings instead of re-reading raw `WORKFLOW.md` YAML. This task sits on top of the raw loader/store landed in `T03` and keeps `WORKFLOW.md` compatibility intact.

## Non-Goals

- No change to the raw `Workflow` loader contract from `T03`.
- No prompt rendering or issue interpolation.
- No workflow bundle selection.
- No orchestrator, runner, tracker, or CLI wiring beyond consuming the normalized settings shape.
- No universal tracker write API or provider-agnostic workflow abstraction.
- No Lark-specific runtime behavior.

## Required Compatibility Behavior

- Preserve `WORKFLOW.md` as the source of truth for runtime config.
- Keep raw file parsing and hot reload semantics from `T03`, including last-known-good retention on reload failure.
- Accept the current external YAML shape used by Symphony, including Linear-flavored fields and defaults.
- Keep Linear-specific compatibility at the edges, not in the core runtime model.
- Preserve existing env fallback behavior for `LINEAR_API_KEY`, `LINEAR_ASSIGNEE`, and `$VAR` path indirection where Symphony already supports it.
- Reject startup if the initial raw load succeeds but typed normalization/validation fails; `T04` must preserve Symphony's fail-fast config boot behavior.

## Proposed Go Model

Keep the raw loader result separate from the typed runtime settings:

- `Workflow` remains the parsed file payload:
  - `Path`
  - `Config map[string]any`
  - `Prompt`
  - `PromptTemplate`
- Add a new normalized settings type in `internal/config`:
  - `Settings`
  - nested typed groups for `Provider`, `Polling`, `Workspace`, `Worker`, `Agent`, `Codex`, `Hooks`, `Observability`, and `Server`

`Settings` is the only typed config contract that downstream runtime packages should consume. The raw map stays available only as an input to normalization and for compatibility/debugging.

`Settings.Provider` is the provider-neutral replacement for the legacy external `tracker` block. The compatibility parser still accepts `tracker.kind`, `tracker.endpoint`, `tracker.project_slug`, `tracker.assignee`, `tracker.active_states`, and `tracker.terminal_states`, but it normalizes them into neutral fields under `Provider`.

Freeze the package surface in `T04`:

- `ParseSettings(workflow Workflow) (Settings, error)`
- `LoadSettings(path string) (Settings, error)`
- `(*Store).Current() (Workflow, error)` remains for raw workflow access
- `(*Store).CurrentSettings() (Settings, error)` becomes the typed retrieval API

No downstream package should read `Workflow.Config` directly after `T04`.

## Normalization Strategy

Normalize in one pass from `Workflow.Config` into `Settings`:

- Canonicalize keys to a consistent string form before casting.
- Drop nil values so absent settings can fall back to defaults.
- Apply defaults in the typed model, not in callers.
- Resolve env-backed secrets and path tokens during finalization.
- Preserve provider-specific input names only inside the compatibility parser, then expose neutral typed fields to callers.

Normalization should keep the current Symphony semantics where they are already observable:

- `tracker.kind` remains accepted on input.
- `tracker.kind` normalizes to `Settings.Provider.Kind`.
- `LINEAR_API_KEY` and `LINEAR_ASSIGNEE` remain the default env fallbacks for Linear runs.
- `workspace.root` keeps `$VAR` indirection and `~` expansion behavior.
- `codex.approval_policy`, `codex.thread_sandbox`, and `codex.turn_sandbox_policy` remain pass-through Codex config values with only shape validation.

## Validation Strategy

Validate semantics at the typed layer after casting:

- Require provider kind input via the legacy `tracker.kind` field.
- Accept exactly `linear` and `memory` in `T04`.
- Require Linear credentials/project slug when `Settings.Provider.Kind == "linear"`.
- Allow `memory` without Linear credentials so later memory-backed test/runtime work is not blocked.
- Validate integer bounds for polling, agent limits, hook timeout, and Codex timeouts.
- Validate that per-state concurrency overrides are positive and keyed by non-blank state names.
- Treat invalid front matter or unsupported shapes as configuration errors, not silent fallback.

Validation belongs in `internal/config`; downstream packages should receive a ready-to-use typed object or a clear error.

Startup and reload rules are explicit:

- `NewStore` must load raw workflow, normalize to `Settings`, validate, and only then return a running store.
- If any step fails during initial load, startup fails.
- Reload success requires both raw parsing and typed normalization/validation to succeed.
- If raw parsing succeeds but typed normalization/validation fails, the reload still counts as failed and the last known good snapshot remains active.

## Path, Env, and Defaults

- Default workflow path stays `WORKFLOW.md` in the current working directory unless an explicit path is provided.
- `workspace.root` should resolve env references before path handling, expand `~` for local paths, and fall back to the configured default workspace root when the resolved value is missing or empty.
- `tracker.api_key` should fall back to `LINEAR_API_KEY` when the workflow omits it or uses the env reference token.
- `tracker.assignee` should fall back to `LINEAR_ASSIGNEE` when omitted or env-referenced.
- Keep the raw `Workflow` path semantics from `T03`; do not add new normalization of the workflow file path itself.

The typed model must pin the concrete defaults inherited from the Symphony reference:

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

## Relationship To `Workflow` And `Store`

- `Workflow` remains the raw loader type.
- `Store` becomes the owner of an atomic last-known-good config snapshot that contains both `Workflow` and `Settings`.
- Internally, the store should swap snapshot state only after raw parse + typed normalize + typed validate all succeed.
- Public readers should expose raw and typed access separately:
  - `Current()` returns the cached raw `Workflow`
  - `CurrentSettings()` returns the cached `Settings`
- Path-switch semantics from `T03` remain: if the desired workflow path changes and the new path is broken, the old snapshot stays active while future polls/current calls keep retrying the desired path.

## Test Focus

`go test ./internal/config/...` should prove:

- raw `Workflow` loading still behaves exactly as in `T03`
- typed settings parse the concrete defaults listed above from representative `WORKFLOW.md` front matter
- env fallbacks resolve for `LINEAR_API_KEY`, `LINEAR_ASSIGNEE`, and `$VAR` path values
- `workspace.root` covers `~` expansion plus missing-vs-empty env token behavior
- unsupported provider kinds fail validation, while `linear` and `memory` normalize successfully
- missing Linear requirements and invalid numeric bounds fail validation
- `LoadSettings` and `CurrentSettings` let callers consume typed config without reparsing raw YAML
- startup fails when raw workflow is parseable but typed config is semantically invalid
- store reload keeps the last-known-good raw workflow and typed settings together on invalid reloads
- raw/typed state never drifts because snapshot replacement is atomic

## Deferred Items

- Prompt rendering and template strictness move to later workflow/prompt tasks.
- `internal/workflow` remains the place for workflow bundle selection, including the Linear compatibility bundle.
- Any broader provider abstraction beyond the current config needs should wait for evidence from later tasks.
