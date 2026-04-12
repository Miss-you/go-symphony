## 1. Align CI Traceability

- [x] 1.1 Create the CI task board and claim `CI01` in the isolated worktree.
- [x] 1.2 Record original Symphony CI behavior, current Go CI behavior, and the accepted final implementation.
- [x] 1.3 Record the accepted action-version decision in workspace and task-board artifacts.

## 2. Validate Existing Workflow

- [x] 2.1 Confirm `.github/workflows/ci.yml` matches the approved CI design.
- [x] 2.2 Parse `.github/workflows/ci.yml` locally.
- [x] 2.3 Run `make build`, `make lint`, `make test-unit`, and `make test-e2e`.

## 3. Close The Change

- [x] 3.1 Record code review and final comparison evidence in `workspace/CI01/`.
- [x] 3.2 Sync `github-actions-ci` into main specs and archive this change.
- [x] 3.3 Mark `CI01` done only after workspace, OpenSpec, verification, and task board state agree.
