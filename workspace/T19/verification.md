# T19 Verification Evidence

Fresh verification run from isolated worktree:

```text
/Users/lihui/Documents/GitHub/go-symphony/.worktrees/t19-verification-workflows
```

## Targeted Gates

- `go test ./internal/tracker/... ./cmd/symphony-verify/...`
  - Passed.
  - Proves the read-only filter and verification command parser/probe/runtime wrapper seams.
- `openspec validate --type change verification-workflows`
  - Passed.
  - Proves the change artifacts are internally valid.

## Broad Gates

- `go test ./...`
  - Passed.
- `make build`
  - Passed.
- `make test-e2e`
  - Passed.
- `make verify`
  - Passed after removing one lint finding in a test fake.
  - Passed again after code review fixes that split `internal/verify/linearprobe` and fixed the docs template.
  - Includes build, lint, unit tests, and e2e-tagged tests.
- `openspec validate --specs`
  - Passed, 18 specs valid.
- `git diff --check`
  - Passed.

## Strict Reverification: `YOU-22`

This rerun was added after discovering that the earlier `YOU-21` and `YOU-22` state transitions and the `go-symphony` PR were operator-driven and therefore invalid as product acceptance evidence.

SOP:

- `workspace/T19/reverification_sop.md`
- Reviewed by an independent subagent before rerun.
- Review findings fixed before live execution:
  - target repo must have an explicit source of truth;
  - operator actions are limited to env prep, read-only observation, and starting/stopping go-symphony;
  - runtime hooks, not manual shell actions, own Linear state transitions and PR publication for this rerun;
  - source checkout and target workspace must be proven distinct.

Source of truth:

- Linear issue: `YOU-22` / `https://linear.app/yousali/issue/YOU-22/write-python-hello-world`
- Target repository: `https://github.com/Miss-you/test-go-symphony.git`
- Target repository source: explicit `TARGET_REPO_URL` / `TARGET_REPO_FULL_NAME` operator input for this verification.
- Branch: `task/you-22-python-hello-world`
- Workflow: local excluded `WORKFLOW.hello.md`
- Workspace root: `/tmp/go-symphony-you22-strict2-workspaces`

Stage 1 read-only probe:

- Initial probe with `t19.env` project slug `93e67ac5cc77` returned `candidates: 0` while `refresh: 1` could read `YOU-22`; this proved the env project slug did not match the issue's Linear project.
- Corrected run with `LINEAR_PROJECT_SLUG=55dad0d295b6` returned:
  - `project: 55dad0d295b6`
  - `active_states: In Progress`
  - `candidates: 1`
  - `YOU-22 | In Progress | write python hello world`
  - `refresh: 1`

Stage 2 runtime hook state transition:

- A first strict attempt used `/tmp/go-symphony-you22-strict-workspaces`.
  - go-symphony `before_run` moved `YOU-22` to `In Progress`.
  - The hook then failed because the local Python did not support `datetime.UTC`.
  - This attempt is not accepted as a complete run, but it is valid evidence that the state mutation came from go-symphony runtime hook execution rather than manual GraphQL.
- The fixed strict run used `/tmp/go-symphony-you22-strict2-workspaces`.
  - Runtime dashboard: `http://127.0.0.1:51882/`
  - Codex session id: `019d9037-8fe0-7833-a11d-0b450ccf2151`
  - `before_run` Linear comment at `2026-04-15T08:17:33Z`: `go-symphony runtime before_run claimed YOU-22...`
  - `after_run` Linear comment at `2026-04-15T08:19:29Z`: `go-symphony runtime after_run verified hello-world output and opened PR...`
  - Final Linear state: `In Review`
  - Final runtime snapshot after `--timeout`: `running: 0`, `retrying: 0`, `total_tokens: 0`

Stage 3 workspace/code evidence:

- Target workspace top: `/private/tmp/go-symphony-you22-strict2-workspaces/YOU-22`
- Source checkout top remained `/Users/lihui/Documents/GitHub/go-symphony/.worktrees/t19-verification-workflows`
- Target workspace remote: `https://github.com/Miss-you/test-go-symphony.git`
- Target workspace branch: `task/you-22-python-hello-world`
- `python3 examples/python_hello/hello.py` output: `hello from codex`
- No `examples/python_hello` directory exists in the go-symphony source checkout.

Stage 4 PR evidence:

- PR: `https://github.com/Miss-you/test-go-symphony/pull/1`
- Repository: `Miss-you/test-go-symphony`
- State: open draft
- Base: `main`
- Head: `task/you-22-python-hello-world`
- Commit: `f75298daaeaeb36494592da05e2da6ac828818ce`
- Commit author: `go-symphony verification <go-symphony@example.local>`
- Changed files:
  - `examples/python_hello/README.md`
  - `examples/python_hello/hello.py`

Known deviations:

- `YOU-22` still has the old invalid attachment `https://github.com/Miss-you/go-symphony/pull/22`; it must not be used as acceptance evidence.
- `cmd/symphony-verify run` still waits for `--timeout` instead of exiting when the filtered issue has no running/retrying entry. The successful side effects were verified through read-only Linear, GitHub, workspace, and dashboard checks before the timeout elapsed.
  - Passed again after code review fixes.

## Manual Live Smoke

Not run in this task because it requires real Linear credentials and a disposable Linear issue, and it intentionally launches real `codex app-server`.

The commands are documented in `docs/verification-workflows.md`:

```bash
go run ./cmd/symphony-verify linear WORKFLOW.verify.md
go run ./cmd/symphony-verify run \
  --i-understand-that-this-will-be-running-without-the-usual-guardrails \
  --only-issue ABC-123 \
  --port 34567 \
  --timeout 10m \
  WORKFLOW.verify.md
```

## Final Completion Gates

After syncing the spec, archiving `verification-workflows`, and marking `T19` done, the final gate passed:

- `make verify`
  - Passed.
  - Covered `go build ./...`, `golangci-lint run ./...` with `0 issues.`, `go test ./...`, and `go test -count=1 -tags=e2e ./...`.
- `openspec validate --specs`
  - Passed.
  - Reported 19 passed, 0 failed.
- `git diff --check`
  - Passed.

## Local Linear Smoke With `t19.env`

Follow-up local validation used `t19.env` from the worktree without printing secrets.

- Fixed `project_slug: $LINEAR_PROJECT_SLUG` handling so the workflow resolves the env var instead of sending the literal string to Linear.
- Updated `t19.env` to use the visible Linear project slug `93e67ac5cc77`.
- Updated `WORKFLOW.verify.md` active states to `Backlog` and `In Review`, matching the current project states.
- `go run ./cmd/symphony-verify linear --limit 20 WORKFLOW.verify.md`
  - Passed against live Linear.
  - Read 7 active candidates, 10 terminal items, and refreshed `YOU-21`.
- Direct Linear GraphQL smoke against `YOU-21`
  - `commentCreate` returned `true`.
  - `issueUpdate` returned `true` while setting the issue to its current state, so no state transition was performed.
- Fixed Linear write mutation variable types from `ID` to `String` after live schema validation rejected the old mutation shape.
- State-transition acceptance against `YOU-21 / test-go-symphony`
  - Available team states were confirmed before mutation.
  - `Backlog -> In Progress` returned success and read back `In Progress`.
  - `In Progress -> In Review` returned success and read back `In Review`.
  - `In Review -> Done` returned success and read back `Done`.
  - A final probe read 6 active candidates, 11 terminal items, and refreshed `YOU-21` as `Done`.
- Acceptance issues recorded for future operators:
  - `LINEAR_PROJECT_SLUG=yousali` was not the project slug visible to the token; the working slug was discovered through Linear `projects`.
  - `active_states: ["Symphony Ready"]` did not match any current project state, so candidates were initially empty.
  - Local `t19.env` contains credentials and is excluded through `.git/info/exclude`.
- `go test ./internal/config/... ./internal/toolbridge/linear/... ./cmd/symphony-verify/...`
  - Passed.
- `git diff --check`
  - Passed.
