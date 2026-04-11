## Why

`internal/workspace` currently carries command execution mechanics behind a private transport seam, while `internal/runner` is only a placeholder. T08 is needed now so local and SSH execution share one runtime contract before Codex protocol and full run integration build on top of it.

## What Changes

- Add a tested `internal/runner` execution-host capability for local command execution, SSH command construction/execution, command result normalization, and stateless worker-host selection.
- Refactor workspace lifecycle code so hook and host-addressed remote commands execute through the runner contract while workspace keeps create/reuse/remove policy.
- Wire orchestrator host admission through the runner host selector using host-load data derived from orchestrator-owned running state.
- Preserve the existing config surface; runner consumes `worker.ssh_hosts` and `worker.max_concurrent_agents_per_host` without adding new provider abstractions.
- Keep Codex app-server sessions and protocol events out of scope for this change.

## Capabilities

### New Capabilities

- `runtime-runner-execution-host`: Local/SSH command execution, command result normalization, SSH command construction, and stateless worker-host selection.

### Modified Capabilities

- `workspace-lifecycle`: Workspace lifecycle continues to own path, hook, and cleanup policy but delegates local/SSH command execution to runner.
- `runtime-orchestrator`: Orchestrator admission uses runner host selection from orchestrator-owned host-load snapshots while retaining ownership of mutable runtime state.

## Impact

- Adds real code and tests under `internal/runner`.
- Updates `internal/workspace` to depend on runner for command execution rather than owning shell transport.
- Updates `internal/orchestrator` host-admission wiring without changing tracker, Codex, dashboard, or workflow contracts.
- Adds OpenSpec coverage for the runner boundary and boundary-preserving changes to workspace and orchestrator specs.
