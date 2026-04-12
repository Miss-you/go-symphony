# Original Symphony CI

Original Symphony keeps CI centered on the Elixir app under `elixir/`, not the repository root.

## Workflow Shape

- `.github/workflows/make-all.yml` is the main CI workflow.
- It triggers on `pull_request` and on `push` to `main`.
- It runs a single `ubuntu-latest` job with `working-directory: elixir`.
- The job checks out the repository, installs tools through `jdx/mise-action@v3`, caches `elixir/deps` and `elixir/_build`, and runs `make all`.
- `.github/workflows/pr-description-lint.yml` is a separate PR-only workflow for PR body validation.

## Makefile Contract

`elixir/Makefile` exposes the CI surface used by the workflow:

- `setup` runs `mix setup`
- `build` runs `mix build`
- `fmt-check` runs `mix format --check-formatted`
- `lint` runs `mix lint`
- `coverage` runs `mix test --cover`
- `dialyzer` runs `mix deps.get` and `mix dialyzer --format short`
- `ci` runs setup, build, format check, lint, coverage, and dialyzer
- `all` delegates to `ci`

The Elixir `e2e` target is intentionally manual/live: it requires `LINEAR_API_KEY` and a `codex` binary, then runs the live e2e test directly. It is not part of the original GitHub Actions main CI workflow.

## Parity Implications

- Keep GitHub Actions aligned with Makefile commands instead of open-coding checks in workflow YAML.
- Preserve PR and mainline triggering for the core quality gate.
- Treat live-provider e2e as distinct from repository command-contract e2e.
- Preserve behavior-level parity rather than exact third-party action versions.
