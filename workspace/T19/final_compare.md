# T19 Final Comparison

## Requested Goal

Enable two staged validation flows:

1. Pull Linear data and inspect candidate/terminal/refresh behavior without starting Codex or runtime workers.
2. After selecting one task, launch the real runtime and Codex app-server path end to end for that one issue.

## Implemented Shape

- Added `cmd/symphony-verify linear` for a read-only Linear probe.
- Added `cmd/symphony-verify run` for a guarded single-issue runtime smoke command.
- Added `internal/tracker.NewFilteredReader` so the smoke path can scope runtime reads by normalized `WorkItem.ID` or `WorkItem.Identifier`.
- Added `internal/verify/linearprobe` so the pure Linear probe logic has a testable dependency boundary that excludes runtime, workspace, orchestrator, and Codex packages.
- Added `docs/verification-workflows.md` with copyable two-stage commands and a minimal workflow template.

## Compatibility Check

- Production `cmd/symphony` behavior is unchanged.
- Linear-specific construction remains in the compatibility shell.
- Core filtering operates on provider-neutral `domain.WorkItem` values and does not introduce tracker writes.
- The runtime smoke reuses `cli.StartRuntime`, so it exercises the existing orchestrator, workspace, runner, dashboard, and Codex app-server integration rather than a parallel execution path.

## Original Symphony Parity

Original Symphony does not expose a separate Linear probe command. It reads Linear through the normal workflow/runtime path and reaches Codex through `codex app-server`. T19 keeps that compatibility path intact while adding a Go-only verification helper around the same seams.

## Residual Risk

- Live Linear/Codex smoke was not executed automatically because it requires `LINEAR_API_KEY`, a disposable Linear issue, and explicit consent to launch real Codex.
- The smoke command filters terminal cleanup to the selected issue. This is intentional for live smoke safety and does not replace normal unfiltered daemon startup validation.
