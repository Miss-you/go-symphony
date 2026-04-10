## 1. Workspace Package Foundation

- [x] 1.1 Create the `internal/workspace` lifecycle surface and private seams for path handling, hook execution, transport, and directory mutation.
- [x] 1.2 Add deterministic workspace naming helpers that normalize identifiers and derive paths from a resolved workspace root.
- [x] 1.3 Add local path safety checks that canonicalize the root and derived path, reject root equality, reject outside-root paths, and reject symlink escapes.

## 2. Lifecycle Semantics

- [x] 2.1 Implement workspace creation so existing directories are reused, stale non-directory paths are replaced, and `after_create` runs only on first creation.
- [x] 2.2 Implement `RunWithHooks(...)` so `before_run` is fatal, the run body only executes after a successful `before_run`, and `after_run` always executes on every exit path.
- [x] 2.3 Implement `Remove(...)` so `before_remove` is best-effort and workspace removal continues even when the hook fails or times out.
- [x] 2.4 Implement `RemoveIssueWorkspaces(...)` so runtime terminal cleanup and startup sweeps share the same removal path and hostless cleanup fans out across configured worker hosts.

## 3. Verification

- [x] 3.1 Add package-scoped tests for identifier normalization, path determinism, root collision, outside-root rejection, and symlink-escape rejection.
- [x] 3.2 Add package-scoped tests for create/reuse/remove behavior and the full hook policy matrix, including `after_run` on failure paths.
- [x] 3.3 Add package-scoped tests for host-aware cleanup fan-out, structured error categories, and the non-terminal no-cleanup case.
