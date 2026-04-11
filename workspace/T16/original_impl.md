# T16 Terminal Dashboard Compatibility - Elixir source findings

I inspected the Elixir implementation under `/Users/apple/Documents/Github/symphony` because the originally requested `/Users/lihui/Documents/GitHub/symphony` path is not present in this environment.

## What the terminal dashboard actually renders

The terminal UI lives in [`elixir/lib/symphony_elixir/status_dashboard.ex`](file:///Users/apple/Documents/Github/symphony/elixir/lib/symphony_elixir/status_dashboard.ex). The top-level frame is a terminal snapshot, not a TUI widget tree:

- `╭─ SYMPHONY STATUS`
- `│ Agents: <running>/<max>`
- `│ Throughput: <n> tps`
- `│ Runtime: <m>s`
- `│ Tokens: in ... | out ... | total ...`
- `│ Rate Limits: ...`
- `│ Project: <Linear project URL or n/a>`
- `│ Dashboard: <http://.../>` when the HTTP port is known
- `│ Next refresh: ...`
- `├─ Running`
- `├─ Backoff queue`
- `╰─`

The default render path clears the terminal and writes the whole frame from the top using `IO.ANSI.home()` and `IO.ANSI.clear()` in `render_to_terminal/1`.

## Terminal-specific rendering behavior

Relevant rendering helpers in [`elixir/lib/symphony_elixir/status_dashboard.ex`](file:///Users/apple/Documents/Github/symphony/elixir/lib/symphony_elixir/status_dashboard.ex):

- `format_snapshot_content/3` builds the entire terminal output.
- `render_now?/2`, `schedule_flush_render/2`, and `flush_delay_ms/2` coalesce updates so the dashboard does not repaint more often than `render_interval_ms`.
- `periodic_rerender_due?/2` forces a rerender at least once per second even if the snapshot fingerprint has not changed.
- `render_offline_status/0` emits a tiny offline frame with `app_status=offline`.
- `normalize_status_lines/1` is currently a no-op, so the rendered content is preserved as built.

Parity implication:

- The Go version needs the same coarse frame structure and the same refresh throttling semantics. The terminal output is not just a static string dump; it is rate-limited and coalesced.

## Event humanization

The dashboard does not print raw event payloads in the running table. It converts Codex events into user-facing phrases via `humanize_codex_message/1` and the helper chain under the same file.

Key humanization rules from [`elixir/lib/symphony_elixir/status_dashboard.ex`](file:///Users/apple/Documents/Github/symphony/elixir/lib/symphony_elixir/status_dashboard.ex):

- `turn/started` -> `turn started`
- `turn/completed` -> `turn completed (...)`
- `turn/failed` -> `turn failed: ...`
- `turn/cancelled` -> `turn cancelled`
- `turn/diff/updated` -> `turn diff updated`
- `turn/plan/updated` -> `plan updated`
- `thread/tokenUsage/updated` -> `thread token usage updated (...)`
- `item/started` and `item/completed` -> `item started/completed: <item type>`
- streaming events become phrases such as:
  - `agent message streaming`
  - `plan streaming`
  - `reasoning summary streaming`
  - `reasoning text streaming`
  - `command output streaming`
  - `file change output streaming`
- approval and input events become:
  - `command approval requested (...)`
  - `file change approval requested (...)`
  - `tool requires user input: ...`
  - `approval request auto-approved`
  - `tool input auto-answered`
- wrapper events are also humanized:
  - `codex/event/exec_command_begin` uses the shell command line itself as the status text
  - `codex/event/agent_message_delta` becomes `agent message streaming: ...`
  - `codex/event/agent_reasoning` becomes `reasoning update: ...`
  - `codex/event/token_count` becomes `token count update (...)`

The running row uses `summarize_message/1`, which delegates to `humanize_codex_message/1`, then truncates the final text to 140 characters.

Parity implication:

- The Go dashboard should preserve the same human-language vocabulary, especially for Codex event names and command/request strings. This is the primary user-visible compatibility surface for the `EVENT` column.

## Throughput display

The throughput line is rendered in the terminal header as `Throughput: <n> tps`.

Implementation details in [`elixir/lib/symphony_elixir/status_dashboard.ex`](file:///Users/apple/Documents/Github/symphony/elixir/lib/symphony_elixir/status_dashboard.ex):

- throughput is computed from a rolling 5-second window via `rolling_tps/3`
- updates are throttled to once per second via `throttled_tps/5`
- `format_tps/1` truncates to an integer and adds thousands separators
- there is also a 10-minute sparkline helper, `tps_graph/3`, with tests, but it is not part of the default terminal frame output in the current snapshots

Tests covering this behavior:

- [`elixir/test/symphony_elixir/orchestrator_status_test.exs`](file:///Users/apple/Documents/Github/symphony/elixir/test/symphony_elixir/orchestrator_status_test.exs) checks the 5-second rolling math, one-update-per-second throttling, and the 10-minute graph helper.

Parity implication:

- The Go implementation needs to preserve the same rolling math and the same one-second throttling behavior even if the visual treatment changes later. If the Go version adds a graph, it should be treated as an extension, not a replacement for the header throughput line.

## Rate-limit display

The terminal header prints a compact rate-limit line in [`elixir/lib/symphony_elixir/status_dashboard.ex`](file:///Users/apple/Documents/Github/symphony/elixir/lib/symphony_elixir/status_dashboard.ex):

- `Rate Limits: <limit_id> | primary ... | secondary ... | credits ...`

Formatting rules:

- `limit_id` falls back to `unknown`
- bucket summaries prefer `remaining/limit reset Ns`
- if only one side is available, the formatter still prints the bucket that exists
- `credits` can render as:
  - `credits unlimited`
  - `credits <balance>`
  - `credits available`
  - `credits none`
  - `credits n/a`
- the compact summary in `humanize_codex_method("account/rateLimits/updated", ...)` says `rate limits updated: primary ...; secondary ...`

Fixture coverage:

- [`elixir/test/fixtures/status_dashboard_snapshots/super_busy.snapshot.txt`](file:///Users/apple/Documents/Github/symphony/elixir/test/fixtures/status_dashboard_snapshots/super_busy.snapshot.txt)
- [`elixir/test/fixtures/status_dashboard_snapshots/backoff_queue.snapshot.txt`](file:///Users/apple/Documents/Github/symphony/elixir/test/fixtures/status_dashboard_snapshots/backoff_queue.snapshot.txt)
- [`elixir/test/fixtures/status_dashboard_snapshots/credits_unlimited.snapshot.txt`](file:///Users/apple/Documents/Github/symphony/elixir/test/fixtures/status_dashboard_snapshots/credits_unlimited.snapshot.txt)

Parity implication:

- The Go dashboard should keep the same compact, pipe-separated summary and the same `credits` variants. The tests show these strings are intentional compatibility output, not incidental debug text.

## Retry queue display

The retry section is rendered under `├─ Backoff queue` in [`elixir/lib/symphony_elixir/status_dashboard.ex`](file:///Users/apple/Documents/Github/symphony/elixir/lib/symphony_elixir/status_dashboard.ex).

Observed behavior:

- retries are sorted by `due_in_ms`
- there is no visible top-three cap in the current renderer; the snapshot fixture explicitly includes a fourth retry row to prove it renders
- each row is formatted as:
  - `↻ <identifier> attempt=<n> in <seconds.millis>s error=<sanitized text>`
- newline and CRLF sequences inside the error are normalized to spaces
- empty retry lists render `No queued retries`

Fixture coverage:

- [`elixir/test/fixtures/status_dashboard_snapshots/backoff_queue.snapshot.txt`](file:///Users/apple/Documents/Github/symphony/elixir/test/fixtures/status_dashboard_snapshots/backoff_queue.snapshot.txt)
- [`elixir/test/fixtures/status_dashboard_snapshots/backoff_queue.evidence.md`](file:///Users/apple/Documents/Github/symphony/elixir/test/fixtures/status_dashboard_snapshots/backoff_queue.evidence.md)

Parity implication:

- The Go implementation should not cap the visible retry list at three entries if it is trying to match current Elixir behavior. The current Elixir fixture proves the terminal dashboard intentionally shows more than three queued retries.

## Refresh and offline behavior

The terminal dashboard has two separate refresh-related behaviors in [`elixir/lib/symphony_elixir/status_dashboard.ex`](file:///Users/apple/Documents/Github/symphony/elixir/lib/symphony_elixir/status_dashboard.ex):

- a live `Next refresh:` line inside the frame
- an offline status frame emitted on application stop

The `Next refresh:` line can show:

- `checking now…`
- a countdown like `2s`
- `n/a`

The offline path is intentionally minimal:

- `render_offline_status/0` prints `╭─ SYMPHONY STATUS`, `│ app_status=offline`, and `╰─`
- it does not print the normal `Timestamp:` line

Tests covering this behavior:

- [`elixir/test/symphony_elixir/orchestrator_status_test.exs`](file:///Users/apple/Documents/Github/symphony/elixir/test/symphony_elixir/orchestrator_status_test.exs) has assertions for the countdown, the checking marker, the spacer line behavior, the empty closing corner, and offline rendering.
- [`elixir/lib/symphony_elixir.ex`](file:///Users/apple/Documents/Github/symphony/elixir/lib/symphony_elixir.ex) calls `StatusDashboard.render_offline_status/0` from `Application.stop/1`.

Parity implication:

- The Go terminal dashboard should preserve the distinct offline message path and the refresh-status line. Those are visible compatibility behaviors, not just internal state.

## User-visible sections and strings to preserve

From the current terminal renderer and fixtures, these user-facing labels are part of the compatibility surface:

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

The running table headers are:

- `ID`
- `STAGE`
- `PID`
- `AGE / TURN`
- `TOKENS`
- `SESSION`
- `EVENT`

Parity implication:

- The Go port should treat these strings as compatibility-facing labels. Changing them will alter the terminal snapshot contract and will likely break snapshot-style assertions.

## Fixtures and tests that matter most

Primary evidence files:

- [`elixir/test/symphony_elixir/status_dashboard_snapshot_test.exs`](file:///Users/apple/Documents/Github/symphony/elixir/test/symphony_elixir/status_dashboard_snapshot_test.exs)
- [`elixir/test/symphony_elixir/orchestrator_status_test.exs`](file:///Users/apple/Documents/Github/symphony/elixir/test/symphony_elixir/orchestrator_status_test.exs)
- [`elixir/test/fixtures/status_dashboard_snapshots/idle.snapshot.txt`](file:///Users/apple/Documents/Github/symphony/elixir/test/fixtures/status_dashboard_snapshots/idle.snapshot.txt)
- [`elixir/test/fixtures/status_dashboard_snapshots/idle_with_dashboard_url.snapshot.txt`](file:///Users/apple/Documents/Github/symphony/elixir/test/fixtures/status_dashboard_snapshots/idle_with_dashboard_url.snapshot.txt)
- [`elixir/test/fixtures/status_dashboard_snapshots/super_busy.snapshot.txt`](file:///Users/apple/Documents/Github/symphony/elixir/test/fixtures/status_dashboard_snapshots/super_busy.snapshot.txt)
- [`elixir/test/fixtures/status_dashboard_snapshots/backoff_queue.snapshot.txt`](file:///Users/apple/Documents/Github/symphony/elixir/test/fixtures/status_dashboard_snapshots/backoff_queue.snapshot.txt)
- [`elixir/test/fixtures/status_dashboard_snapshots/credits_unlimited.snapshot.txt`](file:///Users/apple/Documents/Github/symphony/elixir/test/fixtures/status_dashboard_snapshots/credits_unlimited.snapshot.txt)
- [`elixir/test/support/snapshot_support.exs`](file:///Users/apple/Documents/Github/symphony/elixir/test/support/snapshot_support.exs)

What the tests prove:

- the frame renders deterministically from snapshot data
- the dashboard URL is shown only when a server port is configured
- the retry section includes more than three rows when present
- event summaries are humanized, sanitized, and width-limited
- throughput math is stable
- refresh updates are coalesced
- the offline path is separate from the normal snapshot render

## Bottom line for Go parity

The Elixir terminal dashboard is a compact terminal snapshot renderer with a strong compatibility contract around:

1. the frame layout and labels
2. humanized Codex event text in the running table
3. the compact throughput and rate-limit summaries
4. the retry queue ordering and error sanitization
5. the `Next refresh` and offline behaviors
6. snapshot-driven tests and fixture files that lock all of the above in place

The Go implementation should match those visible behaviors first, then decide separately whether any additional dashboard polish belongs behind a compatibility-safe extension.
