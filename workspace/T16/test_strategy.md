# T16 Test Strategy

## What Must Be Proven

T16 is complete only if the terminal dashboard is demonstrably a compatibility surface over `domain.Snapshot`, not an incidental debug view. Tests must prove:

- snapshot projection is read-only and deterministic
- terminal frame labels, sections, and fallbacks match the Symphony contract
- running rows, retry rows, tokens, rate limits, throughput, and next-refresh text render correctly
- live redraw cadence preserves coalescing, flush timing, and idle rerender behavior
- Codex event summaries use human-readable compatibility phrases when payload data is available
- dashboard fixtures are tied to copied source Symphony fixtures or explicitly derived

## Package Tests

Primary package gate:

```bash
go test ./internal/observability/... ./internal/dashboard/...
```

This proves:

- `Projector` maps `domain.Snapshot` into `DashboardView` without mutating source data
- rolling throughput uses the 5-second token window and one-second throttle
- `Next refresh` renders `checking now...`, countdown seconds, or `n/a`
- idle, dashboard URL, busy, backoff queue, credits-unlimited, unavailable, and offline fixtures match exactly
- retry rows are sorted by due time and not capped at three entries
- retry error text normalizes CR/LF and escaped newline sequences
- rate-limit buckets and credits variants render as expected
- `RenderGate` handles first render, duplicate suppression, pending frame coalescing, interval flush, and one-second idle rerender

## Fixture Provenance

Run:

```bash
go test ./internal/dashboard/... -run TestFixtureProvenance -v
```

This proves:

- unmodified Elixir fixture copies exist under `internal/dashboard/testdata/status_dashboard_snapshots/source/`
- every Go expected fixture has a `provenance.json` mapping or an explicit derived reason
- normalized source and Go frame skeletons match for mapped fixtures
- declared adaptations, such as the missing Go app-server pid field, are explicit rather than accidental

## Runtime Event Summary Test

Run:

```bash
go test ./internal/cli/... -run TestWorkerEmitsHumanizedCodexEvent -v
```

This proves the runtime emits user-facing Codex event summaries into `domain.RunEvent.Message` before the snapshot reaches terminal or API projections.

## Broader Closure Checks

Run before marking T16 done:

```bash
go test ./internal/observability/... ./internal/dashboard/...
go test ./internal/cli/...
go test ./...
make build
make lint
make verify
openspec validate --type change terminal-dashboard-compatibility
openspec validate --specs
git diff --check
```

These checks prove the package behavior, whole-repo compile/test health, lint/build gates, OpenSpec validity, and whitespace hygiene.

## E2E Applicability

T16 does not wire terminal dashboard startup/shutdown into CLI; that belongs to T18. `make test-e2e` is not required unless implementation unexpectedly touches live e2e surfaces. If skipped, record the reason in `workspace/T16/todo.md`.
