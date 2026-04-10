# GitHub Actions CI Design

Date: 2026-04-10
Status: Approved

## Goal

Add a repository CI workflow that runs the minimum required engineering checks on GitHub:

- lint
- build
- unit tests
- e2e tests

## Chosen Shape

Use one workflow file, `.github/workflows/ci.yml`, with four independent jobs:

1. `build`
2. `lint`
3. `unit`
4. `e2e`

This keeps the checks visible and independently enforceable in GitHub while avoiding the overhead of a second dedicated e2e workflow before the repository has real end-to-end infrastructure.

## Execution Model

- Trigger on `push` and `pull_request` for `main`
- Use GitHub-hosted Ubuntu runners
- Use `actions/checkout@v5`
- Use `actions/setup-go` to install Go from the repo module file
- Reuse existing root commands where practical:
  - `make build`
  - `make test-unit`
  - `make test-e2e`
- Run lint in its own job with `golangci/golangci-lint-action`

## Why This Approach

- It preserves a dedicated `e2e` check name now, so later real end-to-end coverage can expand in place.
- It keeps build/test behavior aligned with the repository `Makefile`.
- It avoids a separate workflow file that would mostly duplicate setup steps at the current repo stage.

## Non-Goals

- No matrix builds yet
- No artifact upload yet
- No scheduled workflows yet
- No service containers or external dependency orchestration yet

## References

- GitHub Actions Go docs: https://docs.github.com/en/actions/tutorials/build-and-test-code/go
- `actions/setup-go`: https://github.com/actions/setup-go
- `golangci/golangci-lint-action`: https://github.com/golangci/golangci-lint-action
