## 1. Typed Settings Model

- [x] 1.1 Add the typed `Settings` model and nested typed groups under `internal/config`, using neutral `Provider` naming instead of a downstream `Tracker` type.
- [x] 1.2 Add the compatibility parser that accepts legacy `tracker.*` workflow input and normalizes it one-way into `Settings.Provider`.
- [x] 1.3 Implement `ParseSettings` and `LoadSettings` so callers can obtain typed settings without reparsing `Workflow.Config`.

## 2. Defaults, Resolution, And Validation

- [x] 2.1 Implement Symphony-compatible defaults for provider endpoint/state lists, polling, workspace root, agent limits, Codex settings, hooks timeout, observability settings, and server host.
- [x] 2.2 Implement env fallback and path handling for `LINEAR_API_KEY`, `LINEAR_ASSIGNEE`, `$VAR`, missing-vs-empty env values, and local `~` workspace roots.
- [x] 2.3 Implement typed validation for supported provider kinds, required Linear fields, positive numeric bounds, and per-state concurrency overrides.

## 3. Atomic Reload Snapshot

- [x] 3.1 Extend the reload store to build and cache one atomic snapshot containing raw `Workflow`, typed `Settings`, and reload bookkeeping.
- [x] 3.2 Preserve fail-fast startup when typed config is semantically invalid and preserve the last-known-good snapshot on reload failure.
- [x] 3.3 Add `CurrentSettings()` while keeping the existing raw `Current()` workflow accessor intact.

## 4. Verification And Task Sync

- [x] 4.1 Add focused unit tests for typed defaults, env/path handling, supported provider kinds, startup failure, and atomic reload fallback behavior.
- [x] 4.2 Run `go test ./internal/config/...` and fix any failures.
- [x] 4.3 Update `workspace/T04/test_strategy.md`, `workspace/T04/todo.md`, and the task board from fresh verification evidence.
