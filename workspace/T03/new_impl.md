# T03 New Impl Research Note

## Current State

- `internal/config` is only a placeholder package today (`internal/config/doc.go`); there is no parser, loader, cache, or hot-reload logic yet.
- `internal/workflow`, `internal/orchestrator`, `internal/domain`, and the other runtime packages are also placeholders only, so there is no existing workflow-loading path to extend.
- The executable entrypoint is still minimal (`cmd/symphony/main.go` contains only `main()`), which means `WORKFLOW.md` semantics are not wired into startup yet.
- The approved design already states that `config` owns `WORKFLOW.md` parsing, prompt/template loading, hot reload, and last-known-good retention, so T03 is still pure implementation work rather than a refactor.

## Existing Constraints

- The design keeps the core provider-neutral and explicitly places `WORKFLOW.md` handling in `internal/config`, not in `orchestrator` or `domain`.
- `internal/workflow` is reserved for workflow selection and workflow bundles, including the Linear compatibility bundle, not for file parsing or cache policy.
- The system must preserve user-facing parity with Symphony, including invalid reload fallback behavior, so the loader cannot replace a known-good workflow with a broken one.
- T03 must not introduce a generic tracker workflow or a new provider-neutral abstraction beyond what the design already allows.

## Proposed Package Landing Zone

- Keep the primary loader API in `internal/config`.
- Shape `internal/config` around a small set of responsibilities:
  - read `WORKFLOW.md`
  - render templates / prompts
  - cache the last known good result
  - expose a reloadable config snapshot to the rest of the runtime
- Let `internal/workflow` consume the normalized output from `internal/config` and decide which workflow bundle to activate.
- Keep file-watching or refresh triggers out of the initial loader API unless a later task proves they are required; the first version should focus on deterministic load/reload semantics.

## Risks / Unknowns

- The exact Symphony fallback semantics for malformed or missing `WORKFLOW.md` still need source-level confirmation before implementation is locked.
- It is unclear yet whether reload should be file-watch driven, explicit-trigger driven, or both; that choice affects how much state belongs in `internal/config` versus `internal/orchestrator`.
- Template rendering can easily drift into workflow-specific behavior if the API is not kept narrow.
- If the loader caches too much derived state, future workflow changes may become harder to isolate and test.

