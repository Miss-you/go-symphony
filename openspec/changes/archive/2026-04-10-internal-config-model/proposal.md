## Why

`T03` landed raw `WORKFLOW.md` loading and last-known-good reload, but the Go port still lacks the typed internal settings layer that Symphony uses to apply defaults, env fallbacks, and semantic validation before runtime code starts. `T04` needs to close that gap now so later core tasks can depend on a provider-neutral config contract instead of reparsing raw YAML maps.

## What Changes

- add a typed `Settings` model under `internal/config` that downstream runtime packages use instead of `Workflow.Config`
- normalize the legacy external `tracker.*` workflow input into neutral `Settings.Provider` fields while preserving the current `WORKFLOW.md` shape
- apply Symphony-compatible defaults, env fallbacks, path handling, and semantic validation in `internal/config`
- extend the reload store to keep raw workflow plus typed settings in one atomic last-known-good snapshot
- add focused unit tests for typed defaults, provider validation, env/path handling, startup failure, and atomic reload fallback behavior
- keep raw workflow loading semantics unchanged from `T03`

## Capabilities

### New Capabilities

- `runtime-config`: typed runtime settings normalization, validation, and reload-safe last-known-good snapshots derived from `WORKFLOW.md`

### Modified Capabilities

- None

## Impact

- `internal/config/`
- later core consumers that will read typed settings instead of raw YAML maps
- `workspace/T04/`
- `docs/plans/2026-04-10-go-symphony-design-task.md`
- `openspec/specs/runtime-config/spec.md`
