# Current Go CI State

The Go repository already contains `.github/workflows/ci.yml` on `origin/main`.

## Existing Workflow

- Workflow name: `CI`.
- Triggers: `push` to `main` and `pull_request` targeting `main`.
- Permissions: `contents: read`.
- Jobs: `build`, `lint`, `unit`, and `e2e`, each on `ubuntu-latest`.
- Each job checks out the repository and installs Go from `go.mod`.
- The build job runs `make build`.
- The lint job uses `golangci/golangci-lint-action@v9` with `version: v2.8.0`.
- The unit job runs `make test-unit`.
- The e2e job runs `make test-e2e`.

## Existing Makefile Gates

The root `Makefile` already provides:

- `make build`
- `make lint`
- `make test`
- `make test-unit`
- `make test-e2e`
- `make verify`

`make verify` runs build, lint, test, and e2e. The GitHub Actions workflow intentionally keeps build, lint, unit, and e2e as independent jobs so GitHub exposes separate check names.

## Design Match

The current workflow matches the approved CI design's required shape:

- one workflow file
- push and pull request triggers for `main`
- four independent jobs
- root Makefile commands for build/unit/e2e
- official golangci-lint action for linting

## Deviations To Accept Explicitly

- The design mentions `actions/checkout@v5`; current repo evidence uses `actions/checkout@v6`.
- The design did not pin `actions/setup-go`; current repo evidence uses `actions/setup-go@v6`.
- These are action-version differences, not behavior differences. The behavior-level contract remains intact: checkout, Go setup from `go.mod`, and the approved build/lint/unit/e2e gates all run as separate jobs.

## Work Remaining

The implementation is already present. CI01 should focus on durable traceability:

- add a CI task board
- produce workspace artifacts
- create and archive an internal OpenSpec change
- run the required verification gates
- record the accepted action-version deviation and final comparison
