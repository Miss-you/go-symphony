# T19 Final Implementation

## Goal

Add operator-facing verification helpers for two live-risk flows:

1. Linear read probe: validate real Linear candidate, terminal-state, and refresh reads without starting runtime workers or Codex.
2. Controlled runtime smoke: launch the existing runtime and real `codex app-server` against one explicit Linear issue, so a maintainer can validate the issue-to-Codex path without sweeping all active candidates.

## Accepted Shape

Create a dedicated verification binary:

```text
cmd/symphony-verify
```

with two subcommands:

```text
symphony-verify linear [flags] [path-to-WORKFLOW.md]
symphony-verify run [flags] [path-to-WORKFLOW.md]
```

The production `cmd/symphony` CLI remains unchanged.

## `linear` Subcommand

Behavior:

- Accept optional workflow path, defaulting to `WORKFLOW.md`.
- Accept `--limit <n>` for report truncation; default `10`, `0` means no truncation.
- Accept repeatable `--refresh-id <id>` for explicit `RefreshByIDs` checks.
- Load workflow settings through `config.NewStore(...).CurrentSettings()` so validation matches runtime startup semantics.
- Require `tracker.kind: linear`.
- Create `internal/trackers/linear.Reader`.
- Run `ListCandidates`, `ListByStates(settings.Provider.TerminalStates)`, and `RefreshByIDs`.
- Print a compact report with project, active states, terminal states, counts, and summarized items.

The `linear` path must not import or call runtime, workspace, orchestrator, or Codex packages.

## `run` Subcommand

Behavior:

- Require the same guardrails acknowledgement flag as `cmd/symphony`:
  `--i-understand-that-this-will-be-running-without-the-usual-guardrails`.
- Require `--only-issue <id-or-identifier>`.
- Accept optional `--port <n>` and `--timeout <duration>`.
- Default timeout is `10m`; `--timeout 0` waits until interrupted.
- Load the workflow through `config.NewStore`.
- Create the normal Linear reader, wrap it with a read-only item filter, and call `cli.StartRuntime`.
- Print the dashboard URL when available.
- On shutdown, print a compact final snapshot summary.

The filter matches already-normalized `domain.WorkItem` values by `ID` or `Identifier`:

- `ListCandidates` returns only matching candidates.
- `ListByStates` returns only matching terminal-cleanup candidates.
- `RefreshByIDs` delegates first, then filters returned items.

Filtering terminal cleanup is a smoke-safety tradeoff. It verifies the selected issue's runtime path; it is not a replacement for normal unfiltered daemon startup coverage.

## Tests

Required tests:

- `internal/tracker`
  - filter candidates by `ID` and `Identifier`
  - filter terminal-state reads
  - filter refresh results while preserving underlying reader errors
  - no match returns empty slices
- `cmd/symphony-verify`
  - parse `linear` and `run` arguments
  - reject missing/blank `run --only-issue`
  - reject `run` without acknowledgement
  - render bounded item summaries
  - run linear probe with fake settings/reader and no network
  - reject non-Linear settings without network
  - boundary test: `linear` probe code does not import runtime/Codex packages

Required verification:

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

## Documentation

Add `docs/verification-workflows.md` with copyable commands for:

- Stage 1: Linear probe
- Stage 2: controlled single-issue runtime smoke

The docs must warn that Stage 2 launches real Codex and may mutate the target workspace and Linear issue through the configured workflow and `linear_graphql`.

## Boundaries

In scope:

- Dedicated verification command
- Read-only Linear probe
- Single-issue runtime smoke wrapper
- Provider-neutral read filter
- Operator documentation

Out of scope:

- Disposable Linear project/issue creation
- Generic tracker writes
- Generic workpad abstraction
- Codex protocol changes
- Orchestrator scheduling changes
- Lark or new provider behavior
