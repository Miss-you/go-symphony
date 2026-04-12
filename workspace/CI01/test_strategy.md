# CI01 Test Strategy

## Goal

Prove that the existing GitHub Actions CI workflow is structurally valid and invokes the approved root verification commands.

## Verification Map

| Behavior | Verification | What It Proves |
| --- | --- | --- |
| Workflow YAML is parseable | Local YAML parse of `.github/workflows/ci.yml` | The workflow file is syntactically valid YAML and can be consumed by GitHub Actions. |
| Build job command is healthy | `make build` | The CI build command compiles the repository through the canonical build target. |
| Lint job contract is healthy | `make lint` | The local lint target invoked by project verification is runnable with the current toolchain. |
| Unit job command is healthy | `make test-unit` | The CI unit command executes the short Go test suite. |
| E2E job command is healthy | `make test-e2e` | The CI e2e command executes the tagged e2e test suite without requiring live provider credentials. |
| OpenSpec state is coherent | `openspec validate --type change github-actions-ci` and `openspec validate --specs` | The CI behavior is captured in durable spec artifacts before and after archive. |

## Scope Notes

- No workflow code change is expected unless verification or review finds a concrete mismatch.
- Local checks cannot fully emulate GitHub-hosted runner behavior, so they prove the workflow's command contract and YAML validity.
- The accepted action-version decision is part of the task evidence: `checkout@v6` and `setup-go@v6` remain acceptable unless CI evidence shows breakage.

## Required Closure Gates

Run before marking `CI01` done:

```bash
python3 -c 'import pathlib, yaml; yaml.safe_load(pathlib.Path(".github/workflows/ci.yml").read_text())'
make build
make lint
make test-unit
make test-e2e
openspec validate --type change github-actions-ci
openspec validate --specs
git diff --check
```
