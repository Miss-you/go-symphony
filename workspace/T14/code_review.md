# T14 Code Review

## Scope

Reviewed the T14 runtime assembly implementation against:

- `workspace/T14/final_impl.md`
- `workspace/T14/test_strategy.md`
- `openspec/changes/end-to-end-run-integration/specs/end-to-end-run-integration/spec.md`
- the current diff in `cmd/symphony`, `internal/cli`, and `internal/orchestrator`

## Findings

### Fixed: prompt conditionals leaked template markers

The first implementation of `renderPrompt` removed only part of the default `{% if issue.description %}` block. Real first-turn prompts could include raw `{% if %}`, `{% else %}`, or `{% endif %}` markers.

Resolution:

- Replaced the partial marker removal with `renderIssueDescriptionConditional`.
- Added `TestRenderPromptHandlesDefaultDescriptionConditional`.

### Fixed: worker runs could mix config snapshots

The worker loop re-read `config.Store` at run time while the tracker reader, workspace manager, and orchestrator were built from the startup settings. A workflow file change between startup and dispatch could make one run combine old provider/runtime wiring with a newer prompt or settings snapshot.

Resolution:

- Captured the startup `config.Workflow` and `config.Settings` in `workerManager`.
- Used that captured snapshot for bundle selection, Codex config, max-turn handling, and continuation refresh state checks.
- Added `TestRuntimeUsesStartupWorkflowSnapshotForWorkerPrompts`, which mutates `WORKFLOW.md` before dispatch and verifies the worker still uses the startup prompt.

### Fixed: executable shutdown path was not reachable through normal signals

`cli.Main` already shuts down on context cancellation, but `cmd/symphony` originally passed `context.Background()` directly. That meant normal SIGINT/SIGTERM process use would not enter the runtime close path.

Resolution:

- `cmd/symphony/main.go` now uses `signal.NotifyContext` for `os.Interrupt` and `SIGTERM`.
- `cli.Main` now handles nil contexts defensively and prints startup errors to stderr.

## Residual Risk

No blocking review findings remain for T14.

Remaining intentional limits are tracked in `workspace/T14/todo.md`.
