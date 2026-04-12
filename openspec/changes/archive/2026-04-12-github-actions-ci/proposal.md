## Why

The repository already contains the approved GitHub Actions CI workflow, but the CI work has no dedicated task board, workspace evidence, or durable OpenSpec capability contract. Closing that traceability gap keeps CI behavior, verification evidence, and task state aligned with `origin/main`.

## What Changes

- Add a durable CI capability spec for the approved GitHub Actions workflow shape.
- Record that the existing `.github/workflows/ci.yml` implementation satisfies the approved design.
- Preserve the current action versions as implementation details unless CI evidence shows breakage.
- Capture task-board and workspace evidence for CI verification.

## Capabilities

### New Capabilities

- `github-actions-ci`: GitHub Actions workflow behavior for build, lint, unit, and e2e checks.

### Modified Capabilities

- None.

## Impact

- `.github/workflows/ci.yml`
- `docs/plans/2026-04-10-github-actions-ci-design-task.md`
- `workspace/CI01/`
- `openspec/specs/github-actions-ci/spec.md`
