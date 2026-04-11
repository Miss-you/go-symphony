# T16 Current Go Implementation Research

This note captures the current Go worktree state for terminal dashboard compatibility. It is exploratory only: no code changes are described here, and the goal is to pin down what T16 can reuse versus what still needs a real dashboard layer.

## Files Inspected

- [internal/domain/types.go](/Users/apple/Documents/Github/go-symphony/.worktrees/t16-terminal-dashboard-compatibility/internal/domain/types.go)
- [internal/orchestrator/state.go](/Users/apple/Documents/Github/go-symphony/.worktrees/t16-terminal-dashboard-compatibility/internal/orchestrator/state.go)
- [internal/orchestrator/public.go](/Users/apple/Documents/Github/go-symphony/.worktrees/t16-terminal-dashboard-compatibility/internal/orchestrator/public.go)
- [internal/orchestrator/service_test.go](/Users/apple/Documents/Github/go-symphony/.worktrees/t16-terminal-dashboard-compatibility/internal/orchestrator/service_test.go)
- [internal/orchestrator/public_test.go](/Users/apple/Documents/Github/go-symphony/.worktrees/t16-terminal-dashboard-compatibility/internal/orchestrator/public_test.go)
- [internal/cli/runtime.go](/Users/apple/Documents/Github/go-symphony/.worktrees/t16-terminal-dashboard-compatibility/internal/cli/runtime.go)
- [internal/cli/main.go](/Users/apple/Documents/Github/go-symphony/.worktrees/t16-terminal-dashboard-compatibility/internal/cli/main.go)
- [internal/cli/runtime_test.go](/Users/apple/Documents/Github/go-symphony/.worktrees/t16-terminal-dashboard-compatibility/internal/cli/runtime_test.go)
- [internal/httpapi/dto.go](/Users/apple/Documents/Github/go-symphony/.worktrees/t16-terminal-dashboard-compatibility/internal/httpapi/dto.go)
- [internal/httpapi/handler.go](/Users/apple/Documents/Github/go-symphony/.worktrees/t16-terminal-dashboard-compatibility/internal/httpapi/handler.go)
- [internal/dashboard/doc.go](/Users/apple/Documents/Github/go-symphony/.worktrees/t16-terminal-dashboard-compatibility/internal/dashboard/doc.go)
- [internal/observability/doc.go](/Users/apple/Documents/Github/go-symphony/.worktrees/t16-terminal-dashboard-compatibility/internal/observability/doc.go)
- [cmd/symphony/main.go](/Users/apple/Documents/Github/go-symphony/.worktrees/t16-terminal-dashboard-compatibility/cmd/symphony/main.go)
- [docs/plans/2026-04-10-go-symphony-design.md](/Users/apple/Documents/Github/go-symphony/.worktrees/t16-terminal-dashboard-compatibility/docs/plans/2026-04-10-go-symphony-design.md)
- [docs/plans/2026-04-10-go-symphony-design-task.md](/Users/apple/Documents/Github/go-symphony/.worktrees/t16-terminal-dashboard-compatibility/docs/plans/2026-04-10-go-symphony-design-task.md)
- [workspace/T15/final_compare.md](/Users/apple/Documents/Github/go-symphony/.worktrees/t16-terminal-dashboard-compatibility/workspace/T15/final_compare.md)
- [workspace/T15/todo.md](/Users/apple/Documents/Github/go-symphony/.worktrees/t16-terminal-dashboard-compatibility/workspace/T15/todo.md)

## What Already Exists

### Domain model

[internal/domain/types.go](/Users/apple/Documents/Github/go-symphony/.worktrees/t16-terminal-dashboard-compatibility/internal/domain/types.go) already contains the runtime vocabulary that a terminal dashboard can project without reaching into private scheduler state:

- `Snapshot` with `Running`, `Retrying`, `Polling`, `CodexTotals`, and `RateLimits`
- `ActiveRun` with item identity, state, worker host, workspace path, session ID, turn count, start time, last event metadata, and per-run Codex totals
- `RetryEntry` with item identity, attempt, due time, last error, worker host, and workspace path
- `PollingState` with checking state, next poll time, and interval
- `RunEventKind` plus the tagged `RunEvent` envelope that workers report back to the orchestrator
- `RateLimits`, `RateLimitBucket`, and `RateLimitCredits` for the rate-limit projection

That is already enough for a terminal dashboard to render throughput, queue state, rate-limit state, and event summaries without inventing new core types.

### Orchestrator snapshot projection

[internal/orchestrator/state.go](/Users/apple/Documents/Github/go-symphony/.worktrees/t16-terminal-dashboard-compatibility/internal/orchestrator/state.go) is the only place that owns mutable runtime truth. The important projection details for T16 are:

- `applyRunEvent` mutates running-state metadata from `domain.RunEvent`
- `RunEventTurnCompleted` increments turn count
- `RunEventCodexEventReceived` updates aggregate Codex totals through delta accounting
- `RunEventRunCompleted` and `RunEventRunFailed` move work into the retry queue
- `RunEventRetryScheduled` is metadata-only and does not rewrite retry bookkeeping
- `snapshot()` returns a sorted `domain.Snapshot`

The sort order is already deterministic and should be treated as part of compatibility:

- running entries sort by `ItemIdentifier`, then `ItemID`, then `StartedAt`
- retry entries sort by `DueAt`, then `ItemIdentifier`, then `ItemID`

Snapshot projection also copies `RateLimits` before returning it, which means a dashboard renderer can safely read the snapshot without worrying about later mutation.

### CLI/runtime integration points

[internal/cli/runtime.go](/Users/apple/Documents/Github/go-symphony/.worktrees/t16-terminal-dashboard-compatibility/internal/cli/runtime.go) currently exposes a very narrow facade:

- `StartRuntime(...)` wires reader, workspace, orchestrator, Codex worker manager, and event emission
- `Runtime.Snapshot()` returns `domain.Snapshot`
- `Runtime.Close()` shuts down runtime pieces

The CLI entrypoint is equally thin:

- [internal/cli/main.go](/Users/apple/Documents/Github/go-symphony/.worktrees/t16-terminal-dashboard-compatibility/internal/cli/main.go) starts the runtime and blocks on context cancellation
- [cmd/symphony/main.go](/Users/apple/Documents/Github/go-symphony/.worktrees/t16-terminal-dashboard-compatibility/cmd/symphony/main.go) only installs signal handling and delegates to `cli.Main`

There is no terminal dashboard wiring in either file yet. `internal/cli.Runtime` is therefore the natural seam for any later dashboard runner, but it currently only gives read access to the orchestrator snapshot.

### Nearby projection example

[internal/httpapi/dto.go](/Users/apple/Documents/Github/go-symphony/.worktrees/t16-terminal-dashboard-compatibility/internal/httpapi/dto.go) is a useful nearby pattern even though it belongs to T15, not T16. It shows the current project style for compatibility surfaces:

- consume `domain.Snapshot`
- project into a surface-specific DTO
- keep route/view logic out of orchestrator state
- preserve nil versus empty distinctions where compatibility depends on them

That projection style is the closest existing precedent for how a terminal dashboard presenter should behave.

## Terminal Dashboard Surface Today

[internal/dashboard/doc.go](/Users/apple/Documents/Github/go-symphony/.worktrees/t16-terminal-dashboard-compatibility/internal/dashboard/doc.go) is still just a package stub. There are no dashboard renderer files, no fixture files, and no test files in `internal/dashboard`.

[internal/observability/doc.go](/Users/apple/Documents/Github/go-symphony/.worktrees/t16-terminal-dashboard-compatibility/internal/observability/doc.go) is also only a stub. The repo therefore does not yet have a shared presenter layer that dashboard code can reuse.

The current task-board note for T16 and the design doc both say the same thing: terminal dashboard compatibility is intentionally deferred to this task, after T15 established the snapshot projection source.

- [docs/plans/2026-04-10-go-symphony-design.md](/Users/apple/Documents/Github/go-symphony/.worktrees/t16-terminal-dashboard-compatibility/docs/plans/2026-04-10-go-symphony-design.md) frames T16 around ANSI rendering, throughput, retry queue, rate limits, event humanization, and fixture testing
- [workspace/T15/final_compare.md](/Users/apple/Documents/Github/go-symphony/.worktrees/t16-terminal-dashboard-compatibility/workspace/T15/final_compare.md) and [workspace/T15/todo.md](/Users/apple/Documents/Github/go-symphony/.worktrees/t16-terminal-dashboard-compatibility/workspace/T15/todo.md) explicitly defer terminal dashboard rendering and full Codex message humanization to T16

## Nearby Test Patterns

There is no dashboard-specific test pattern yet, so the closest evidence comes from adjacent packages.

### Orchestrator tests

[internal/orchestrator/service_test.go](/Users/apple/Documents/Github/go-symphony/.worktrees/t16-terminal-dashboard-compatibility/internal/orchestrator/service_test.go) shows the shape of stable snapshot testing in this repo:

- fake clock and fake timer factories keep timing deterministic
- tests assert exact `Snapshot()` content rather than private state
- snapshot ordering is verified explicitly
- aggregate token totals and rate limits are checked from the projection, not from internal maps
- `TestRetryScheduledEventIsMetadataOnly` confirms `RunEventRetryScheduled` should not rewrite retry bookkeeping

That is the most relevant style reference for T16 because dashboard output will also need a deterministic projection source.

### CLI runtime tests

[internal/cli/runtime_test.go](/Users/apple/Documents/Github/go-symphony/.worktrees/t16-terminal-dashboard-compatibility/internal/cli/runtime_test.go) shows how runtime behavior is currently validated:

- tests wait for state changes with `waitFor`
- runtime snapshots are polled until the expected projection appears
- worker transport fixtures are used to drive Codex turn behavior
- prompt rendering is checked through the worker path, not through CLI presentation

For T16, that suggests dashboard tests should stay focused on rendering a supplied snapshot fixture, not on spinning the full runtime unless a test is explicitly about integration.

### HTTP API tests

[internal/httpapi/handler.go](/Users/apple/Documents/Github/go-symphony/.worktrees/t16-terminal-dashboard-compatibility/internal/httpapi/handler.go) and the T15 tests show another useful pattern: compatibility surfaces are tested as pure projections over a controlled runtime seam. The dashboard layer should follow the same rule.

## Concrete Constraints For T16

1. The dashboard must read from `domain.Snapshot`, not from orchestrator-private maps, worker handles, or CLI internals.
2. `internal/dashboard` should stay projection-only. It should not become a second runtime owner or a second observability state machine.
3. Event humanization should be derived from `domain.RunEventKind` plus the existing message fields already carried in `ActiveRun`, not from a new event vocabulary.
4. Throughput, retry queue, and rate-limit rendering should use the snapshot fields already present in `domain.ActiveRun`, `domain.RetryEntry`, `domain.CodexTotals`, and `domain.RateLimits`.
5. Stable ordering matters. If the renderer reorders entries, that should be deliberate and covered by fixture tests because the orchestrator snapshot already exposes deterministic ordering.
6. Nil versus empty should stay meaningful for optional snapshot pieces, especially `RateLimits` and the optional timestamp fields.
7. The existing `internal/cli.Runtime` seam is enough for read-only access, but there is no dashboard launch path yet. T16 will need to add a presentation layer rather than trying to piggyback on `cli.Main` as it stands today.
8. The repo currently has no terminal-dashboard fixtures, so the first real tests for T16 will likely need to establish the fixture format before the renderer can be locked down.

## Current Gap Summary

The Go worktree is already strong on runtime truth ownership and snapshot projection. The remaining T16 gap is the presentation layer itself:

- no `internal/dashboard` implementation
- no shared presenter package under `internal/observability`
- no terminal rendering tests or fixtures
- no CLI wiring that actually invokes a dashboard renderer

That means T16 is not blocked by missing core state. It is blocked by the absence of a compatibility renderer that consumes the already-frozen snapshot model and turns it into ANSI output with stable fixtures.
