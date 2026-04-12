## Context

`origin/main` already includes `.github/workflows/ci.yml`. The workflow matches the approved CI design by running on push and pull requests to `main` and by exposing four independent jobs: build, lint, unit, and e2e.

Original Symphony used an Elixir-centered workflow that ran `make all` under `elixir/`. The Go repository intentionally uses root Makefile targets so CI stays aligned with the Go module and the repository's canonical verification commands.

## Goals / Non-Goals

**Goals:**

- Keep one CI workflow with separate build, lint, unit, and e2e jobs.
- Reuse root Makefile targets for build and test jobs.
- Use the official golangci-lint action for lint annotations and setup.
- Record the existing action-version choices as accepted implementation details.
- Add durable traceability for a workflow that already exists in the repository.

**Non-Goals:**

- No matrix builds.
- No artifact upload for CI.
- No scheduled CI workflow.
- No service containers or external dependency orchestration.
- No PR-description lint behavior in this task.

## Decisions

1. Accept the existing workflow implementation.
   - The workflow already satisfies the approved behavior shape.
   - Rewriting it would add churn without improving compatibility.

2. Keep four independent jobs instead of one bundled `make verify` job.
   - GitHub exposes separate check names for build, lint, unit, and e2e.
   - This matches the approved design and gives clearer failure localization.

3. Keep current third-party action versions.
   - The approved design names `actions/checkout@v5`, while the live workflow uses `actions/checkout@v6` and `actions/setup-go@v6`.
   - This is accepted as an implementation detail because the user-visible contract is checkout, Go setup from `go.mod`, and the required Makefile gates.
   - Change action versions only if CI or review evidence shows concrete breakage.

## Risks / Trade-offs

- Version drift from the older design text could confuse future readers. Mitigation: record the accepted decision in the task board, final implementation notes, and spec.
- Local verification cannot execute the GitHub-hosted runner environment. Mitigation: parse the workflow YAML locally and run the same root commands that the workflow invokes.
- The e2e job currently proves the repository e2e target, not live Linear/Codex provider behavior. Mitigation: keep live-provider e2e outside this CI task unless a future design expands the CI contract.
