# Runtime Listening And Lifecycle Productization

## Writing Principles

- A reader who did not see the chat should understand the goal, what was verified, and what still needs product work.
- Separate product behavior from operator actions. Do not credit manual Linear or GitHub changes as go-symphony behavior.
- Keep the story short enough to be useful during implementation. Link to detailed evidence instead of duplicating it.
- State why each follow-up matters, not just what to build.

## Context

The original validation goal was to prove that go-symphony can take work from a Linear project, execute the task with Codex, and hand the result back through GitHub and Linear.

T19 added a guarded verification path and live smoke workflow. The strict rerun against `YOU-22` proved the core loop can work when scoped to one issue:

1. Read the intended Linear issue from the intended project.
2. Create a managed workspace for the target repository.
3. Run Codex in that workspace.
4. Produce a Python hello-world change.
5. Verify the output.
6. Open a PR in the correct target repository.
7. Comment back to Linear.
8. Move the Linear issue to review.

Detailed evidence is in `workspace/T19/verification.md`. The stricter source-attribution SOP is in `workspace/T19/reverification_sop.md`.

## What Was Verified

- `cmd/symphony-verify linear --only-issue` can prove Linear reads for one issue without starting Codex or mutating Linear.
- `cmd/symphony-verify run --only-issue YOU-22` can exercise the runtime, workspace lifecycle, Codex app-server, and target-repository PR path for one issue.
- The strict rerun opened `https://github.com/Miss-you/test-go-symphony/pull/1` from a go-symphony-managed workspace.
- Linear state changes in the strict rerun were caused by go-symphony workflow hooks, not by manual GraphQL mutations.

## Boundary Problems

### 1. Project-Wide Listening Is Not Yet Productized

The successful run used `cmd/symphony-verify run --only-issue YOU-22`. That is intentionally narrow. It is good for controlled validation, but it is not the same as running go-symphony as a long-lived service that watches a project and safely picks up whichever eligible issue appears next.

Why this matters:

- Real usage should not require an operator to name every issue.
- Project-level operation needs stronger safeguards for candidate selection, concurrency, stale workspaces, retries, and observability.
- A verification helper should not become the production interface by accident.

Follow-up work:

- Define the supported production command for long-lived project watching.
- Define a single source of truth for the watched project. Acceptable sources are an explicit workflow/env field, a durable task-board field, or a Linear project field that can be read back during verification.
- Decide how operators scope work safely inside that project: active states, assignee/routing, labels, or an explicit allowlist.
- Add an acceptance test or live run that starts without `--only-issue`, picks up a disposable eligible issue, and does not touch unrelated project issues.
- Add a clear shutdown/idle behavior for verification mode so successful single-issue runs do not need to wait for `--timeout`.

Acceptance standard:

- go-symphony can run against a project without `--only-issue`.
- The watched Linear project is resolved from the documented source of truth and is printed or exposed in verification evidence.
- It claims exactly the intended eligible issue under documented routing rules.
- It leaves unrelated active issues untouched.
- A run pointed at the wrong project fails verification, even if it can refresh an issue by explicit ID.
- Dashboard/API evidence shows when the service is idle, running, retrying, or stopped.

### 2. Lifecycle Writes Are Hook-Based, Not First-Class Runtime Behavior

The strict rerun used `before_run` and `after_run` workflow hooks to move Linear state, verify code, commit, push, create the PR, comment on Linear, and move the issue to review. This is valid go-symphony behavior because the hooks run inside the go-symphony runtime, but it is still a workflow script rather than a first-class product capability.

Why this matters:

- Hook scripts are harder to validate, reuse, and observe than typed runtime behavior.
- `after_run` is best-effort in the workspace lifecycle, so a green Codex turn alone does not prove PR publication or Linear handoff succeeded.
- The current approach mixes lifecycle policy with per-workflow shell code.

Follow-up work:

- Add a Linear-specific compatibility-shell lifecycle adapter for claim and handoff actions.
- The adapter should be triggered from the runtime/workflow assembly boundary, for example `internal/workflow` plus `internal/toolbridge/linear`, instead of adding generic write behavior to the core orchestrator.
- Keep provider-specific writes out of `internal/tracker`; do not introduce a universal tracker write API.
- Reuse `internal/toolbridge/linear.Bridge.UpdateIssueState` and `CreateComment` where possible.
- Make PR publication an explicit runtime/hook outcome with observable success or failure.
- In the product path, Codex writes code; go-symphony lifecycle code owns state transitions and PR publication.

Acceptance standard:

- A run start can move a Linear issue to `In Progress` through go-symphony-owned lifecycle code.
- A successful handoff can comment with the PR URL and move the issue to `In Review`.
- Failures in state transition, verification, push, PR creation, or Linear comment are visible in runtime state or logs.
- The same source-attribution rule from `workspace/T19/reverification_sop.md` can prove the side effects were product behavior.

## Problems Encountered During T19 Validation

- The first live state-transition evidence was invalid because the operator manually mutated Linear state.
- The first PR evidence was invalid because the PR was manually opened in `Miss-you/go-symphony`, which was not the target repository for the Linear task.
- The workflow project slug from local env did not match the Linear project containing `YOU-22`; read probe initially returned zero candidates until the correct project slug was used.
- A hook failed on `datetime.UTC`, which is not available in the local Python version. The strict rerun switched to `datetime.timezone.utc`.
- The verification runner currently waits for its timeout even after the single issue has no running or retrying entry.

## Next Implementation Slice

Build a product-ready path for:

1. Long-lived project watching without `--only-issue`.
2. First-class Linear lifecycle claim and handoff behavior in the compatibility shell.
3. Observable PR publication and Linear handoff results.
4. A verification mode that exits cleanly when the scoped run is complete.

This should be tracked as new work separate from T19. T19 proved the core loop is feasible; this TODO captures what is needed to make it durable and operator-safe.
