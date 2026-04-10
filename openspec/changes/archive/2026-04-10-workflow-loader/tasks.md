## 1. Loader Contract

- [x] 1.1 Add raw workflow types, typed loader errors, and workflow-path resolution helpers under `internal/config`.
- [x] 1.2 Implement file loading and parsing for prompt-only files, YAML front matter files, unterminated front matter, and typed load failures.
- [x] 1.3 Add the narrow blank-prompt compatibility helper while keeping the raw loaded workflow blank-preserving.
- [x] 1.4 Add unit tests that prove the path-resolution, parsing, and blank-prompt helper contract matches the approved `WORKFLOW.md` semantics.

## 2. Reload Store

- [x] 2.1 Implement a reloadable `internal/config` store that caches the active workflow, polls every second, and detects file changes from a stable stamp.
- [x] 2.2 Add deterministic test seams for tick delivery plus file read/stat/hash behavior.
- [x] 2.3 Implement last-known-good semantics for `Current`, `ForceReload`, and workflow-path changes.
- [x] 2.4 Add unit tests that cover valid reload, invalid reload fallback, startup failure without a known-good workflow, and path-change retry behavior.

## 3. Verification and Task Sync

- [x] 3.1 Run `go test ./internal/config/...` and fix any failures.
- [x] 3.2 Update `workspace/T03/test_strategy.md`, `workspace/T03/todo.md`, and the task board from fresh verification evidence.
