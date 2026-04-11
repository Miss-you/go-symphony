# T17 Code Review

## Initial Review

Reviewer reported one blocking issue and one medium issue:

- Blocking: `internal/web.NewHandler` existed but was not mounted by the shipped runtime, so configured users could not reach `/`, `/dashboard.css`, or `/api/v1/*`.
- Medium: static asset tests covered `/dashboard.css` and one vendor path but did not lock all required vendor asset content types and cache headers.

## Resolution

- Added minimal `internal/cli.StartRuntime` HTTP server lifecycle wiring when `server.port` is configured.
- Mounted `internal/web.NewHandler` with snapshot, refresh, workspace root, dashboard URL, max-agent context, and project URL options.
- Added `Runtime.DashboardURL()` for internal runtime tests.
- Added runtime coverage proving configured server routes serve `/`, `/dashboard.css`, and delegated `/api/v1/state`.
- Extended static asset tests to cover all required vendor asset paths and cache headers.
- Updated `final_impl.md`, OpenSpec artifacts, and `test_strategy.md` so server wiring is recorded as T17 scope while CLI flag parsing remains T18 scope.

## Verification After Fix

- `go test ./internal/web/... ./internal/cli/...`
- `openspec validate web-dashboard-static-assets`
- `go test ./...`
- `make build`
- `make lint`
- `make test-e2e`
- `make verify`

## Follow-up Review

Follow-up review found no remaining blocking issues. It confirmed:

- The configured runtime now starts and mounts the web handler when `server.port` is set.
- Runtime coverage reaches `/`, `/dashboard.css`, and `/api/v1/state` through `Runtime.DashboardURL()`.
- Static asset coverage now includes all required vendor paths.
- No new route compatibility, security, or boundary violations were reported.
