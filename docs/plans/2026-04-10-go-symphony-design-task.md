# Go Symphony Design Task Board

## Source Design

- Source: `docs/plans/2026-04-10-go-symphony-design.md`
- Design Date: 2026-04-10
- Design Status: Approved design draft
- Task Board Scope: Translate the approved design into execution-tracking truth for one-task-at-a-time delivery.

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
| T01 | Compatibility Contract | Freeze parity scope, terminology mapping, and explicit non-goals as execution inputs. | - | P0 | done | Codex | 2026-04-10 17:40 CST | workspace/T01 | freeze-compatibility-contract | Parity checklist, terminology mapping, and scope boundaries are captured in durable repo artifacts and validated against the design. | Gate: `openspec validate freeze-compatibility-contract`; N/A gates recorded in `workspace/T01/todo.md` |
| T02 | Repo Skeleton | Create the initial Go repo/package layout without leaking provider-specific behavior into core packages. | T01 | P1 | done | Codex | 2026-04-10 18:01 CST | workspace/T02 | repo-skeleton-bootstrap | `cmd/` and `internal/` layout exist and an empty build can run. | Gate: `go test ./...`; verified with `go test ./...` and `make verify`; internal bootstrap task under the approved design, no durable capability-spec sync required. |
| T03 | WORKFLOW Loader | Implement `WORKFLOW.md` parsing, template loading, hot reload, and last-known-good semantics. | T02 | P2 | done | Codex | 2026-04-10 18:57 CST | workspace/T03 | workflow-loader | Loader behavior matches current semantics, including invalid reload fallback. | Gate: `go test ./internal/config/...`; broader verification: `go test ./...`, `make build`, `make lint`; `make test-e2e` applicability recorded in `workspace/T03/todo.md`. |
| T04 | Internal Config Model | Normalize existing external config into a provider-neutral internal config model. | T03 | P2 | done | Codex | 2026-04-10 21:26 CST | workspace/T04 | internal-config-model | Old config input still works and the core no longer depends on Linear-only concepts. | Gate: `go test ./internal/config/...`; broader verification: `go test ./...`, `make build`, `make lint`, `make test-e2e`; residual notes recorded in `workspace/T04/todo.md`. |
| T05 | Domain Model | Define the provider-neutral runtime model for orchestration and snapshots. | T04 | P3 | done | Codex | 2026-04-10 23:38 CST | workspace/T05 | domain-model | Core domain types avoid Linear naming and provider-specific write semantics. | Gate: `go test ./internal/domain/...`; broader verification: `go test ./...`, `make build`, `make lint`, `make test-e2e`; residual notes recorded in `workspace/T05/todo.md`. |
| T06 | Orchestrator Core | Build the single-owner runtime loop for polling, scheduling, retry, reconcile, and snapshots. | T05 | P4 | done | Codex | 2026-04-11 00:14 CST | workspace/T06 | orchestrator-core | Orchestrator is the sole mutable state owner and workers emit `RunEvent`s only. | Isolated worktree: `.worktrees/t06-orchestrator-core`; Gate: `go test ./internal/orchestrator/...`; e2e applicability note recorded in `workspace/T06/todo.md`; review findings recorded in `workspace/T06/code_review.md`; change archived at `openspec/changes/archive/2026-04-11-orchestrator-core` |
| T07 | Workspace Lifecycle | Implement workspace naming, hooks, cleanup, and terminal-state cleanup semantics. | T06 | P5 | done | Codex | 2026-04-11 01:16 CST | workspace/T07 | workspace-lifecycle | Hook and cleanup behavior matches design expectations across success, failure, and timeout. | Isolated worktree: `.worktrees/t07-workspace-lifecycle`; Gate: `go test ./internal/workspace/...`; e2e/runtime applicability notes recorded in `workspace/T07/todo.md`; review findings recorded in `workspace/T07/code_review.md` and `workspace/T07/code_review_followup.md`; change archived at `openspec/changes/archive/2026-04-11-workspace-lifecycle` |
| T08 | Runner / ExecutionHost | Separate local and SSH execution concerns from workspace logic behind one runtime contract. | T07 | P6 | done | Codex | 2026-04-11 09:15 CST | workspace/T08 | runner-execution-host | Local and SSH runs share one contract and SSH details stay outside `workspace`. | Isolated worktree: `.worktrees/t08-runner-executionhost`; Gate: `go test ./internal/runner/...`; broader verification: `go test ./...`, `make build`, `make lint`, `make test-e2e`; review findings recorded in `workspace/T08/code_review.md`; change archived at `openspec/changes/archive/2026-04-11-runner-execution-host` |
| T09 | Codex App-Server Protocol | Implement session lifecycle, protocol events, tool calls, approvals, and timeout handling. | T08 | P7 | done | Codex | 2026-04-11 11:30 CST | workspace/T09 | codex-app-server-protocol | App-server behavior is handled as a protocol compatibility target rather than thin stdio wrapping. | Isolated worktree: `.worktrees/t09-codex-app-server-protocol`; Gate: `go test ./internal/codex/...`; broader verification: `go test -count=1 ./...`, `make build`, `make lint`, `make test-e2e`; review findings recorded in `workspace/T09/code_review.md`; change archived at `openspec/changes/archive/2026-04-11-codex-app-server-protocol` |
| T10 | TrackerReader + Memory Adapter | Freeze the core read-only tracker interface and provide a memory adapter for local/test closure. | T05, T06 | P4 | done | Codex | 2026-04-11 01:16 CST | workspace/T10 | tracker-reader-memory-adapter | Core tracker surface is read-only and memory-backed tests can drive the runtime loop. | Isolated worktree: `.worktrees/t10-tracker-memory-adapter`; Gate: `go test ./internal/tracker/... ./internal/trackers/memory/...`; review findings recorded in `workspace/T10/code_review.md`; residual notes recorded in `workspace/T10/todo.md`; change archived at `openspec/changes/archive/2026-04-11-tracker-reader-memory-adapter` |
| T11 | Linear Reader Adapter | Recreate current Linear read behavior, normalization, routing, and error handling. | T10 | P8 | done | Codex | 2026-04-11 09:15 CST | workspace/T11 | linear-reader-adapter | Linear fetch/refresh/pagination/assignee/error behavior matches the current product contract. | Isolated worktree: `.worktrees/t11-linear-reader-adapter`; Gate: `go test ./internal/trackers/linear/...`; review findings recorded in `workspace/T11/code_review.md` and `workspace/T11/code_review_followup.md`; change archived at `openspec/changes/archive/2026-04-11-linear-reader-adapter` |
| T12 | Linear ToolBridge | Implement `linear_graphql` and other Linear-specific write behavior in the compatibility shell only. | T09, T11 | P9 | done | Codex | 2026-04-11 13:13 CST | workspace/T12 | linear-toolbridge | Tracker writes stay out of core interfaces and tool injection is compatibility-shell owned. | Isolated worktree: `.worktrees/t12-linear-toolbridge`; Gate: `go test ./internal/codex/... ./internal/toolbridge/...`; broader verification: `go test ./...`, `make build`, `make lint`, `make test`, `make test-e2e`, `make verify`, `openspec validate --type change linear-toolbridge`; review findings recorded in `workspace/T12/code_review.md`; residual notes recorded in `workspace/T12/todo.md`; change archived at `openspec/changes/archive/2026-04-11-linear-toolbridge` |
| T13 | Linear Workflow Bundle | Implement workflow selection and the first concrete `compat_linear_default` workflow bundle. | T12 | P9 | done | Codex | 2026-04-11 16:21 CST | workspace/T13 | linear-workflow-bundle | The initial workflow is explicitly Linear-specific and later expansion remains possible. | Isolated worktree: `.worktrees/t13-linear-workflow-bundle`; Gate: `go test ./internal/workflow/...`; broader verification: `go test -count=1 ./...`, `make build`, `make lint`, `make verify`, `openspec validate --specs`; review findings recorded in `workspace/T13/code_review.md`; residual notes recorded in `workspace/T13/todo.md`; change archived at `openspec/changes/archive/2026-04-11-linear-workflow-bundle`. |
| T14 | End-to-End Run Integration | Connect runtime, workspace, runner, codex, tracker, toolbridge, and workflow into complete runs. | T06, T07, T08, T09, T11, T12, T13 | P10 | done | Codex | 2026-04-11 17:01 CST | workspace/T14 | end-to-end-run-integration | Memory and Linear paths both run end to end with retry, cleanup, event normalization, and shutdown behavior covered. | Isolated worktree: `.worktrees/t14-end-to-end-run-integration`; review findings recorded in `workspace/T14/code_review.md`; final comparison in `workspace/T14/final_compare.md`; residual notes in `workspace/T14/todo.md`; change archived at `openspec/changes/archive/2026-04-11-end-to-end-run-integration/`; final validation passed (`make verify`, `openspec validate --specs`, `git diff --check`). |
| T15 | HTTP API Compatibility | Recreate `/api/v1/state`, `/api/v1/refresh`, and `/api/v1/:issue_identifier` semantics. | T14 | P11 | done | Codex | 2026-04-12 00:12 CST | workspace/T15 | http-api-compatibility | API DTOs, refresh semantics, and error codes are compatibility-checked. | Isolated worktree: `.worktrees/t15-httpapi-compatibility`; Gate: `go test ./internal/httpapi/...`; broader verification passed (`go test -count=1 ./...`, `make build`, `make lint`, `make test-e2e`, `make verify`, `openspec validate --type change http-api-compatibility`, `openspec validate --specs`, `git diff --check`); change archived at `openspec/changes/archive/2026-04-12-http-api-compatibility/` |
| T16 | Terminal Dashboard Compatibility | Recreate terminal dashboard output, event humanization, and fixture coverage. | T14, T15 | P12 | done | Codex | 2026-04-12 00:57 CST | workspace/T16 | terminal-dashboard-compatibility | Snapshot fixtures validate output and dashboard behavior is treated as a compatibility surface. | Isolated worktree: `.worktrees/t16-terminal-dashboard-compatibility`; review findings recorded in `workspace/T16/code_review.md` and `workspace/T16/code_review_followup.md`; final comparison in `workspace/T16/final_compare.md`; residual notes in `workspace/T16/todo.md`; change archived at `openspec/changes/archive/2026-04-12-terminal-dashboard-compatibility/`; final validation passed (`make verify`, `openspec validate --specs`, `git diff --check`). |
| T17 | Web Dashboard + Static Assets | Recreate `/` dashboard behavior and required static assets. | T15 | P12 | done | Codex | 2026-04-12 01:42 CST | workspace/T17 | web-dashboard-static-assets | Web observability is present at `/` and static assets are served. | Isolated worktree: `.worktrees/t17-web-dashboard-static-assets`; review findings recorded in `workspace/T17/code_review.md`; final comparison in `workspace/T17/final_compare.md`; residual notes in `workspace/T17/todo.md`; change archived at `openspec/changes/archive/2026-04-12-web-dashboard-static-assets/`; final validation passed (`make verify`, `openspec validate --specs`, `git diff --check`). |
| T18 | CLI + Full Parity Test Sweep | Recreate CLI behavior, startup acknowledgement, shutdown rendering, and full parity-oriented verification. | T16, T17 | P13 | todo | - | - | - | - | CLI behavior matches the design and the full parity test matrix passes or is explicitly documented. | Gate: `go test ./...` |

## Claiming Rules

- Claim only one task at a time.
- Update this file before creating `workspace/<task-id>/`.
- Record `Owner`, `Claimed At`, and `Workspace` when claiming.
- Append every status transition to `Change Log`; do not rely on chat history.
- If a task is blocked, write `resume_to=<state>` in `Notes` before leaving it.

## Change Log

- 2026-04-10: Initialized task board from approved design. No execution evidence existed yet, so all tasks start at `todo`.
- 2026-04-10 17:40 CST: Claimed `T01` for end-to-end delivery. Owner=`Codex`.
- 2026-04-10 17:40 CST: Created `workspace/T01/` and moved `T01` to `research`.
- 2026-04-10 17:49 CST: Accepted `workspace/T01/final_impl.md` after rubric review (avg 91/100, no high-severity issues) and moved `T01` to `spec`.
- 2026-04-10 17:50 CST: Created OpenSpec change `freeze-compatibility-contract` and linked it to `T01`.
- 2026-04-10 17:53 CST: Spec review passed with no high-severity issues; moved `T01` to `implementing` for documentation-only contract landing.
- 2026-04-10 17:54 CST: Recorded non-applicable verification gates in `workspace/T01/todo.md` and moved `T01` to `verifying`.
- 2026-04-10 17:54 CST: Fresh verification passed (`openspec validate`, content check, scope guard) and moved `T01` to `review`.
- 2026-04-10 17:58 CST: Synced `compatibility-contract` into main specs, archived `freeze-compatibility-contract` to `openspec/changes/archive/2026-04-10-freeze-compatibility-contract/`, and marked `T01` done.
- 2026-04-10 18:01 CST: Claimed `T02` for repo bootstrap, set `workspace/T02`, and reserved internal change name `repo-skeleton-bootstrap`.
- 2026-04-10 18:02 CST: Wrote the `T02` implementation plan and workspace verification artifacts, then moved `T02` to `spec`.
- 2026-04-10 18:03 CST: Initialized `go.mod`, wrote a failing `cmd/symphony/main_test.go`, observed the expected red failure (`undefined: main`), and moved `T02` to `implementing`.
- 2026-04-10 18:05 CST: Added `cmd/symphony/main.go` and the approved flat `internal/...` placeholder package layout, then moved `T02` to `verifying`.
- 2026-04-10 18:06 CST: Verification passed (`go test ./...`, `make verify`, `git diff --check`), reviewed the repo layout against the approved design, and marked `T02` done.
- 2026-04-10 18:57 CST: Claimed `T03` for end-to-end delivery. Owner=`Codex`.
- 2026-04-10 18:57 CST: Created `workspace/T03/` and moved `T03` to `research`.
- 2026-04-10 19:17 CST: Accepted `workspace/T03/final_impl.md` after round-two rubric review (avg 95/100, no high-severity issues), linked OpenSpec change `workflow-loader`, and kept `T03` in `spec`.
- 2026-04-10 19:20 CST: Spec review passed with no remaining high-severity issues; moved `T03` to `implementing`.
- 2026-04-10 19:25 CST: Implemented the `internal/config` workflow loader/store with passing package-level TDD gate and moved `T03` to `verifying`.
- 2026-04-10 19:26 CST: Fresh verification passed (`go test ./internal/config/...`, `go test ./...`, `make build`, `make lint`, `make test-e2e`), recorded the current e2e applicability note in `workspace/T03/todo.md`, and moved `T03` to `review`.
- 2026-04-10 19:29 CST: Review found no blocking implementation issues, synced `workflow-loader` into `openspec/specs/workflow-loader/spec.md`, archived the change to `openspec/changes/archive/2026-04-10-workflow-loader/`, and marked `T03` done.
- 2026-04-10 21:26 CST: Claimed `T04` for end-to-end delivery. Owner=`Codex`, workspace=`workspace/T04`.
- 2026-04-10 21:26 CST: Created `workspace/T04/` in the isolated worktree and moved `T04` to `research`.
- 2026-04-10 21:38 CST: Accepted `workspace/T04/final_impl.md` after round-two rubric review (avg 91.5/100, no high-severity issues) and moved `T04` to `spec`.
- 2026-04-10 21:42 CST: Created OpenSpec change `internal-config-model` and generated proposal/design/specs/tasks to apply-ready.
- 2026-04-10 21:45 CST: Spec review passed with no remaining high-severity issues; moved `T04` to `implementing`.
- 2026-04-10 21:50 CST: Implemented the typed settings layer, validation, and atomic raw-plus-typed store snapshot with a passing package-level TDD gate, then moved `T04` to `verifying`.
- 2026-04-10 21:53 CST: Fresh verification passed (`go test ./internal/config/...`, `go test ./...`, `make build`, `make lint`, `make test-e2e`) and recorded the current e2e applicability note in `workspace/T04/todo.md`.
- 2026-04-10 21:53 CST: Entered `review` for `T04` after verification closed with fresh evidence.
- 2026-04-10 21:58 CST: Review found no blocking implementation issues, synced `runtime-config` into `openspec/specs/runtime-config/spec.md`, archived the change to `openspec/changes/archive/2026-04-10-internal-config-model/`, and marked `T04` done.
- 2026-04-10 23:38 CST: Claimed `T05` for end-to-end delivery in isolated worktree `.worktrees/t05-domain-model`. Owner=`Codex`.
- 2026-04-10 23:38 CST: Created `workspace/T05/` in the isolated worktree and moved `T05` to `research`.
- 2026-04-10 23:48 CST: Created OpenSpec change `domain-model`, generated proposal/design/specs/tasks, and wrote `workspace/T05/test_strategy.md`.
- 2026-04-10 23:49 CST: Accepted `workspace/T05/final_impl.md` after rubric review (avg 85.5/100, no high-severity issues) and moved `T05` to `spec`.
- 2026-04-10 23:49 CST: Spec review passed with no high-severity issues; moved `T05` to `implementing`.
- 2026-04-10 23:50 CST: Implemented the provider-neutral domain types and passing package-scoped contract tests, then moved `T05` to `verifying`.
- 2026-04-10 23:53 CST: Fresh verification passed (`go test ./internal/domain/...`, `go test ./...`, `make build`, `make lint`, `make test-e2e`) and recorded the current e2e applicability note in `workspace/T05/todo.md`.
- 2026-04-10 23:53 CST: Entered `review` for `T05` after verification closed with fresh evidence.
- 2026-04-10 23:54 CST: Review found no blocking implementation issues, synced `runtime-domain-model` into `openspec/specs/runtime-domain-model/spec.md`, archived the change to `openspec/changes/archive/2026-04-10-domain-model/`, and marked `T05` done.
- 2026-04-11 00:14 CST: Claimed `T06` for end-to-end delivery in isolated worktree `.worktrees/t06-orchestrator-core`. Owner=`Codex`, workspace=`workspace/T06`.
- 2026-04-11 00:14 CST: Created `workspace/T06/` in the isolated worktree and moved `T06` to `research`.
- 2026-04-11 00:35 CST: Revised `workspace/T06/final_impl.md` after a blocking review round and re-accepted it after round-two rubric review (reviews 98/93, avg 95.5/100, no high-severity issues); `T06` remains in `spec`.
- 2026-04-11 00:37 CST: Linked OpenSpec change `orchestrator-core` to `T06` and continued spec artifact creation.
- 2026-04-11 00:43 CST: Created OpenSpec change artifacts for `orchestrator-core`, wrote `workspace/T06/test_strategy.md`, and cleared spec review with no high-severity issues.
- 2026-04-11 00:43 CST: Moved `T06` to `implementing` to start TDD-based orchestrator core implementation.
- 2026-04-11 00:53 CST: Fresh verification passed (`go test ./internal/orchestrator/...`, `go test ./...`, `make build`, `make lint`, `make test-e2e`) and recorded the current e2e applicability note in `workspace/T06/todo.md`.
- 2026-04-11 01:00 CST: Review surfaced two blocking correctness issues in `T06` (retry entries were never scheduled for delivery, and candidate dispatch double-admitted host capacity). Both were fixed with new package-scoped tests before closure.
- 2026-04-11 01:00 CST: Fresh verification passed again after the review fixes (`go test ./internal/orchestrator/...`, `go test ./...`, `make build`, `make lint`, `make test-e2e`), and `T06` entered `review`.
- 2026-04-11 01:03 CST: Synced `orchestrator-core` delta specs into `openspec/specs/runtime-orchestrator/spec.md` and `openspec/specs/runtime-domain-model/spec.md`.
- 2026-04-11 01:03 CST: Archived `orchestrator-core` to `openspec/changes/archive/2026-04-11-orchestrator-core/` and marked `T06` done.
- 2026-04-11 01:16 CST: Claimed `T07` for end-to-end delivery in isolated worktree `.worktrees/t07-workspace-lifecycle`. Owner=`Codex`, workspace=`workspace/T07`.
- 2026-04-11 01:16 CST: Created `workspace/T07/` in the isolated worktree and moved `T07` to `research`.
- 2026-04-11 01:16 CST: Claimed `T10` for end-to-end delivery in isolated worktree `.worktrees/t10-tracker-memory-adapter`. Owner=`Codex`, workspace=`workspace/T10`.
- 2026-04-11 01:16 CST: Created `workspace/T10/` in the isolated worktree and moved `T10` to `research`.
- 2026-04-11 01:27 CST: Accepted `workspace/T07/final_impl.md` after round-two rubric review (avg 90.5/100, no high-severity issues) and moved `T07` to `spec`.
- 2026-04-11 01:31 CST: Accepted `workspace/T10/final_impl.md` after round-two rubric review (reviews 95/90, avg 92.5/100, no high-severity issues) and moved `T10` to `spec`.
- 2026-04-11 01:31 CST: Created apply-ready OpenSpec change `workspace-lifecycle` for `T07` and wrote `workspace/T07/test_strategy.md`.
- 2026-04-11 01:32 CST: Created OpenSpec change `tracker-reader-memory-adapter`, generated apply-ready artifacts, and wrote `workspace/T10/test_strategy.md`.
- 2026-04-11 01:32 CST: Spec review for `workspace-lifecycle` passed with no high-severity issues and `T07` moved to `implementing`.
- 2026-04-11 01:33 CST: Spec review passed with no high-severity issues; moved `T10` to `implementing`.
- 2026-04-11 01:36 CST: Implemented `internal/tracker.TrackerReader` and the deterministic memory reader with passing package-scoped TDD gate (`go test ./internal/tracker/... ./internal/trackers/memory/...`), then moved `T10` to `verifying`.
- 2026-04-11 01:37 CST: Fresh broader verification passed (`go test ./...`, `make build`, `make lint`, `make test-e2e`), and `T10` entered `review`.
- 2026-04-11 01:38 CST: Synced `runtime-tracker-reader` delta specs into `openspec/specs/runtime-tracker-reader/spec.md`.
- 2026-04-11 01:38 CST: Archived `tracker-reader-memory-adapter` to `openspec/changes/archive/2026-04-11-tracker-reader-memory-adapter/` and marked `T10` done.
- 2026-04-11 01:39 CST: Fresh final verification passed after sync/archive (`go test ./...`, `make build`, `make lint`, `make test-e2e`), confirming `T10` closure evidence.
- 2026-04-11 01:40 CST: Implemented the `internal/workspace` lifecycle package with passing package-scoped contract tests and moved `T07` to `verifying`.
- 2026-04-11 01:41 CST: Fresh verification passed (`go test ./internal/workspace/...`, `go test ./...`, `make build`, `make lint`, `make test-e2e`) and recorded the current e2e/runtime applicability notes in `workspace/T07/todo.md`.
- 2026-04-11 01:41 CST: Entered `review` for `T07` after verification closed with fresh evidence.
- 2026-04-11 01:47 CST: Review surfaced one blocking correctness issue (blank worker-host cleanup recursion) and one medium logging gap for best-effort hooks; both were fixed with added package-scoped regression tests.
- 2026-04-11 01:47 CST: Fresh verification passed again after the review fixes (`go test ./internal/workspace/...`, `go test ./...`, `make build`, `make lint`, `make test-e2e`).
- 2026-04-11 01:48 CST: Synced `workspace-lifecycle` into `openspec/specs/workspace-lifecycle/spec.md`.
- 2026-04-11 01:48 CST: Archived `workspace-lifecycle` to `openspec/changes/archive/2026-04-11-workspace-lifecycle/` and marked `T07` done.
- 2026-04-11 09:15 CST: Claimed `T08` for end-to-end delivery in isolated worktree `.worktrees/t08-runner-executionhost`. Owner=`Codex`, workspace=`workspace/T08`.
- 2026-04-11 09:15 CST: Created `workspace/T08/` in the isolated worktree after a passing baseline (`go test ./...`) and moved `T08` to `research`.
- 2026-04-11 09:15 CST: Accepted `workspace/T08/final_impl.md` after round-two rubric review (reviews 92/88, avg 90/100, no high-severity issues) and moved `T08` to `spec`.
- 2026-04-11 09:15 CST: Created and validated OpenSpec change `runner-execution-host` for `T08` and linked it in the task board.
- 2026-04-11 09:15 CST: Wrote `workspace/T08/test_strategy.md`, cleared spec review with no high-severity issues, and moved `T08` to `implementing`.
- 2026-04-11 09:15 CST: Implemented runner contract, workspace executor delegation, and runner-backed orchestrator host selection with package-scoped TDD gates, then moved `T08` to `verifying`.
- 2026-04-11 09:15 CST: Fresh verification passed (`go test ./internal/runner/...`, `go test ./internal/workspace/... ./internal/orchestrator/...`, `go test ./...`, `make build`, `make lint`, `make test-e2e`).
- 2026-04-11 09:15 CST: Entered `review` for `T08` after verification closed with fresh evidence.
- 2026-04-11 09:15 CST: Code review found three issues (user-prefixed IPv6 SSH host parsing, remote workspace fail-fast behavior, and default service admission wiring); all were fixed with targeted tests.
- 2026-04-11 09:15 CST: Fresh verification passed again after review fixes (`go test ./internal/runner/... ./internal/workspace/... ./internal/orchestrator/...`, `go test ./...`, `make build`, `make lint`, `make test-e2e`).
- 2026-04-11 09:15 CST: Claimed `T11` for end-to-end delivery in isolated worktree `.worktrees/t11-linear-reader-adapter`. Owner=`Codex`, workspace=`workspace/T11`.
- 2026-04-11 09:15 CST: Created `workspace/T11/` in the isolated worktree and moved `T11` to `research`.
- 2026-04-11 09:22 CST: Accepted `workspace/T11/final_impl.md` after round-two rubric review (reviews 92/89, avg 90.5/100, no high-severity issues) and moved `T11` to `spec`.
- 2026-04-11 09:27 CST: Created apply-ready OpenSpec change `linear-reader-adapter` and wrote `workspace/T11/test_strategy.md`.
- 2026-04-11 09:28 CST: Spec review passed with no high-severity issues; moved `T11` to `implementing`.
- 2026-04-11 09:34 CST: Implemented the read-only Linear reader adapter with passing package-scoped TDD gate (`go test ./internal/trackers/linear/...`) and moved `T11` to `verifying`.
- 2026-04-11 09:37 CST: Fresh verification passed (`go test -count=1 ./internal/trackers/linear/...`, `go test -count=1 ./...`, `make build`, `make lint`, `make test-e2e`, `openspec validate linear-reader-adapter`) and moved `T11` to `review`.
- 2026-04-11 09:44 CST: Review fixes were verified, final comparison recorded in `workspace/T11/final_compare.md`, synced `linear-reader-adapter` into main specs, archived the change to `openspec/changes/archive/2026-04-11-linear-reader-adapter/`, and marked `T11` done after fresh final gates.
- 2026-04-11 09:47 CST: Synced `runtime-runner-execution-host`, `workspace-lifecycle`, and `runtime-orchestrator` specs for `T08`, archived `runner-execution-host` to `openspec/changes/archive/2026-04-11-runner-execution-host/`, and marked `T08` done after `openspec validate --specs` passed.
- 2026-04-11 11:30 CST: Claimed `T09` for end-to-end delivery in isolated worktree `.worktrees/t09-codex-app-server-protocol`. Owner=`Codex`, workspace=`workspace/T09`.
- 2026-04-11 11:30 CST: Created `workspace/T09/` in the isolated worktree after a passing baseline (`go test ./...`) and moved `T09` to `research`.
- 2026-04-11 11:38 CST: Accepted `workspace/T09/final_impl.md` after rubric review (reviews 88/94, avg 91/100, no high-severity issues) and moved `T09` to `spec`.
- 2026-04-11 11:42 CST: Created and validated OpenSpec change `codex-app-server-protocol`, wrote `workspace/T09/test_strategy.md`, and linked the change to `T09`.
- 2026-04-11 11:43 CST: Spec review passed with no high-severity issues; moved `T09` to `implementing`.
- 2026-04-11 11:51 CST: Implemented `internal/codex` session protocol handling with package-scoped TDD coverage, passed `go test ./internal/codex/...`, validated `codex-app-server-protocol`, and moved `T09` to `verifying`.
- 2026-04-11 11:52 CST: Fresh verification passed (`go test -count=1 ./internal/codex/...`, `go test -count=1 ./...`, `make build`, `make lint`, `make test-e2e`, `openspec validate codex-app-server-protocol`) and moved `T09` to `review`.
- 2026-04-11 12:00 CST: Code review found four blocking protocol/verification issues; fixes were applied and reverified (`go test -count=1 ./internal/codex/...`, `go test -count=1 ./...`, `make build`, `make lint`, `make test-e2e`, `openspec validate codex-app-server-protocol`), and follow-up review found no blocking issues.
- 2026-04-11 12:02 CST: Final comparison recorded in `workspace/T09/final_compare.md`, synced `codex-app-server-protocol` into main specs, archived the change to `openspec/changes/archive/2026-04-11-codex-app-server-protocol/`, confirmed `openspec validate --specs`, and marked `T09` done.
- 2026-04-11 13:13 CST: Claimed `T12` for end-to-end delivery in isolated worktree `.worktrees/t12-linear-toolbridge`. Owner=`Codex`, workspace=`workspace/T12`.
- 2026-04-11 13:13 CST: Created `workspace/T12/` in the isolated worktree after a passing baseline (`go test ./...`) and moved `T12` to `research`.
- 2026-04-11 13:25 CST: Accepted `workspace/T12/final_impl.md` after round-two rubric review (reviews 93/91, avg 92/100, no high-severity issues) and moved `T12` to `spec`.
- 2026-04-11 13:53 CST: Created apply-ready OpenSpec change `linear-toolbridge`, validated it with `openspec validate linear-toolbridge`, wrote `workspace/T12/test_strategy.md`, and linked the change to `T12`.
- 2026-04-11 13:55 CST: Spec review passed with no high-severity issues, the low-risk proposal scope note was fixed and revalidated, and `T12` moved to `implementing`.
- 2026-04-11 14:05 CST: Implemented the generic Codex tool boundary and Linear ToolBridge with package-level TDD coverage, passed `go test ./internal/codex/... ./internal/toolbridge/...`, and moved `T12` to `verifying`.
- 2026-04-11 14:07 CST: Fresh verification passed (`go test ./...`, `make build`, `make lint`, `make test`, `make test-e2e`, `make verify`, `openspec validate linear-toolbridge`) and moved `T12` to `review`.
- 2026-04-11 14:24 CST: Code review found no blocking issues; final comparison was recorded in `workspace/T12/final_compare.md`, residual risks were recorded in `workspace/T12/todo.md`, specs were synced, `linear-toolbridge` was archived to `openspec/changes/archive/2026-04-11-linear-toolbridge/`, final validation passed (`openspec validate --specs`, `make verify`), and `T12` was marked done.
- 2026-04-11 16:21 CST: Claimed `T13` for end-to-end delivery in isolated worktree `.worktrees/t13-linear-workflow-bundle` after a passing baseline (`go test ./...`). Owner=`Codex`, workspace=`workspace/T13`.
- 2026-04-11 16:21 CST: Created `workspace/T13/` in the isolated worktree and moved `T13` to `research`.
- 2026-04-11 16:21 CST: Accepted `workspace/T13/final_impl.md` after rubric review (reviews 90/88, avg 89/100, no high-severity issues) and moved `T13` to `spec`.
- 2026-04-11 16:21 CST: Created and validated OpenSpec change `linear-workflow-bundle` for `T13` and linked it in the task board.
- 2026-04-11 16:21 CST: Spec review passed with no high-severity issues; moved `T13` to `implementing`.
- 2026-04-11 16:21 CST: Implemented `internal/workflow` with TDD coverage for selection, prompt fallback, Linear ToolBridge wiring, and dependency boundaries; moved `T13` to `verifying`.
- 2026-04-11 16:21 CST: Fresh verification passed (`go test -count=1 ./internal/workflow/...`, `go test -count=1 ./...`, `make build`, `make lint`, `openspec validate --type change linear-workflow-bundle`) and moved `T13` to `review`.
- 2026-04-11 16:21 CST: Code review found no blocking issues for `T13`; accepted residual verification notes were recorded in `workspace/T13/todo.md`.
- 2026-04-11 16:21 CST: Final comparison recorded in `workspace/T13/final_compare.md`, synced `workflow-selection` into main specs, archived `linear-workflow-bundle` to `openspec/changes/archive/2026-04-11-linear-workflow-bundle/`, final validation passed (`openspec validate --specs`, `make verify`), and marked `T13` done.
- 2026-04-11 17:01 CST: Claimed `T14` for end-to-end delivery in isolated worktree `.worktrees/t14-end-to-end-run-integration` after a passing baseline (`go test ./...`). Owner=`Codex`, workspace=`workspace/T14`.
- 2026-04-11 17:01 CST: Created `workspace/T14/` in the isolated worktree and moved `T14` to `research`.
- 2026-04-11 17:15 CST: Accepted `workspace/T14/final_impl.md` after round-two rubric review (reviews 88/93, avg 90.5/100, no high-severity issues) and moved `T14` to `spec`.
- 2026-04-11 17:15 CST: Created and validated OpenSpec change `end-to-end-run-integration`, wrote `workspace/T14/test_strategy.md`, and linked the change to `T14`.
- 2026-04-11 17:15 CST: Tightened `T14` verification gates after spec review so the task board matches `workspace/T14/final_impl.md` and `workspace/T14/test_strategy.md`.
- 2026-04-11 17:22 CST: Spec review passed after targeted fixes; moved `T14` to `implementing`.
- 2026-04-11 17:32 CST: Implemented the runtime assembly seam, memory no-network path, Linear tool injection, post-turn refresh/max-turn handling, event normalization, startup cleanup, and idempotent shutdown tests; moved `T14` to `verifying`.
- 2026-04-11 17:34 CST: Fresh verification passed (`go test -count=1 ./internal/orchestrator/... ./internal/cli/... ./cmd/symphony/...`, `go test -count=1 ./internal/...`, `go test -count=1 ./...`, `make build`, `make lint`, `make test-e2e`, `openspec validate --type change end-to-end-run-integration`, `git diff --check`) and moved `T14` to `review`.
- 2026-04-11 17:45 CST: Code review fixed prompt-marker leakage, config snapshot divergence, and executable signal shutdown; fresh verification passed; recorded `workspace/T14/code_review.md`, `workspace/T14/final_compare.md`, and `workspace/T14/todo.md`.
- 2026-04-11 17:45 CST: Synced `end-to-end-run-integration` into main specs, archived the change to `openspec/changes/archive/2026-04-11-end-to-end-run-integration/`, final validation passed (`make verify`, `openspec validate --specs`, `git diff --check`), and marked `T14` done.
- 2026-04-12 00:12 CST: Claimed `T15` for end-to-end delivery in isolated worktree `.worktrees/t15-httpapi-compatibility` after a passing baseline (`go test ./...`). Owner=`Codex`, workspace=`workspace/T15`.
- 2026-04-12 00:12 CST: Created `workspace/T15/` in the isolated worktree and moved `T15` to `research`.
- 2026-04-12 00:26 CST: Accepted `workspace/T15/final_impl.md` after round-two rubric review (reviews 86/92, avg 89/100, no high-severity issues) and moved `T15` to `spec`.
- 2026-04-12 00:28 CST: Created and validated OpenSpec change `http-api-compatibility` for `T15` and linked it in the task board.
- 2026-04-12 00:31 CST: Spec review passed with no high-severity issues; moved `T15` to `implementing`.
- 2026-04-12 00:34 CST: Implemented `internal/httpapi` with TDD coverage for routes, state DTOs, issue DTOs, refresh semantics, error envelopes, and dependency boundaries; moved `T15` to `verifying`.
- 2026-04-12 00:38 CST: Fresh verification passed (`go test -count=1 ./internal/httpapi/...`, `go test -count=1 ./...`, `make build`, `make lint`, `make test-e2e`, `make verify`, `openspec validate --type change http-api-compatibility`, `openspec validate --specs`, `git diff --check`) and moved `T15` to `review`.
- 2026-04-12 00:42 CST: Review found no blocking implementation issues, final comparison was recorded in `workspace/T15/final_compare.md`, synced `http-api-compatibility` into main specs, archived the change to `openspec/changes/archive/2026-04-12-http-api-compatibility/`, final validation passed (`make verify`, `openspec validate --specs`, `git diff --check`), and marked `T15` done.
- 2026-04-12 00:57 CST: Claimed `T16` for end-to-end delivery in isolated worktree `.worktrees/t16-terminal-dashboard-compatibility` after a passing baseline (`go test ./...`). Owner=`Codex`, workspace=`workspace/T16`.
- 2026-04-12 00:57 CST: Created `workspace/T16/` in the isolated worktree and moved `T16` to `research`.
- 2026-04-12 00:57 CST: Accepted `workspace/T16/final_impl.md` after round-three rubric review (reviews 97/94, avg 95.5/100, no high-severity issues) and moved `T16` to `spec`.
- 2026-04-12 00:57 CST: Created and validated OpenSpec change `terminal-dashboard-compatibility` for `T16` and linked it in the task board.
- 2026-04-12 00:57 CST: Wrote `workspace/T16/test_strategy.md`, cleared spec review with no high-severity issues, and moved `T16` to `implementing`.
- 2026-04-12 00:57 CST: Implemented dashboard projection, ANSI renderer, render gate, fixture provenance checks, and Codex event summaries with package gates passing; moved `T16` to `verifying`.
- 2026-04-12 00:57 CST: Fresh verification passed (`go test ./internal/observability/... ./internal/dashboard/...`, `go test ./internal/cli/...`, `go test ./...`, `make build`, `make lint`, `make verify`, `openspec validate --type change terminal-dashboard-compatibility`, `openspec validate --specs`, `git diff --check`) and moved `T16` to `review`.
- 2026-04-12 01:35 CST: Addressed T16 review findings for idle rerender, terminal repaint prefix, due-now refresh text, fixture provenance strength, and turn-completed event messages; follow-up recorded in `workspace/T16/code_review_followup.md`.
- 2026-04-12 01:35 CST: Recorded T16 final comparison and residual notes, synced `terminal-dashboard-compatibility` into main specs, archived the change to `openspec/changes/archive/2026-04-12-terminal-dashboard-compatibility/`, final validation passed (`make verify`, `openspec validate --specs`, `git diff --check`), and marked `T16` done.
- 2026-04-12 01:42 CST: Claimed `T17` for end-to-end delivery in isolated worktree `.worktrees/t17-web-dashboard-static-assets` after a passing baseline (`go test ./...`). Owner=`Codex`, workspace=`workspace/T17`.
- 2026-04-12 01:42 CST: Created `workspace/T17/` in the isolated worktree and moved `T17` to `research`.
- 2026-04-12 01:49 CST: Accepted `workspace/T17/final_impl.md` after rubric review (reviews 91/87, avg 89/100, no high-severity issues) and moved `T17` to `spec`.
- 2026-04-12 01:53 CST: Created and linked OpenSpec change `web-dashboard-static-assets` for `T17`, wrote `workspace/T17/test_strategy.md`, and tightened spec artifacts to match `workspace/T17/final_impl.md`.
- 2026-04-12 01:55 CST: Spec review passed with no high-severity issues, test strategy gaps were fixed, and `T17` moved to `implementing`.
- 2026-04-12 02:00 CST: Implemented `internal/web` with TDD coverage for root HTML, snapshot updates, unavailable rendering, static assets, API delegation, route errors, and boundary dependencies; package gate and change validation passed, then moved `T17` to `verifying`.
- 2026-04-12 02:01 CST: Fresh verification passed (`go test ./internal/web/...`, `go test ./internal/httpapi/... ./internal/observability/...`, `go test ./...`, `make build`, `make lint`, `make test-e2e`, `make verify`, `openspec validate web-dashboard-static-assets`) and moved `T17` to `review`.
- 2026-04-12 02:08 CST: Code review found one blocking reachability issue and one asset-test coverage gap; both were fixed with runtime server wiring, expanded tests, updated artifacts, and fresh `make verify`.
- 2026-04-12 02:11 CST: Follow-up review found no blocking issues; final comparison and residual notes were recorded in `workspace/T17/`.
- 2026-04-12 02:12 CST: Synced `web-dashboard-static-assets` into main specs, archived the change to `openspec/changes/archive/2026-04-12-web-dashboard-static-assets/`, final validation passed (`make verify`, `openspec validate --specs`, `git diff --check`), and marked `T17` done.
