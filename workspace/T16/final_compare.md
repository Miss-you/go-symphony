# T16 Final Comparison

## Task Goal

T16 targets the terminal dashboard compatibility surface: snapshot projection, ANSI frame rendering, live redraw cadence, Codex event humanization, and executable fixture provenance.

## Source Compatibility Check

- Elixir Symphony renders a `SYMPHONY STATUS` terminal frame with agent counts, throughput, runtime, token totals, rate-limit summaries, project/dashboard links, running rows, and backoff queue rows. The Go renderer preserves those user-visible sections and labels.
- The Go implementation keeps the dashboard presentation-only. `internal/observability` projects `domain.Snapshot`; `internal/dashboard` renders the view and gates redraws. Neither owns orchestrator state.
- The source dashboard coalesces rapid frame changes and still rerenders idle snapshots periodically. `RenderGate` now covers changed-frame coalescing, delayed flush, and stable-fingerprint idle rerender.
- Source fixture coverage is represented by copied fixture snapshots under `internal/dashboard/testdata/status_dashboard_snapshots/source/`, with explicit Go fixture provenance and executable drift checks.
- Codex event messages now use human-readable summaries for lifecycle, turn, token, tool, command, and fallback events where Go payload data exists.

## Intentional Differences

- Go does not expose Elixir's `codex_app_server_pid`; the terminal table keeps the `PID` column and uses available runtime host information or `n/a`.
- CLI startup/shutdown wiring is intentionally deferred to T18.
- Web dashboard behavior and assets are intentionally deferred to T17.
- Go-specific unavailable and offline frames are derived fixtures because they have no direct source fixture equivalent.

## Residual Risk

No unrecorded high-severity parity risk remains for T16. The remaining integration risk is already assigned to downstream tasks that wire the renderer into live CLI startup/shutdown and web surfaces.
