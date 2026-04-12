# CI01 Final Implementation v1

## Implementation Decision

Accept the current repository state as the implementation for `CI01` and do not add new workflow code unless review finds a concrete design mismatch.

Current repo evidence already matches the approved CI shape:

- `.github/workflows/ci.yml` exists.
- It triggers on `push` and `pull_request` to `main`.
- It defines four independent jobs: `build`, `lint`, `unit`, and `e2e`.
- The jobs use the intended commands or action:
  - `make build`
  - `golangci/golangci-lint-action`
  - `make test-unit`
  - `make test-e2e`

This is a traceability closure task. The workflow is already in place, so the remaining work is to align task metadata, workspace artifacts, OpenSpec artifacts, and fresh verification evidence.

## Symphony Parity

Original Symphony CI centers on a single `make all` workflow under `elixir/`, plus a separate PR-description lint workflow.

For go-symphony, the approved CI design intentionally chose a Go-native shape:

- one CI workflow file
- separate visible GitHub check jobs
- root `Makefile` commands
- no PR-description lint behavior in this task scope

That difference is acceptable because the CI parity target is behavior, not repository layout.

## Action Version Deviation Handling

The current workflow uses:

- `actions/checkout@v6`
- `actions/setup-go@v6`
- `golangci/golangci-lint-action@v9` with `version: v2.8.0`

The approved design names `actions/checkout@v5`, but does not require exact action-version parity as a user-visible behavior. Treat the version difference as an accepted implementation detail unless review or CI evidence shows concrete breakage.

Decision rule:

- Keep the existing versions if they satisfy the workflow contract.
- Change them only if a review or CI failure demonstrates a compatibility issue.
- Do not churn action versions just to make the implementation textually match the older design sentence.
- Record this decision in the OpenSpec change, `workspace/CI01/final_impl.md`, and the task board notes before closing `CI01`.

## OpenSpec And Test Strategy

This task should close through traceability and verification, not further feature code.

Evidence to capture:

- task board transition history for `CI01`
- `workspace/CI01/` artifacts
- OpenSpec change material
- fresh verification output for the required gates

Test strategy:

- confirm the workflow YAML parses
- run `make build`
- run `make lint`
- run `make test-unit`
- run `make test-e2e`

Purpose of the gates:

- YAML parse proves the workflow is structurally valid.
- `make build` proves the repository compiles through the canonical build target.
- `make lint` proves the lint contract is runnable locally and matches the workflow intent.
- `make test-unit` proves the unit-test target is runnable.
- `make test-e2e` proves the e2e target is wired and does not regress the CI contract.

If any gate fails due to missing local prerequisites, record that explicitly in `workspace/CI01/todo.md` and the task board instead of treating the task as fully verified.

## Closure Steps

1. Create and validate a narrow OpenSpec change for `github-actions-ci`.
2. Write `workspace/CI01/test_strategy.md`.
3. Verify the existing workflow against the approved CI design and implementation plan.
4. Run the required local gates.
5. Record code review and final comparison.
6. Archive the change after spec, workspace, and task board state agree.

## Final Check

No workflow code change is required unless review identifies a concrete mismatch between the approved design and the live repository state.
