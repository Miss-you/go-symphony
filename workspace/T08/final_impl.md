# T08 Final Implementation

## Review Gate

`final_impl_v1.md` required one correction round.

Round one found two blockers:

- host admission could not be capacity-aware when the proposed API accepted only `preferredHost`
- runner was too broad because it absorbed workspace lifecycle CRUD policy

Round two passed:

- `review_1_round2.md`: 92 / 100, no high-severity issues
- `review_2_round2.md`: 88 / 100, no high-severity issues
- average: 90 / 100

The one medium round-two note is accepted here: T08 must wire `runner.HostSelection` into the orchestrator admission path instead of leaving it as a dormant helper.

## Goal

Implement the first real `internal/runner` boundary for local and SSH execution while preserving the T07 workspace lifecycle contract.

T08 makes local and SSH command execution share one Go contract. It must not move mutable runtime state out of the orchestrator, and it must not move workspace lifecycle policy out of `internal/workspace`.

## Source-Faithful Boundary

The Elixir implementation splits the responsibilities this way:

- `Orchestrator` selects a worker host from live running state and configured host capacity.
- `Workspace` owns path naming, create/reuse/remove semantics, and hook ordering.
- `SSH` only transports shell commands or ports to a remote host.
- `AgentRunner` and `Codex.AppServer` consume the selected host and workspace path but do not own workspace lifecycle policy.

The Go implementation should mirror that split:

- runner provides local/SSH execution primitives and deterministic host-selection helpers
- workspace decides what workspace operation is being performed
- orchestrator decides when a run is admitted, retried, stopped, or cleaned up
- Codex app-server lifecycle remains out of T08 and belongs to T09

## Package Responsibilities

### `internal/runner`

Own:

- one host-aware command execution contract
- a local executor for `host == ""`
- an SSH executor for non-empty hosts
- SSH host argument parsing and command construction
- optional `SYMPHONY_SSH_CONFIG` handling
- command output/status normalization
- stateless worker-host choice based on a caller-supplied occupancy snapshot

Do not own:

- mutable running-run state
- total orchestrator concurrency gates
- state-specific dispatch gates
- workspace path derivation or lifecycle policy
- Codex session lifecycle or app-server protocol events

Recommended API shape:

```go
type CommandRequest struct {
    Host    string
    Dir     string
    Command string
    Timeout time.Duration
}

type CommandResult struct {
    Output string
    Status int
}

type Executor interface {
    RunCommand(ctx context.Context, req CommandRequest) (CommandResult, error)
}

type HostLoad struct {
    Host    string
    Running int
}

type HostSelection struct {
    Hosts      []string
    MaxPerHost *int
}

func (s HostSelection) Select(preferred string, loads []HostLoad) (host string, admitted bool)
```

`HostSelection.Select` is pure. The orchestrator supplies `loads` from its private `running` map. If no SSH hosts are configured, it admits local execution with `host == ""`. If hosts are configured, it honors an eligible preferred host first, otherwise chooses the least-loaded eligible configured host with stable config-order tie-breaking. If every host is at `MaxPerHost`, it returns `admitted=false`.

Total `agent.max_concurrent_agents` and state-specific limits stay in `internal/orchestrator`; they are not runner concerns.

### `internal/workspace`

Keep:

- `PathForIdentifier`
- local path validation and symlink safety
- create/reuse/remove lifecycle policy
- hook ordering and fatal/best-effort decisions
- terminal cleanup fan-out over configured worker hosts
- remote workspace create/remove shell-script semantics as workspace lifecycle policy

Change:

- remove the current private command transport implementation from workspace
- inject or construct a `runner.Executor`
- use that executor for hook commands and host-addressed remote operations

Workspace may still build the remote lifecycle command because the command describes workspace policy. Runner only decides how that command is executed locally or via SSH.

### `internal/orchestrator`

Keep:

- all mutable runtime state
- claim/running/retry maps
- total concurrency and state-limit gates
- cleanup intent through `stopRunRequest.CleanupWorkspace`
- `startRunRequest.PreferredHost`, `startRunResult.WorkerHost`, and retry host lineage

Change:

- add a private host-load projection from orchestrator-owned `running` state
- adapt the existing `serviceDeps.admitRun(preferredHost)` path to call `runner.HostSelection.Select(preferredHost, loads)`
- store only the selected host/result facts back into orchestrator state

Do not let runner reach into orchestrator state. The data flow is one-way: orchestrator snapshots host loads, passes them to runner selection, then stores the selected host.

### `internal/config`

Do not expand the config model for T08.

Runner uses the existing normalized settings:

- `Worker.SSHHosts`
- `Worker.MaxConcurrentAgentsPerHost`

## Implementation Plan

### 1. Add runner tests first

Create `internal/runner/execution_host_test.go` with red tests for:

- local command execution runs in the requested directory and returns output/status
- local command timeout returns a distinguishable timeout error
- SSH command construction uses `ssh -T`, respects `SYMPHONY_SSH_CONFIG`, parses `host:port`, preserves `user@host:port`, and handles bracketed IPv6
- SSH execution wraps the remote command through `bash -lc`
- host selection admits local execution when no hosts are configured
- host selection honors an eligible preferred host
- host selection chooses the least-loaded eligible host with stable tie-breaking
- host selection rejects when all configured hosts are at the per-host cap
- host selection treats unknown preferred hosts as ineligible and falls back

These tests prove the boundary directly and avoid real SSH by injecting the process starter.

### 2. Implement runner executor and selector

Create:

- `internal/runner/execution_host.go`
- `internal/runner/local.go`
- `internal/runner/ssh.go`

Implementation details:

- use `exec.CommandContext` for local commands
- run local commands as `sh -lc <command>` with `Dir` set to the request directory when present
- normalize exit status into `CommandResult.Status`
- return `context.DeadlineExceeded` or a typed timeout error when the context times out
- build SSH argv separately from execution so tests can assert exact behavior without network access
- use `bash -lc <quoted-command>` on the remote side
- keep host parsing small and covered by tests instead of adding a broad SSH abstraction

### 3. Refactor workspace to depend on runner for command execution only

Modify `internal/workspace/manager.go` so:

- hook commands call `runner.Executor.RunCommand`
- local workspace create/remove still use local filesystem operations owned by workspace
- remote workspace create/remove still follow workspace lifecycle semantics, but execute their remote shell scripts through `runner.Executor`
- workspace no longer contains local shell process launching or SSH-specific command construction

Keep the existing public workspace behavior stable:

- `after_create` only runs on new create
- `before_run` is fatal
- `after_run` is best-effort and always attempted after workspace creation
- `before_remove` is best-effort
- hostless terminal cleanup fans out across configured worker hosts

### 4. Wire orchestrator admission through runner selection

Add a private helper that derives host loads from orchestrator-owned state, then use `runner.HostSelection` from the existing admission seam.

The shape should stay narrow:

```go
func (s *schedulerState) hostLoads() []runner.HostLoad {
    // derived only from s.running
}

func (s *schedulerState) selectHost(policy runner.HostSelection, preferred string) (string, bool) {
    return policy.Select(preferred, s.hostLoads())
}
```

This is not a long-lived runner pool. It is a pure selector used by the orchestrator at dispatch time.

### 5. Keep Codex out of scope

Do not implement:

- app-server process session handling
- `thread/start` or `turn/start`
- tool approval events
- Codex timeout/read-loop protocol details

T08 only provides the execution-host layer that T09 can use.

## Verification Strategy

Required gates:

1. `go test ./internal/runner/...`
   - proves the new local/SSH execution contract and stateless host selector
2. `go test ./internal/workspace/...`
   - proves the T07 lifecycle contract survived the executor refactor
3. `go test ./internal/orchestrator/...`
   - proves runtime state ownership and host facts still pass through the runner-backed admission path
4. `go test ./...`
   - proves repo-wide integration safety
5. `make build`
   - proves the canonical build still works
6. `make lint`
   - catches formatting and structural regressions
7. `make test-e2e`
   - run the standard command contract; if it remains a no-op/non-applicable at this stage, record that in `workspace/T08/todo.md`

## Non-Goals

- no Codex app-server protocol implementation
- no universal tracker write API
- no provider-agnostic workflow abstraction
- no long-lived runner scheduler or mutable host pool
- no moving workspace path policy into runner
- no real SSH network dependency in tests

## Risks And Mitigations

- Risk: runner accidentally becomes a second owner of runtime capacity.
  - Mitigation: `HostSelection.Select` is pure and receives occupancy data from orchestrator.

- Risk: workspace lifecycle policy is duplicated in runner.
  - Mitigation: runner exposes command execution only; workspace keeps create/remove/hook decisions.

- Risk: SSH quoting diverges from Elixir behavior.
  - Mitigation: cover argv construction, port parsing, config-file handling, and remote `bash -lc` wrapping in unit tests.

- Risk: overreaching into T09.
  - Mitigation: no app-server session or protocol types in `internal/runner`.

## Acceptance Criteria

T08 is complete when:

- `internal/runner` has a tested local/SSH command execution contract
- `internal/runner` has a tested stateless host selector that preserves Elixir per-host capacity behavior from caller-supplied host loads
- orchestrator admission uses the runner selector while retaining ownership of mutable running state
- `internal/workspace` no longer owns local shell launching or SSH command construction
- `internal/workspace` still owns workspace lifecycle policy
- verification gates pass or any non-applicable e2e limitation is explicitly recorded
