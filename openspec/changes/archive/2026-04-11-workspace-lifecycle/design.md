## Context

`go-symphony` already normalizes workspace root and hook configuration in `internal/config`, and the orchestrator already carries workspace paths plus cleanup intent. What is still missing is a concrete workspace lifecycle boundary that preserves the Elixir-visible semantics without freezing runner, Codex, or tracker-write behavior too early.

The current repository state is intentionally split: `internal/workspace` is still a placeholder, while `internal/orchestrator` already knows when a run should be cleaned up. T07 needs to turn that split into a narrow, testable lifecycle package.

## Goals / Non-Goals

**Goals:**
- Define a small `internal/workspace` boundary for deterministic workspace naming, path safety, lifecycle hooks, create/reuse/remove semantics, and terminal cleanup handoff.
- Preserve the Elixir hook asymmetry: `after_create` and `before_run` are blocking, while `after_run` and `before_remove` are best-effort.
- Preserve host-aware cleanup fan-out when no explicit host is supplied, while keeping concrete SSH transport policy outside the workspace contract.
- Keep the package provider-neutral and compatible with later orchestrator/runner integration.

**Non-Goals:**
- Do not add runner execution policy, SSH topology selection, or capacity management.
- Do not add Codex protocol parsing or tracker read/write semantics.
- Do not create a generic filesystem abstraction or provider-agnostic workflow system.

## Decisions

1. **Use a dedicated workspace lifecycle package, not a broad runner/service abstraction.**
   - Rationale: T07 is a lifecycle boundary, not an execution-host boundary. A small package keeps path safety, hook ordering, and cleanup semantics crisp.
   - Alternatives considered:
     - Fold the behavior into `internal/orchestrator`. Rejected because it would mix scheduling ownership with filesystem lifecycle.
     - Build a generic runtime service. Rejected because it would freeze abstractions needed later by T08/T09.

2. **Make path safety explicit inside `internal/workspace`.**
   - Rationale: config already resolves env vars and `~`, but the workspace package must still canonicalize the resolved root and derived local path, reject root equality, reject outside-root paths, and reject symlink escapes.
   - Alternatives considered:
     - Trust config-level expansion alone. Rejected because it weakens the safety boundary.
     - Defer all validation to callers. Rejected because the workspace boundary would no longer be authoritative.

3. **Preserve Elixir hook asymmetry with a single run-phase helper.**
   - Rationale: `RunWithHooks(...)` can lock the `before_run` / run body / `after_run` ordering and guarantee `after_run` runs on every exit path without inventing separate policies at call sites.
   - Alternatives considered:
     - Expose separate hook calls only. Rejected because callers could easily forget the `after_run` finally-equivalent.
     - Fold hook execution into orchestrator callbacks. Rejected because it pushes lifecycle policy out of the package that owns it.

4. **Keep host-aware terminal cleanup as part of the lifecycle contract, but keep transport machinery private.**
   - Rationale: Elixir parity includes fan-out cleanup when no explicit host is supplied, so the workspace contract must preserve that observable behavior. The actual SSH mechanics can stay behind package-private seams until T08.
   - Alternatives considered:
     - Make cleanup local-only for T07. Rejected because it would leave parity gaps.
     - Export a full remote execution interface now. Rejected because it would prematurely freeze the T08 boundary.

## Risks / Trade-offs

- [Risk] Host-aware cleanup semantics can be over-coupled to transport details if the package interface expands too far. → [Mitigation] Keep transport private and expose only the lifecycle contract needed by orchestrator callers.
- [Risk] Root canonicalization can become inconsistent if path validation is split between config and workspace. → [Mitigation] Make `internal/workspace` the final authority for canonical root/path safety.
- [Risk] A loose hook API could let callers skip `after_run` on error paths. → [Mitigation] Use one run-phase wrapper that owns the full ordering and always executes `after_run`.

## Migration Plan

1. Add the `internal/workspace` package and its tests.
2. Land the workspace lifecycle contract and keep it isolated from runner/Codex/tracker code.
3. Wire orchestrator and later runtime layers to call the workspace package through the approved lifecycle entry points.
4. Keep rollback simple: because this is a new package boundary, removal is mostly revert-by-change rather than data migration.

## Open Questions

- None that change the approved T07 scope. The remaining work is implementation detail, not product direction.
