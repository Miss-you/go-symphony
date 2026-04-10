## Why

`go-symphony` already carries workspace root and hook configuration, and the orchestrator already knows when terminal cleanup is required, but the actual workspace lifecycle boundary is still missing. This change closes that gap so workspace naming, path safety, hook ordering, and cleanup semantics match the approved T07 scope instead of being implied by later runner or protocol work.

## What Changes

- Add a dedicated workspace lifecycle capability for deterministic naming, safe identifier normalization, workspace path validation, lifecycle hook execution, and create/reuse/remove semantics.
- Preserve the Elixir-visible hook asymmetry: `after_create` and `before_run` are blocking, while `after_run` and `before_remove` are best-effort.
- Preserve terminal-state cleanup behavior driven by orchestrator intent, including host-aware cleanup fan-out when no explicit host is supplied.
- Keep SSH transport, runner policy, Codex protocol handling, and tracker write behavior out of this change.
- Add focused tests that lock path safety, hook behavior, and cleanup semantics without widening the core runtime boundary.

## Capabilities

### New Capabilities
- `workspace-lifecycle`: Workspace path derivation, safety checks, lifecycle hook semantics, create/reuse/remove behavior, and terminal cleanup handoff.

### Modified Capabilities
- None.

## Impact

- Adds a new `openspec/specs/workspace-lifecycle/spec.md` capability contract.
- Drives `internal/workspace` implementation and its package-scoped tests.
- Clarifies the lifecycle handoff between `internal/orchestrator` cleanup intent and workspace removal behavior.
- Does not change runner, Codex, tracker, or dashboard contracts.
