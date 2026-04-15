## Context

The main `cmd/symphony` binary already starts the full runtime: config store, tracker reader, terminal cleanup, orchestrator, workspace manager, workflow bundle, Codex app-server workers, and optional HTTP dashboard. That is the correct production path, but it is too broad for staged live validation.

T19 adds a separate verification edge. It should reuse existing runtime seams without changing normal daemon behavior.

## Goals / Non-Goals

**Goals:**

- Verify Linear read configuration without starting runtime workers or Codex.
- Verify a real Linear issue-to-Codex path with a single explicit issue filter.
- Keep verification behavior out of the production `cmd/symphony` CLI.
- Keep tracker writes and Linear workflow behavior in existing compatibility-shell packages.
- Make the default test suite credential-free.

**Non-Goals:**

- Do not create disposable Linear projects or issues.
- Do not add a generic tracker write interface.
- Do not change Codex app-server protocol behavior.
- Do not change orchestrator scheduling semantics.
- Do not add any new provider behavior.

## Decisions

### Add `cmd/symphony-verify`

Use a dedicated binary with explicit subcommands:

```text
symphony-verify linear [flags] [path-to-WORKFLOW.md]
symphony-verify run [flags] [path-to-WORKFLOW.md]
```

This keeps verification-only flags away from `cmd/symphony` and makes operator intent explicit.

### Make `linear` read-only and dependency-injected

The `linear` subcommand loads settings with `config.NewStore(...).CurrentSettings()` so validation matches runtime startup. It then creates the normal Linear reader and calls the existing read methods.

The command implementation should keep the probe execution behind injectable dependencies so tests can supply fake settings and fake readers without network access.

### Keep `run` a wrapper around the existing runtime

The `run` subcommand should not create a second runtime. It creates the normal Linear reader, wraps it with a read-only item filter, and passes that reader into `cli.StartRuntime`.

The command requires the guardrails acknowledgement flag and `--only-issue` before starting runtime. It may accept `--port` and `--timeout` as verification convenience options.

### Filter normalized work items, not provider payloads

The read filter belongs in `internal/tracker` because it works on `domain.WorkItem` and the read-only `TrackerReader` interface. It must not know about Linear-specific fields beyond the normalized `ID` and `Identifier`.

Filtering `ListByStates` during the smoke command scopes terminal cleanup to the selected item. This is a smoke-safety tradeoff, not a replacement for the normal unfiltered daemon path.

## Risks / Trade-offs

- [Risk] Verification flags could drift into production CLI semantics. -> Mitigation: keep all new operator helpers in `cmd/symphony-verify`.
- [Risk] Linear probe tests accidentally require real credentials. -> Mitigation: dependency-inject the probe runner and cover success with fake readers.
- [Risk] Single-issue smoke gives false confidence about full startup cleanup. -> Mitigation: document that scoped terminal cleanup is specific to the smoke path.
- [Risk] Runtime smoke wrapper becomes a second orchestrator. -> Mitigation: call `cli.StartRuntime` and only add edge-level timeout / reporting.
