# T07 Code Review

## Findings

- High severity: `RemoveIssueWorkspaces` can recurse forever if `workerHosts` contains an empty string. The hostless branch at [internal/workspace/manager.go](/Users/lihui/Documents/GitHub/go-symphony/.worktrees/t07-workspace-lifecycle/internal/workspace/manager.go#L218) calls `RemoveIssueWorkspaces(identifier, host)` for every configured host, and a blank host re-enters the same hostless branch because `workerHost != ""` is false. `internal/config` does not currently reject blank `worker.ssh_hosts` entries either ([internal/config/settings.go](/Users/lihui/Documents/GitHub/go-symphony/.worktrees/t07-workspace-lifecycle/internal/config/settings.go#L120) and [internal/config/settings.go](/Users/lihui/Documents/GitHub/go-symphony/.worktrees/t07-workspace-lifecycle/internal/config/settings.go#L191)), so a malformed workflow can stack-overflow or hang terminal cleanup instead of failing cleanly.

- Medium severity: best-effort hook failures and timeouts are silently swallowed instead of being logged. `runHook` returns `nil` for best-effort paths on timeout or command failure without emitting any record at all ([internal/workspace/manager.go](/Users/lihui/Documents/GitHub/go-symphony/.worktrees/t07-workspace-lifecycle/internal/workspace/manager.go#L307) and [internal/workspace/manager.go](/Users/lihui/Documents/GitHub/go-symphony/.worktrees/t07-workspace-lifecycle/internal/workspace/manager.go#L318)), which diverges from the T07 contract that says `after_run` / `before_remove` failures are logged and ignored ([workspace/T07/final_impl.md](/Users/lihui/Documents/GitHub/go-symphony/.worktrees/t07-workspace-lifecycle/workspace/T07/final_impl.md#L129)). That makes cleanup and post-run failures invisible in production and is a regression in observability.

## Open Questions / Residual Risks

- The current implementation still treats remote transport as a private seam with only fake coverage in tests. That matches T07 scope, but it means any later T08 wiring needs to preserve the blank-host guard and the host-aware fan-out behavior explicitly.
- There is no dedicated test asserting that malformed `worker.ssh_hosts` input is rejected or normalized before `RemoveIssueWorkspaces` runs.

## Verdict

Not ready to accept as-is. The blank-host recursion is a blocking correctness bug, and the silent best-effort hook swallowing is a real observability regression that should be fixed before closing T07.
