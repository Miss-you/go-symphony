# AGENTS.md

## Scope

- Build `go-symphony` as a Go implementation of `openai/symphony`.
- Preserve user-visible compatibility before introducing broader abstractions.
- Keep this file durable and concise. Put task-specific detail in `docs/plans/` or `workspace/`.

## Source Of Truth

- Architecture and package boundaries: `docs/plans/2026-04-10-go-symphony-design.md`
- Task sequencing and verification gates: `docs/plans/2026-04-10-go-symphony-design-task.md`
- If chat history, task docs, and repo state disagree, trust repo evidence and update the task board before continuing.

## Architecture Rules

- Keep core packages provider-neutral: `internal/{config,domain,orchestrator,tracker,workspace,runner,codex}`.
- Keep provider-specific behavior in the compatibility shell: `internal/trackers/linear`, `internal/toolbridge/linear`, `internal/workflow`, and compatibility-facing API/UI packages.
- Use neutral core terms such as `WorkItem` and `provider`. Keep `issue`, `linear_graphql`, and other provider-specific terms in compatibility layers only.
- The orchestrator is the single owner of mutable runtime state. Workers report `RunEvent`s. `observability` is projection-only.
- Do not introduce a universal tracker write API, universal workpad, or provider-agnostic default workflow in V1.
- Do not add Lark-specific runtime behavior in V1.

## Repo Layout

- This repository is still pre-skeleton. Complete T02 before adding runtime code outside docs and task artifacts.
- Expected V1 layout: `cmd/symphony/`, `internal/{cli,codex,config,dashboard,domain,httpapi,observability,orchestrator,runner,tracker,trackers/{linear,memory},web,workflow,workspace}`.
- `docs/plans/` holds durable design and execution docs. `workspace/<task-id>/` holds task artifacts, not product code.

## Workflow

- Claim one task at a time in the task board before creating `workspace/<task-id>/`.
- Record every task-state transition in the task board `Change Log`. If blocked, write `resume_to=<state>` in `Notes`.
- Prefer small, package-scoped changes that match the task-board verification gates.
- Keep the root `AGENTS.md` short. Add nested `AGENTS.md` or `AGENTS.override.md` only when a subarea needs different rules.

## Build And Verification

- Canonical commands:
  - `make build`
  - `make lint`
  - `make test`
  - `make test-unit`
  - `make test-e2e`
  - `make verify`
- The current `Makefile` intentionally guards these targets until `go.mod` and the Go package skeleton exist.
- Use `go test ./...` as the broad gate, then tighten to package-scoped commands as packages land.
- Before closing a task, run the relevant `make` targets and keep verification evidence in the repo or task artifacts.
