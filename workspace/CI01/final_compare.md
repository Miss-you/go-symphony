# CI01 Final Comparison

## Task Goal

`CI01` exists to close the approved GitHub Actions CI design with durable task-board, workspace, OpenSpec, and verification evidence.

## Source Design Comparison

The approved CI design requires:

- one workflow file at `.github/workflows/ci.yml`
- triggers on `push` and `pull_request` for `main`
- four independent jobs: `build`, `lint`, `unit`, `e2e`
- `make build` for build
- `golangci/golangci-lint-action` for lint
- `make test-unit` for unit tests
- `make test-e2e` for e2e tests

`origin/main` already satisfies this behavior. This branch does not change `.github/workflows/ci.yml`; it records the task/spec/workspace evidence that was missing.

## Original Symphony Comparison

Original Symphony uses an Elixir-centered `make-all` workflow that runs `make all` under `elixir/`. The Go implementation intentionally differs by splitting checks into independent jobs and using root Makefile targets. That difference matches the approved Go CI design and keeps failures easier to localize in GitHub checks.

## Accepted Action-Version Decision

The approved design mentions `actions/checkout@v5`; the live workflow uses `actions/checkout@v6` and `actions/setup-go@v6`.

This is accepted because the compatibility contract is behavioral:

- checkout runs
- Go is installed from `go.mod`
- the required build/lint/unit/e2e gates run

Action major versions remain implementation details unless CI evidence shows a real breakage.

## Verification Evidence

Fresh verification passed:

- YAML parse of `.github/workflows/ci.yml`
- `make build`
- `make lint`
- `make test-unit`
- `make test-e2e`
- `openspec validate --type change github-actions-ci`
- `openspec validate --specs`
- `git diff --check`
- final post-archive `make verify`

## Residual Risk

No blocking residual risk remains for CI01. The PR must be described as a traceability/spec-closeout change because the workflow itself already existed on `origin/main`.
