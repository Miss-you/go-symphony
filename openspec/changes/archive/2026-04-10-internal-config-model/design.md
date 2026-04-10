## Context

`T03` established raw `WORKFLOW.md` loading and last-known-good reload semantics in `internal/config`, but the Go port still lacks the typed settings layer that Symphony uses before runtime code starts. Today the only structured output is `Workflow.Config map[string]any`, which would force later core packages to keep reparsing raw YAML and would leak legacy `tracker.*` naming into the provider-neutral core.

The Elixir implementation solves this by parsing workflow front matter into a structured schema with defaults, env fallbacks, path handling, and semantic validation before runtime callers consume settings. `T04` must land the same observable behavior while keeping the Go core provider-neutral and limiting Linear-specific naming to compatibility parsing.

## Goals / Non-Goals

**Goals:**

- introduce a typed `Settings` model in `internal/config`
- keep the external `WORKFLOW.md` shape source-compatible with Symphony
- normalize legacy `tracker.*` workflow input into neutral `Settings.Provider` fields
- centralize defaults, env resolution, path handling, and semantic validation in `internal/config`
- extend the reload store so raw workflow and typed settings update as one atomic last-known-good snapshot
- preserve fail-fast startup when typed config is semantically invalid

**Non-Goals:**

- changing raw workflow parsing or reload behavior from `T03`
- prompt rendering, template strictness, or workflow-bundle selection
- orchestrator, runner, tracker, or CLI wiring beyond consuming typed settings
- a universal tracker write API or broader provider abstraction
- Lark-specific runtime behavior

## Decisions

### 1. Freeze the typed API around `Settings`

`T04` introduces a single downstream typed contract:

- `ParseSettings(workflow Workflow) (Settings, error)`
- `LoadSettings(path string) (Settings, error)`
- `(*Store).CurrentSettings() (Settings, error)`

`(*Store).Current() (Workflow, error)` remains for raw access, but downstream runtime code should treat `Settings` as the only supported typed entry point.

Alternative considered:

- Let later packages normalize `Workflow.Config` ad hoc. Rejected because it duplicates config logic, weakens parity, and leaks raw YAML shape into the core.

### 2. Accept legacy `tracker.*` input, normalize to `Settings.Provider`

The external workflow format remains source-compatible with Symphony, but the internal typed shape becomes provider-neutral:

- accept `tracker.kind`, `tracker.endpoint`, `tracker.project_slug`, `tracker.assignee`, `tracker.active_states`, and `tracker.terminal_states`
- normalize them one-way into `Settings.Provider`
- keep legacy `tracker.*` names confined to the compatibility parser

Supported provider kinds are explicit in `T04`: `linear` and `memory`.

Alternative considered:

- Keep a typed `Tracker` group internally for parity with Elixir. Rejected because `internal/config` is part of the provider-neutral core and should not force later core packages to speak in tracker-specific terms.

### 3. Apply defaults, env resolution, and validation inside `internal/config`

`internal/config` owns all typed normalization steps:

- canonicalize map keys to strings
- drop nil values before defaulting
- apply concrete defaults in typed structs
- resolve `LINEAR_API_KEY`, `LINEAR_ASSIGNEE`, `$VAR`, and `~` handling where Symphony already does
- normalize nested Codex policy maps
- validate provider kind, required Linear fields, positive numeric bounds, and per-state concurrency overrides

Alternative considered:

- Split defaults/env resolution across later consumers. Rejected because it would create semantic drift and make startup validation incomplete.

### 4. Store raw and typed config as one atomic snapshot

The reload store should not cache raw workflow and typed settings independently. It should build a single internal snapshot containing:

- raw `Workflow`
- typed `Settings`
- file stamp
- desired/loaded path bookkeeping

Snapshot replacement happens only after raw parse, typed normalization, and typed validation all succeed. If any step fails, the previous snapshot remains active.

Alternative considered:

- Cache raw workflow first and derive typed settings lazily. Rejected because a partially updated cache can leave raw and typed state out of sync during reload failures.

### 5. Preserve fail-fast startup and conservative reload

Initial config load must fail startup if either raw workflow loading or typed validation fails. Later reloads keep the last known good snapshot and continue retrying the desired path.

Alternative considered:

- Allow startup with a raw workflow plus deferred typed validation. Rejected because it regresses Symphony's existing config boot contract and pushes config failure into unrelated runtime code.

## Risks / Trade-offs

- `Settings.Provider` is a neutral bridge, but the external workflow input is still `tracker.*`. Mitigation: keep the mapping one-way and block downstream reads from `Workflow.Config`.
- Pinning concrete defaults now increases implementation scope slightly. Mitigation: those defaults are observable compatibility behavior and must be asserted explicitly anyway.
- Storing both raw and typed config in one snapshot duplicates some data. Mitigation: it keeps reload semantics coherent and avoids a split-brain cache.
- Supporting `memory` now may seem early, but later tracker/runtime tests depend on it. Mitigation: keep support narrow and limited to the explicit `linear`/`memory` matrix.

## Migration Plan

No user-facing migration is required. The rollout is internal:

1. add `Settings` parsing and validation under `internal/config`
2. extend the store to cache atomic raw+typed snapshots
3. keep raw workflow APIs intact while adding typed retrieval APIs
4. switch later tasks to consume `CurrentSettings()` instead of `Workflow.Config`

Rollback is straightforward: revert the `T04` changes and fall back to the `T03` raw loader/store.

## Open Questions

- None for `T04`. Later tasks may revisit whether typed config retrieval should remain split across `Current()` and `CurrentSettings()`, but that does not block the current implementation.
