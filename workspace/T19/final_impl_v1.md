# T19 Final Implementation Draft v1

## Goal

Add small operator-facing verification helpers so a maintainer can validate the two risky live flows separately:

1. Linear read probe: load a real `WORKFLOW.md`, call the Linear reader, and inspect candidate / terminal / refresh data without starting workspaces, the orchestrator, or Codex.
2. Controlled runtime smoke: run the normal Symphony runtime against real Linear and real `codex app-server`, but restrict candidate dispatch to one explicit issue identifier so the operator can safely verify the issue-to-Codex path.

This task should not reimplement orchestration, tracker reads, or Codex protocol handling.

## Existing Evidence

Original Symphony:

- The Elixir runtime has no standalone probe command, but its real read surfaces are `fetch_candidate_issues`, `fetch_issues_by_states`, and `fetch_issue_states_by_ids` in the Linear client.
- The live e2e path proves the full flow by creating a Linear issue, starting the runtime, running a real Codex app-server turn, checking workspace side effects, and closing the issue.
- The operator contract is the `WORKFLOW.md` file plus CLI startup flags.

Current Go implementation:

- `internal/trackers/linear.Reader` already exposes `ListCandidates`, `ListByStates`, and `RefreshByIDs`.
- `internal/cli.StartRuntime` already wires config, tracker reader, startup cleanup, orchestrator, workers, dashboard, workflow selection, and Codex execution.
- `internal/codex` already owns app-server protocol behavior.
- The current CLI starts the full runtime immediately; there is no read-only Linear probe.
- The current runtime has no operator-safe single-issue filter. A narrow active state works, but it depends on external Linear hygiene.

## Proposed Code Changes

### 1. Add a dedicated verification command

Create `cmd/symphony-verify` with two subcommands:

```text
symphony-verify linear [flags] [path-to-WORKFLOW.md]
symphony-verify run [flags] [path-to-WORKFLOW.md]
```

Keep `cmd/symphony` unchanged. Verification-only behavior should not become part of the production daemon CLI.

### 2. Add `linear`: a read-only Linear probe

Behavior:

- Accept an optional workflow path, defaulting to `WORKFLOW.md`.
- Accept `--limit <n>` for display truncation; default to `10`, `0` means no limit.
- Accept repeatable `--refresh-id <id>` to explicitly verify `RefreshByIDs`.
- Load typed settings through `config.NewStore(...).CurrentSettings()` so startup validation matches runtime semantics.
- Require `tracker.kind: linear`; fail fast for memory or unsupported providers.
- Create `linear.NewReader(settings.Provider, nil)`.
- Run:
  - `ListCandidates`
  - `ListByStates(settings.Provider.TerminalStates)`
  - `RefreshByIDs` for explicit refresh IDs, and for the first listed candidate when no explicit refresh ID is provided.
- Print a compact text report:
  - provider project, active states, terminal states
  - candidate count and summarized items
  - terminal count and summarized items
  - refresh count and summarized items

The probe must not import or call `internal/cli.StartRuntime`, `workspace`, `orchestrator`, or `codex`.

Testing seam:

- Keep the probe logic behind a small dependency-injected function so tests can pass a fake `tracker.TrackerReader`.
- Add a boundary test that `cmd/symphony-verify` does not import `internal/cli`, `internal/codex`, `internal/orchestrator`, or `internal/workspace` for the `linear` path.

### 3. Add `run`: a guarded single-issue runtime smoke

Behavior:

- Require the same guardrails acknowledgement flag used by `cmd/symphony` before starting runtime:
  `--i-understand-that-this-will-be-running-without-the-usual-guardrails`.
- Require `--only-issue <id-or-identifier>` so this command never launches an unbounded live sweep.
- The flag is optional and order-agnostic like `--port` and `--logs-root`.
- Accept optional `--port <n>` to enable the dashboard/API and optional `--timeout <duration>` to stop automatically; default `10m`, `0` means wait until interrupted.
- Load the workflow through `config.NewStore`, create the normal Linear reader, wrap it with a small read-only filter, and call `cli.StartRuntime` directly.
- Print the dashboard URL when available and print a compact final snapshot summary on shutdown.

The reader filter:

  - `ListCandidates` returns only work items whose `ID` or `Identifier` matches the value.
  - `ListByStates` is filtered the same way so startup terminal cleanup is scoped during this smoke command.
  - `RefreshByIDs` delegates to the underlying reader and then filters returned items by the same predicate.

This terminal-cleanup filtering is a smoke-safety tradeoff: it verifies the selected issue's runtime path, not broad startup cleanup coverage. The existing unfiltered `cmd/symphony` path remains the source of truth for normal daemon behavior.

### 4. Add operator documentation

Add `docs/verification-workflows.md` with two copyable workflows:

- Stage 1: Linear probe command.
- Stage 2: controlled single-issue runtime smoke with `symphony-verify run --only-issue`, a narrow `WORKFLOW.md`, dashboard/API checks, and shutdown expectations.

The doc should warn that live smoke still runs real Codex and may mutate the target repository workspace and Linear issue via `linear_graphql`.

## Testing Strategy

Unit tests:

- `internal/tracker` tests for the identifier filter wrapper:
  - filters candidate and terminal lists by `ID` and `Identifier`
  - preserves refresh results for the selected item
  - returns empty slices when there is no match
- `cmd/symphony-verify` tests:
  - parse `linear` and `run` arguments
  - reject missing/blank `run --only-issue`
  - reject `run` without guardrails acknowledgement
  - render bounded probe item summaries
  - run linear probe success with a fake reader and no network
  - reject non-Linear settings without network calls
  - prove the linear subcommand path has no dependency on runtime/Codex packages

Integration/e2e:

- Existing `make test-e2e` remains the no-network runtime smoke.
- Add no live Linear tests by default. The real Linear/Codex validation is manual and documented because it needs credentials and intentionally launches Codex.

Verification commands:

```bash
go test ./internal/tracker/... ./cmd/symphony-verify/...
go test ./...
make build
make test-e2e
make verify
openspec validate --type change verification-workflows
openspec validate --specs
git diff --check
```

## Boundaries

In scope:

- Read-only Linear probe.
- Single-issue runtime dispatch filter.
- Dedicated verification command that keeps production CLI behavior unchanged.
- Operator documentation.
- Tests proving no accidental runtime startup in the probe path and no broad candidate dispatch in the filtered runtime path.

Out of scope:

- Creating disposable Linear projects/issues.
- Adding a generic tracker write API.
- Adding a generic workpad abstraction.
- Changing the Codex app-server protocol.
- Changing orchestration scheduling semantics.
- Adding Lark or any new provider behavior.

## Acceptance Criteria

- An operator can run a command that verifies Linear candidate/terminal/refresh reads without starting Codex.
- An operator can run the real runtime against one chosen Linear issue using `symphony-verify run --only-issue`.
- Existing unfiltered runtime behavior remains unchanged.
- The new code stays at the CLI / verification edge and does not contaminate core provider-neutral runtime packages with Linear-specific writes.
