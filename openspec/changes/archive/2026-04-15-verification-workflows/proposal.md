## Why

Operators currently have to start the full Symphony runtime to learn whether Linear reads are configured correctly, and they have no guarded command for a single-issue live Codex smoke run. That mixes two failure domains: Linear data fetching and issue-to-Codex execution.

## What Changes

- Add a dedicated `cmd/symphony-verify` binary with `linear` and `run` subcommands.
- Add a read-only Linear probe that loads the configured workflow, calls the existing Linear reader, and prints candidate / terminal / refresh summaries without starting runtime workers.
- Add a guarded single-issue runtime smoke wrapper that launches the existing runtime against one explicit issue identifier.
- Add a provider-neutral read filter for normalized `domain.WorkItem` values.
- Add operator documentation for the two-stage verification flow.

## Capabilities

### New Capabilities

- `verification-workflows`: Operator-facing verification workflows for Linear reads and controlled runtime/Codex smoke runs.

### Modified Capabilities

- None.

## Impact

- New command package under `cmd/symphony-verify`.
- Small read-only helper in `internal/tracker`.
- Documentation under `docs/`.
- Tests for parser behavior, probe rendering, filter behavior, and verification-command boundaries.
