# GitHub Actions CI Design Task Board

## Source Design

- Source: `docs/plans/2026-04-10-github-actions-ci-design.md`
- Design Date: 2026-04-10
- Design Status: Approved
- Task Board Scope: Translate the approved CI design into one execution-tracking task.

## Status Legend

- `todo`: not yet claimed; ready only when all hard dependencies are `done`
- `claimed`: claimed in the task board, but research has not started
- `research`: workspace exists and implementation approach is being derived
- `spec`: `final_impl.md`, change artifacts, and `test_strategy.md` are aligned
- `implementing`: approved behavior is being implemented under one change
- `verifying`: required build/test/lint/e2e checks are running or being fixed
- `review`: review findings are being triaged and closed
- `blocked`: temporarily interrupted; `Notes` must record `resume_to=<state>`
- `done`: task board, workspace artifacts, OpenSpec state, and repo evidence align

## Dependency Rules

- A task is claimable only when `Status=todo` and every task in `Depends On` is `done`.
- Hard dependencies are explicit in the table; there is no separate `ready` status.
- Parallel groups are advisory only and apply after dependencies are satisfied.
- One claimed task maps to one workspace directory and one OpenSpec change unless the table says otherwise.
- If repo evidence disagrees with task text, repo evidence wins and the table must be updated before more work continues.

## Task Table

| ID | Title | Goal | Depends On | Parallel | Status | Owner | Claimed At | Workspace | Change | Done When | Notes |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| CI01 | GitHub Actions CI Workflow | Add a single GitHub Actions workflow with independent build, lint, unit, and e2e jobs. | - | P0 | done | Codex | 2026-04-12 10:27 CST | workspace/CI01 | github-actions-ci | `.github/workflows/ci.yml` runs on push and pull requests to `main`; build, lint, unit, and e2e jobs reuse the approved commands or official lint action. | Isolated worktree: `.worktrees/github-actions-ci`; accepted action-version decision: keep current `checkout@v6` / `setup-go@v6` unless CI evidence shows breakage; change archived at `openspec/changes/archive/2026-04-12-github-actions-ci/`; final gate: `make verify`, `openspec validate --specs`, `git diff --check`. |
| CI02 | Claude Code GitHub Workflows | Wire up Claude Code as a GitHub App-backed assistant (on-demand issue/PR assist and automated PR review) via two dedicated workflows; intentionally kept separate from the `ci.yml` correctness gate. | CI01 | — | done | Yousa | 2026-05-10 CST (reverse-logged) | _(no workspace dir)_ | _(no OpenSpec change; operator-tooling scope)_ | `.github/workflows/claude.yml` and `.github/workflows/claude-code-review.yml` exist on `main` and are triggered by the documented events (issue/PR comment for `claude.yml`; PR open/sync for `claude-code-review.yml`); they do not block the `ci.yml` build/lint/test/e2e gates. | Landed as PR #26 (commit `41a8a96`), reverse-logged on 2026-05-10 per AGENTS.md ("repo evidence wins"). Scope is operator-assist tooling, not product correctness, so no OpenSpec spec sync is required; however the task-board entry is mandatory so future CI task authors see it as a committed surface. **Process debt**: no workspace dir, no final_compare, no code review artifact; review-of-review for `claude-code-review.yml` outputs is handled inside GitHub Actions run logs, not via `workspace/CI02/`. Follow-up: if either workflow ever becomes a merge blocker, it must be reclassified under CI01's correctness scope. |

## Claiming Rules

- Claim only one task at a time.
- Update this file before creating `workspace/<task-id>/`.
- Record `Owner`, `Claimed At`, and `Workspace` when claiming.
- Append every status transition to `Change Log`; do not rely on chat history.
- If a task is blocked, write `resume_to=<state>` in `Notes` before leaving it.

## Change Log

- 2026-04-12 10:27 CST: Initialized task board from approved CI design. No execution evidence existed yet, so `CI01` starts at `todo`.
- 2026-04-12 10:27 CST: Claimed `CI01` for end-to-end delivery in isolated worktree `.worktrees/github-actions-ci` after a passing baseline (`go test ./...`). Owner=`Codex`, workspace=`workspace/CI01`.
- 2026-04-12 10:27 CST: Created `workspace/CI01/` in the isolated worktree and moved `CI01` to `research`.
- 2026-04-12 10:27 CST: Accepted `workspace/CI01/final_impl.md` after round-two rubric review (scores 95/97, avg 96/100, no high-severity issues) and moved `CI01` to `spec`.
- 2026-04-12 10:27 CST: Created and validated OpenSpec change `github-actions-ci`, wrote `workspace/CI01/test_strategy.md`, cleared spec review with no findings, and moved `CI01` to `implementing`.
- 2026-04-12 10:27 CST: Applied the traceability tasks for `github-actions-ci`, parsed `.github/workflows/ci.yml` locally, and moved `CI01` to `verifying`.
- 2026-04-12 10:27 CST: Fresh verification passed (YAML parse, `make build`, `make lint`, `make test-unit`, `make test-e2e`, `openspec validate --type change github-actions-ci`, `openspec validate --specs`, `git diff --check`) and moved `CI01` to `review`.
- 2026-04-12 10:27 CST: Review found closeout-only issues; recorded `workspace/CI01/code_review.md`, `workspace/CI01/final_compare.md`, and residual notes.
- 2026-04-12 10:27 CST: Synced `github-actions-ci` into main specs, archived the change to `openspec/changes/archive/2026-04-12-github-actions-ci/`, final validation passed (`make verify`, `openspec validate --specs`, `git diff --check`), and marked `CI01` done.
- 2026-05-10 CST: Reverse-logged `CI02` (`Claude Code GitHub Workflows`) as `done`. Source commit `41a8a96` (PR #26) added `.github/workflows/claude.yml` and `.github/workflows/claude-code-review.yml` on top of `CI01`'s correctness gate. Scope is explicitly operator-assist tooling (Claude GitHub App responding to issue/PR comments and opening review comments on PRs), not product correctness; both workflows are intentionally non-blocking for merges. Process debt recorded in CI02 Notes: no workspace dir, no final_compare, no code review artifact. Reclassification rule: if either workflow ever becomes a merge blocker it must move back under CI01's scope.
