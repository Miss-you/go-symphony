## ADDED Requirements

### Requirement: Main CI Workflow
The repository SHALL provide a GitHub Actions CI workflow that runs on pushes to `main` and pull requests targeting `main`.

#### Scenario: Main branch push
- **WHEN** a commit is pushed to `main`
- **THEN** GitHub Actions runs the CI workflow.

#### Scenario: Pull request to main
- **WHEN** a pull request targets `main`
- **THEN** GitHub Actions runs the CI workflow.

### Requirement: Independent CI Jobs
The CI workflow SHALL expose independent build, lint, unit, and e2e jobs.

#### Scenario: Build job
- **WHEN** the CI workflow runs
- **THEN** the build job checks out the repository, installs Go from `go.mod`, and runs `make build`.

#### Scenario: Lint job
- **WHEN** the CI workflow runs
- **THEN** the lint job checks out the repository, installs Go from `go.mod`, and runs the official `golangci/golangci-lint-action`.

#### Scenario: Unit job
- **WHEN** the CI workflow runs
- **THEN** the unit job checks out the repository, installs Go from `go.mod`, and runs `make test-unit`.

#### Scenario: E2E job
- **WHEN** the CI workflow runs
- **THEN** the e2e job checks out the repository, installs Go from `go.mod`, and runs `make test-e2e`.

### Requirement: CI Action Version Policy
The CI workflow SHALL treat third-party action versions as implementation details as long as they preserve the workflow contract.

#### Scenario: Action version differs from older design text
- **WHEN** the workflow uses newer checkout or setup-go action versions than an older design note names
- **THEN** the workflow remains compliant if checkout, Go setup from `go.mod`, and the required build/lint/unit/e2e gates still run.
