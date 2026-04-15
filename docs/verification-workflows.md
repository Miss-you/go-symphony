# Verification Workflows

Use these workflows when you want to validate live go-symphony behavior in two separate stages:

1. Linear can be read correctly.
2. A selected Linear issue can drive a real runtime and Codex app-server run.

Both workflows use the same `WORKFLOW.md` configuration shape as the main `symphony` command.

## Stage 1: Probe Linear Reads

This stage is read-only. It loads the workflow, creates the Linear reader, and checks candidates, terminal-state reads, and refresh-by-ID reads. It does not start workspaces, the orchestrator, or Codex.

```bash
export LINEAR_API_KEY='...'

go run ./cmd/symphony-verify linear \
  --limit 10 \
  WORKFLOW.verify.md
```

To explicitly refresh one or more known Linear issue IDs:

```bash
go run ./cmd/symphony-verify linear \
  --refresh-id '<linear-issue-id>' \
  --refresh-id '<another-linear-issue-id>' \
  WORKFLOW.verify.md
```

Expected report shape:

```text
Linear probe
project: your-project-slug
active_states: Todo, In Progress
terminal_states: Done, Canceled
candidates: 1
- ABC-123 | Todo | Small disposable verification task
terminal: 0
refresh: 1
- ABC-123 | Todo | Small disposable verification task
```

If this fails, fix Linear credentials, project slug, state names, or assignee routing before running Codex.

## Stage 2: Run One Issue Through Runtime And Codex

This stage starts the real runtime and launches real `codex app-server`. Use a disposable Linear issue and a narrow `WORKFLOW.verify.md`.

The `--only-issue` flag is required so the smoke run cannot sweep every active candidate.

```bash
export LINEAR_API_KEY='...'

go run ./cmd/symphony-verify run \
  --i-understand-that-this-will-be-running-without-the-usual-guardrails \
  --only-issue ABC-123 \
  --port 34567 \
  --timeout 10m \
  WORKFLOW.verify.md
```

Then inspect:

```bash
curl -s http://127.0.0.1:34567/api/v1/state
curl -s -X POST http://127.0.0.1:34567/api/v1/refresh
```

`--only-issue` matches either the normalized work item `ID` or `Identifier`. The command filters candidate, refresh, and terminal-cleanup reads for the smoke run. This is intentionally narrower than normal daemon startup.

## Minimal Workflow Template

```md
---
tracker:
  kind: linear
  api_key: $LINEAR_API_KEY
  project_slug: $LINEAR_PROJECT_SLUG
  active_states: ["Symphony Ready"]
  terminal_states: ["Done", "Canceled"]

workspace:
  root: /tmp/go-symphony-verify-workspaces

hooks:
  after_create: |
    git clone "$SOURCE_REPO_URL" .
  timeout_ms: 120000

agent:
  max_concurrent_agents: 1
  max_turns: 1
  max_retry_backoff_ms: 300000

codex:
  command: codex app-server
  approval_policy: never
  thread_sandbox: workspace-write
  turn_timeout_ms: 3600000
  read_timeout_ms: 5000
  stall_timeout_ms: 300000

server:
  host: 127.0.0.1
  port: 34567
---

You are working on a disposable go-symphony verification issue.

Issue: {{ issue.identifier }}
Title: {{ issue.title }}
State: {{ issue.state }}
URL: {{ issue.url }}

Description:
{{ issue.description }}

Make one small, reversible change in this workspace, run the most relevant Go test, and report the result.
```

## Live-Run Risks

Stage 2 is not a dry run:

- It starts real `codex app-server`.
- It can modify files inside the selected workspace.
- It can call the configured `linear_graphql` tool during the Codex session.
- It can mutate the target Linear issue if the workflow prompt or repo skills instruct Codex to do so.

Use a disposable issue and a workflow with a narrow active state such as `Symphony Ready`.

## Live Acceptance Notes

The April 2026 local acceptance run surfaced a few details that are easy to forget:

- Load local env files with `set -a; . ./t19.env; set +a` so `LINEAR_API_KEY` and `LINEAR_PROJECT_SLUG` are exported to the Go process.
- `project_slug` must be the Linear project `slugId` that the API token can see. It is not necessarily the workspace name or team key.
- If the probe prints `project: $LINEAR_PROJECT_SLUG`, the workflow was not resolving the environment variable. Current code supports `project_slug: $LINEAR_PROJECT_SLUG`.
- If candidates are `0`, first query the project issue list directly and check the exact state names. Linear state names are case-sensitive user-facing names such as `Backlog`, `In Progress`, `In Review`, and `Done`.
- After an issue moves to a terminal state such as `Done`, it should disappear from `candidates` and appear under `terminal`.
- Linear write mutations in the current GraphQL schema expect `String!` variables for `issueId` and `stateId`. Do not use `ID!` in `commentCreate` or `issueUpdate` helper mutations.
- Keep local credential files such as `t19.env` out of git. The repository docs should show env variable names and placeholders only.

The verified state-transition smoke path was:

```text
Backlog -> In Progress -> In Review -> Done
```

Use a disposable issue for this test, because the final state is a real Linear mutation.
