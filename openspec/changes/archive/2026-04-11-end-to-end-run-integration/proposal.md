## Why

`go-symphony` already has the core runtime pieces, but the repository still lacks the end-to-end assembly contract that proves they can start, run, refresh, retry, and shut down as one coherent Symphony-compatible loop. T14 closes that gap so later work can build on a verified runtime boundary instead of ad hoc wiring.

## What Changes

- add the runtime assembly contract for starting the Symphony loop from `config.Store`, the tracker reader, the workspace manager, the orchestrator, and the Codex session layer
- define the startup cleanup and shutdown cleanup behavior that keeps terminal workspaces from accumulating
- define the memory path as a no-network runtime bundle for local and test verification
- define Linear workflow-driven tool injection for the provider-backed path
- define post-turn refresh, `max_turns` normal completion, retry metadata, and event normalization for the worker loop
- define idempotent shutdown and config snapshot handling across the active runtime lifecycle

## Capabilities

### New Capabilities
- `end-to-end-run-integration`: end-to-end runtime assembly and loop behavior for Symphony-compatible runs, including startup cleanup, provider-neutral assembly, memory no-network execution, Linear tool injection, post-turn refresh, retry lineage, event normalization, and shutdown semantics

### Modified Capabilities
- None

## Impact

- `internal/cli`
- `internal/orchestrator`
- `internal/config`
- `internal/workspace`
- `internal/workflow`
- `internal/toolbridge/linear`
- `internal/runner`
- `internal/codex`
- `cmd/symphony`
- `workspace/T14/`
