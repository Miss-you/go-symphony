# T08 Test Strategy

## Goal

`T08 Runner / ExecutionHost` must prove that local and SSH command execution now live behind `internal/runner`, while `internal/workspace` keeps lifecycle policy and `internal/orchestrator` keeps mutable runtime state.

The tests should not just prove that code compiles. They must prove that the boundary is usable by later Codex and full-run integration without letting SSH details leak back into workspace or runtime state leak into runner.

## What Must Be Proven

1. `internal/runner` can execute local commands through the shared command contract, with working directory, output, exit status, and timeout behavior visible to callers.
2. `internal/runner` constructs SSH execution correctly without requiring real SSH in tests: `ssh -T`, optional `SYMPHONY_SSH_CONFIG`, host/port parsing, user-prefixed host handling, bracketed IPv6 handling, and remote `bash -lc` wrapping.
3. `internal/runner` exposes pure host selection that preserves Elixir-style per-host capacity behavior using caller-supplied host loads.
4. `internal/workspace` still enforces T07 lifecycle behavior, but hook and remote command execution goes through a runner executor instead of private local/SSH transport.
5. `internal/orchestrator` wires host admission through runner selection using host loads derived from orchestrator-owned running entries, while keeping total concurrency, state limits, claims, retries, and cleanup intent under orchestrator ownership.
6. T08 does not implement Codex app-server protocol behavior, tracker writes, or provider workflow behavior.

## Verification Matrix

### 1. Runner package contract

Command:

```bash
go test ./internal/runner/...
```

Why this matters:

- This is the task-board gate for T08.
- It proves local and SSH execution share one contract.
- It proves SSH behavior without depending on network access or a real worker host.
- It proves the host selector is deterministic and stateless.

Required test coverage:

- local command runs in the requested directory
- local command returns combined output and status for success and non-zero exits
- command timeout returns a distinguishable timeout error
- SSH argv includes `ssh -T`
- SSH argv includes `-F <config>` when `SYMPHONY_SSH_CONFIG` is set
- `host:port` maps to `-p <port> <host>`
- `user@host:port` preserves the user prefix
- bracketed IPv6 is not misparsed
- remote command is wrapped with `bash -lc`
- no configured SSH hosts admits local execution
- eligible preferred host wins
- fallback chooses least-loaded eligible host with stable config-order tie-breaking
- unknown preferred host falls back to eligible configured hosts
- all hosts at per-host capacity rejects admission

### 2. Workspace lifecycle regression

Command:

```bash
go test ./internal/workspace/...
```

Why this matters:

- T08 changes how workspace executes hook and remote commands, so T07 semantics must remain intact.
- The proof is that workspace remains policy owner while runner becomes command executor.

Required test coverage:

- existing T07 path, create/reuse/remove, hook, and cleanup tests still pass
- lifecycle hook commands are sent to a runner executor with workspace path, timeout, and worker host
- host-addressed remote create/remove paths delegate command execution to runner
- workspace tests do not rely on real SSH
- workspace does not construct SSH argv or launch local shells directly

### 3. Orchestrator admission integration

Command:

```bash
go test ./internal/orchestrator/...
```

Why this matters:

- Runner selection must be part of the live admission path, not only a tested helper.
- The orchestrator must remain the sole owner of mutable runtime state.

Required test coverage:

- host loads are derived from running entries
- preferred host lineage is passed to runner selection for retries
- runner capacity rejection prevents dispatch without mutating runner-owned state
- total concurrency and state limits remain orchestrator gates
- retry and cleanup ownership is unchanged

### 4. Repository-wide compile and regression sweep

Command:

```bash
go test ./...
```

Why this matters:

- Runner, workspace, and orchestrator are shared core packages.
- This catches compile-time boundary drift and package import mistakes across the module.

### 5. Canonical build and lint gates

Commands:

```bash
make build
make lint
```

Why this matters:

- `make build` proves the canonical CLI build path still works.
- `make lint` catches formatting, `go vet`, and static issues missed by focused tests.

### 6. E2E command-contract gate

Command:

```bash
make test-e2e
```

Why this matters:

- T08 does not yet wire a full Codex run path, so e2e is not the primary proof of runner correctness.
- Running the command still proves the repository command contract remains healthy.
- If it is still a no-op or not behaviorally meaningful at this stage, record that limitation in `workspace/T08/todo.md` and the task-board notes before closure.

## Acceptance Threshold

T08 can leave verification only if:

- `go test ./internal/runner/...` passes and proves the new runner contract directly
- workspace and orchestrator package tests pass after the boundary refactor
- `go test ./...`, `make build`, and `make lint` pass
- `make test-e2e` passes or its current applicability limitation is recorded
- review finds no unhandled bug/regression-level issue
