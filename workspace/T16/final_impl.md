# Terminal Dashboard Compatibility Implementation Plan

## Goal

Recreate the Elixir terminal dashboard as a Go compatibility surface over the existing runtime snapshot. T16 should preserve the user-visible terminal frame, event summaries, throughput, retry queue, rate-limit display, next-refresh text, offline frame, and fixture coverage without making the orchestrator or tracker own presentation state.

## Review Fixes Incorporated

This revision addresses the first review rejection.

- Live redraw semantics are in scope through a small `internal/dashboard.RenderGate`, tested independently from the pure renderer. It covers render coalescing, `render_interval_ms`, pending flush timing, and the Elixir-style once-per-second idle rerender.
- Fixture provenance is explicit and executable. Go snapshot fixtures should be seeded from the Elixir fixture set where possible, with unmodified source copies under `internal/dashboard/testdata/status_dashboard_snapshots/source/` and a provenance test that fails if a fixture lacks a source mapping or allowed adaptation.
- Offline and unavailable states are explicit view modes, not a free-form app-status string.
- Codex event humanization is split between runtime event summarization and renderer fallback formatting so the dashboard does not need raw orchestrator state.

## Output Contract

Preserve these literal labels and sections:

- `SYMPHONY STATUS`
- `Agents`
- `Throughput`
- `Runtime`
- `Tokens`
- `Rate Limits`
- `Project`
- `Dashboard`
- `Next refresh`
- `Running`
- `Backoff queue`
- `No active agents`
- `No queued retries`
- `Snapshot unavailable`
- `Orchestrator snapshot unavailable`
- `app_status=offline`

The running table headers remain:

- `ID`
- `STAGE`
- `PID`
- `AGE / TURN`
- `TOKENS`
- `SESSION`
- `EVENT`

Keep the `PID` label for compatibility. The current Go snapshot model does not expose a Codex app-server OS pid, so the value should use the stable runtime identifier available to the view, or `n/a` when none exists. Do not expand core state just to invent a pid field.

## Package Shape

### `internal/observability`

Add a projection-only dashboard view model:

- `type DashboardMode string`
  - `DashboardModeNormal`
  - `DashboardModeOffline`
  - `DashboardModeUnavailable`
- `type DashboardContext struct`
  - `Now time.Time`
  - `MaxAgents int`
  - `DashboardURL string`
  - `ProjectURL string`
- `type DashboardView struct`
  - header totals and rate-limit state
  - `Running []RunningRow`
  - `Retrying []RetryRow`
  - `ThroughputTPS int`
  - `NextRefresh string`
  - `Mode DashboardMode`
  - `UnavailableReason string`
- `type Projector struct`
  - owns only the bounded token samples needed for the 5-second TPS window and one-second TPS throttle
- `func NewProjector() *Projector`
- `func (p *Projector) Project(snapshot domain.Snapshot, ctx DashboardContext) DashboardView`
- `func (p *Projector) Offline() DashboardView`
- `func (p *Projector) Unavailable(reason string) DashboardView`
- `func SummarizeCodexEvent(event codex.Event) string`

`internal/observability` may import `internal/codex` for event summarization and `internal/domain` for snapshot projection. It must not import orchestrator-private packages or hold runtime truth.

### `internal/dashboard`

Add a pure ANSI renderer plus a small timing gate:

- `func Render(view observability.DashboardView) string`
- `func RenderOffline() string`
- `func RenderUnavailable(reason string) string`
- `type RenderGate struct`
- `func NewRenderGate(renderInterval time.Duration) *RenderGate`
- `func (g *RenderGate) Enqueue(content string, fingerprint string, now time.Time) RenderDecision`
- `func (g *RenderGate) Flush(now time.Time) (string, bool)`
- `func (g *RenderGate) ForceIdleRerender(fingerprint string, now time.Time) bool`

The renderer emits one deterministic string. It should prepend the ANSI home/clear sequence for live terminal output, then render the same frame shape as Symphony.

`RenderGate` may hold only terminal presentation timing state: last rendered content, last rendered timestamp, pending content, pending flush time, and last snapshot fingerprint. It must not read or mutate snapshots.

### `internal/cli`

Make only the narrow runtime-side change required for event text parity:

- Modify `emitCodexEvent` so `domain.RunEvent.Message` uses `observability.SummarizeCodexEvent(event)` when a better user-facing summary is available.
- Keep `RunEvent.Kind`, session ID, totals, and rate limits unchanged.
- Do not import `internal/dashboard` from `internal/cli`.
- Do not wire dashboard startup or shutdown into CLI in T16. T18 owns full CLI behavior.

## TDD Plan

### Task 1: Projector and Throughput Cache

Files:

- Create `internal/observability/dashboard.go`
- Create `internal/observability/dashboard_test.go`

Write failing tests first:

1. Project a `domain.Snapshot` into a stable `DashboardView`.
2. Preserve the snapshot's deterministic running and retry ordering.
3. Compute rolling throughput from the last 5 seconds of token totals.
4. Throttle throughput updates to at most once per second.
5. Derive `Next refresh` as `checking now...`, a countdown like `2s`, or `n/a`.
6. Carry `DashboardURL`, `ProjectURL`, and `MaxAgents` from context.
7. Return explicit offline and unavailable views without mutating the snapshot source.

Red command:

```bash
go test ./internal/observability/... -run TestProjector -v
```

Then implement the smallest projector and bounded sample cache needed to pass.

### Task 2: Terminal Frame and Snapshot Fixtures

Files:

- Create `internal/dashboard/renderer.go`
- Create `internal/dashboard/renderer_test.go`
- Create `internal/dashboard/testdata/status_dashboard_snapshots/idle.snapshot.txt`
- Create `internal/dashboard/testdata/status_dashboard_snapshots/idle_with_dashboard_url.snapshot.txt`
- Create `internal/dashboard/testdata/status_dashboard_snapshots/super_busy.snapshot.txt`
- Create `internal/dashboard/testdata/status_dashboard_snapshots/backoff_queue.snapshot.txt`
- Create `internal/dashboard/testdata/status_dashboard_snapshots/credits_unlimited.snapshot.txt`
- Create `internal/dashboard/testdata/status_dashboard_snapshots/snapshot_unavailable.snapshot.txt`
- Create `internal/dashboard/testdata/status_dashboard_snapshots/orchestrator_snapshot_unavailable.snapshot.txt`
- Create `internal/dashboard/testdata/status_dashboard_snapshots/offline.snapshot.txt`
- Create `internal/dashboard/testdata/status_dashboard_snapshots/source/idle.snapshot.txt`
- Create `internal/dashboard/testdata/status_dashboard_snapshots/source/idle_with_dashboard_url.snapshot.txt`
- Create `internal/dashboard/testdata/status_dashboard_snapshots/source/super_busy.snapshot.txt`
- Create `internal/dashboard/testdata/status_dashboard_snapshots/source/backoff_queue.snapshot.txt`
- Create `internal/dashboard/testdata/status_dashboard_snapshots/source/credits_unlimited.snapshot.txt`
- Create `internal/dashboard/testdata/status_dashboard_snapshots/provenance.json`
- Create `workspace/T16/fixture_provenance.md`

Write failing snapshot tests that compare exact output for:

1. Idle frame with no dashboard URL.
2. Idle frame with dashboard URL.
3. Busy frame with multiple running rows, throughput, and rate limits.
4. Backoff queue frame with at least four retry rows, proving no top-three cap.
5. Credits-unlimited rate-limit frame.
6. Snapshot-unavailable frame.
7. Orchestrator-snapshot-unavailable frame.
8. Minimal offline frame.

Fixture provenance rules:

- Copy the Elixir source fixtures named in `workspace/T16/original_impl.md` into `internal/dashboard/testdata/status_dashboard_snapshots/source/` without editing their content.
- Seed Go expected fixtures from those copied source fixtures where the Go snapshot model can express the same data.
- Record each Go fixture's source file and any deliberate adaptation in `internal/dashboard/testdata/status_dashboard_snapshots/provenance.json` and `workspace/T16/fixture_provenance.md`.
- Keep adaptations small and explicit. Example: Go lacks `codex_app_server_pid`, so the `PID` column can render `n/a` or a stable runtime identifier.
- Add `TestFixtureProvenance` that reads `provenance.json`, verifies every copied source fixture exists, verifies every Go expected fixture has a source mapping or an explicit `derived` reason, and compares the normalized source/Go frame skeleton for mapped fixtures. The normalization should strip ANSI codes and allow only declared row-cell adaptations such as the `PID` column value.

Red command:

```bash
go test ./internal/dashboard/... -run TestRenderSnapshot -v
```

Then implement the pure renderer. The renderer must not fetch data itself.

Run the executable provenance check:

```bash
go test ./internal/dashboard/... -run TestFixtureProvenance -v
```

This gate is required before accepting the fixture set. A paper-only provenance note is not enough for T16.

### Task 3: Live Redraw Gate

Files:

- Create `internal/dashboard/live.go`
- Create `internal/dashboard/live_test.go`

Write failing tests for the Elixir live semantics:

1. First content renders immediately.
2. Identical content is suppressed.
3. Changed content before `render_interval_ms` is stored as pending.
4. Pending content flushes at the computed interval boundary.
5. A stable snapshot fingerprint still allows an idle rerender after one second.
6. The gate owns no business state and never mutates snapshot/view input.

Red command:

```bash
go test ./internal/dashboard/... -run TestRenderGate -v
```

Then implement the smallest timing gate needed to pass.

### Task 4: Event Humanization and Formatting Edges

Files:

- Create `internal/dashboard/humanize.go`
- Create `internal/dashboard/humanize_test.go`
- Modify `internal/dashboard/renderer.go`
- Modify `internal/observability/dashboard.go`
- Modify `internal/cli/runtime.go`
- Modify `internal/cli/runtime_test.go`

Write failing tests for compatibility strings:

1. `turn/started` -> `turn started`
2. `turn/completed` -> `turn completed (...)`
3. `turn/failed` -> `turn failed: ...`
4. `turn/cancelled` -> `turn cancelled`
5. `turn/diff/updated` -> `turn diff updated`
6. `turn/plan/updated` -> `plan updated`
7. `thread/tokenUsage/updated` -> `thread token usage updated (...)`
8. `item/started` and `item/completed` -> `item started/completed: <item type>`
9. Streaming, approval, and user-input wrapper events get readable phrases.
10. `codex/event/exec_command_begin` uses the command line when the payload has it.
11. `codex/event/agent_message_delta`, `codex/event/agent_reasoning`, and `codex/event/token_count` get the expected wrapper text.
12. Running-row event text truncates at 140 characters without corrupting the output.
13. Retry errors normalize CR/LF and escaped newline sequences to spaces.
14. Rate-limit credits render as `unlimited`, numeric balance, `available`, `none`, or `n/a`.
15. `cli.emitCodexEvent` stores the shared observability summary in `RunEvent.Message` for payloads where Go has enough Codex event data.

Red command:

```bash
go test ./internal/dashboard/... ./internal/observability/... ./internal/cli/... -run 'TestHumanize|TestFormat|TestWorkerEmitsHumanizedCodexEvent' -v
```

Then implement the smallest helper set needed to pass. Do not add a third-party terminal or snapshot library.

## Verification Gates

Run these before closing T16:

1. `go test ./internal/observability/... ./internal/dashboard/...`
2. `go test ./internal/cli/...`
3. `go test ./...`
4. `make build`
5. `make lint`
6. `make verify`
7. `openspec validate --type change terminal-dashboard-compatibility`
8. `openspec validate --specs`
9. `git diff --check`

The dashboard package verification must include both exact Go fixture checks and `TestFixtureProvenance`; otherwise the fixture strategy is not accepted.

`make test-e2e` is not required unless T16 unexpectedly touches live runtime/e2e surfaces. If skipped, record the reason in `workspace/T16/todo.md`.

## Non-Goals

- Do not change `internal/domain` or `internal/orchestrator` snapshot ownership.
- Do not add a live TUI, graph view, or dashboard polish beyond the current Elixir-compatible frame.
- Do not add a new provider-agnostic dashboard abstraction.
- Do not introduce a second business state machine under `observability`.
- Do not add third-party terminal rendering or snapshot testing dependencies.
- Do not wire dashboard startup/shutdown into CLI in T16 beyond event-message summarization.
- Do not change HTTP API or web dashboard surfaces.
- Do not invent a new literal pid field in core state just to satisfy the `PID` column label.
