# T19 Reverification SOP

## Goal

Verify the two live workflows without crediting manual operator actions as product behavior:

1. Linear intake: go-symphony can read the intended Linear project, list and refresh the intended issue, and avoid unrelated issues.
2. End-to-end execution: a go-symphony-managed run can move the Linear issue through the expected handoff states, create code in the target repository workspace, verify it locally, publish a PR to the correct repository, and leave auditable evidence.

## Source Attribution Rule

Every externally visible side effect must be attributed to one of these sources:

- `go-symphony/reader`: read-only Linear probe or runtime tracker reads.
- `go-symphony/runtime`: orchestrator, worker, workspace lifecycle, or configured hooks executed by the go-symphony process.
- `go-symphony/codex-tool`: Codex running inside a go-symphony-managed session and using the injected `linear_graphql` tool.
- `operator`: direct shell, direct GraphQL, manual GitHub, or manual Linear action outside the go-symphony run.

Only the first three count as product verification. `operator` actions are limited to environment preparation, read-only inspection, starting the go-symphony command, and stopping it if it exceeds the agreed timeout.

The operator must not prepare results for Stage 3 or Stage 4. The operator must not pre-clone the target repository into the managed workspace, create the task branch, edit task files, commit, push, open a PR, add Linear comments, or mutate Linear state.

## Required Fixtures

- A disposable Linear issue whose current state is known before the run.
- A correct target GitHub repository URL from an explicit source of truth. The source must be one of:
  - a workflow/env setting such as `TARGET_REPO_URL` and `TARGET_REPO_FULL_NAME`;
  - a Linear issue or project field that explicitly contains the repository URL;
  - a user-provided repository URL recorded in the audit log.
- For `YOU-22`, use `TARGET_REPO_URL=https://github.com/Miss-you/test-go-symphony.git` and `TARGET_REPO_FULL_NAME=Miss-you/test-go-symphony` unless the user provides a different explicit target.
- A branch name derived from the Linear issue identifier, for example `task/you-22-python-hello-world`.
- A local workflow file excluded from git, such as `WORKFLOW.hello.md`.
- A local env file excluded from git, such as `t19.env`.

## Stage 1: Linear Intake Probe

Run the read-only probe with the same workflow file and env the runtime will use.

Acceptance:

- The reported `project` value is the resolved Linear project slug, not a literal `$ENV_VAR`.
- `candidates` contains only the intended issue when `--only-issue` is used.
- Terminal-state reads work and do not start workspaces or Codex.
- No Linear comments, state transitions, or GitHub side effects occur.

Failure means config, state names, project slug, assignee routing, or the issue filter must be fixed before starting Codex.

## Stage 2: State Transition Proof

State transitions must be caused by the go-symphony run.

Current rerun mechanism:

- `before_run` performs the claim transition to `In Progress` from inside the managed workspace lifecycle.
- `after_run` verifies the generated project, publishes the PR, records a Linear comment, and performs the handoff transition to `In Review` from inside the managed workspace lifecycle.
- The workflow prompt must not ask Codex to perform Linear state transitions or PR publication for this rerun. Codex is responsible for writing and locally testing the hello-world project only.
- Because `after_run` is best-effort in the workspace lifecycle, the final acceptance check must verify the PR URL, Linear comment, and final state externally. A green Codex run alone is not sufficient.

Alternative mechanism for a separate test:

- Codex may use the injected `linear_graphql` tool to mutate Linear during a go-symphony-managed session.
- That proves toolbridge write capability, but it does not prove runtime lifecycle ownership.
- If first-class runtime lifecycle transitions are required later, add a Linear-only compatibility-shell hook that calls `internal/toolbridge/linear.Bridge.UpdateIssueState`. Keep this out of `internal/tracker` and do not add a universal tracker write API.

Acceptance:

- The issue state is captured before and after the run.
- The state changes happen between the go-symphony run start and stop timestamps.
- Evidence points to `go-symphony/runtime`, not `operator`.
- Manual direct GraphQL state mutation by the operator invalidates the state-transition acceptance for that run.

## Stage 3: Workspace And Code Proof

The target repository must be cloned by the go-symphony workspace lifecycle, not edited in the go-symphony source checkout.

Acceptance:

- The workspace path is under the configured workspace root and matches the Linear issue identifier.
- `after_create` clones the correct target repository URL.
- The working branch is the intended task branch.
- `git rev-parse --show-toplevel` in the go-symphony source checkout and the target workspace return different directories.
- `git remote get-url origin` inside the target workspace equals `TARGET_REPO_URL`.
- Codex creates the expected hello-world Python project in the target workspace.
- Local verification runs inside that workspace and produces the expected output.
- No product code for the target task is committed to the go-symphony repository.

## Stage 4: PR Proof

PR creation must be performed inside the go-symphony-managed execution path.

Current rerun mechanism:

- A go-symphony `after_run` hook performs local verification, commit, push, and PR creation from inside the managed workspace.
- Codex must not run `git commit`, `git push`, or `gh pr create` in this rerun. That keeps the execution boundary simple: Codex writes code, go-symphony runtime hooks publish.

Acceptance:

- The PR repository matches the target repository.
- The PR branch matches the planned task branch.
- The PR contains only target task files.
- The PR URL is added to Linear by the go-symphony `after_run` hook or by the PR integration triggered by the go-symphony-created PR.
- A manually opened PR by the operator is not valid acceptance evidence.

## Stage 5: Audit Record

Record the run in `workspace/T19/verification.md`.

Required fields:

- Linear issue identifier and URL.
- Target repository URL and the source of truth used to select it.
- Workflow file used.
- Workspace root and issue workspace path.
- Run start and stop timestamps.
- State before and after, with source attribution.
- Operator observation commands.
- Product-generated evidence: workspace path, hook side effects, PR URL, Linear comment, and Linear state.
- Known deviations.

## Known Invalid Evidence

- The previous `YOU-21` and `YOU-22` state transitions were performed manually by the operator and do not prove go-symphony state-transition behavior.
- `https://github.com/Miss-you/go-symphony/pull/22` was opened manually by the operator from a workspace and targets the wrong repository for the Linear task. It must not be used as acceptance evidence for the target-repository workflow.
