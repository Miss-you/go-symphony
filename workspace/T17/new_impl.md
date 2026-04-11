# T17 Current Go Implementation Research

## Current State

The Go worktree has the runtime projection pipeline in place, but `internal/web` is still a stub.

- `internal/web/doc.go` is a placeholder.
- `internal/httpapi/handler.go` and `internal/httpapi/dto.go` already serve JSON compatibility routes for state, refresh, and issue detail.
- `internal/observability/dashboard.go` projects `domain.Snapshot` into a shared dashboard view model.
- `internal/dashboard/renderer.go`, `internal/dashboard/humanize.go`, and `internal/dashboard/live.go` render the terminal dashboard from the observability view.
- `internal/cli/runtime.go` owns runtime assembly and snapshot access but does not yet mount a web handler.
- `cmd/symphony/main.go` delegates to `cli.Main`.
- Task `T17` is now claimed in this isolated worktree and is in `research`.

## Reusable APIs

T17 should reuse existing projection contracts instead of creating a second dashboard state model.

- `domain.Snapshot`, `domain.ActiveRun`, `domain.RetryEntry`, `domain.PollingState`, `domain.CodexTotals`, and `domain.RateLimits`.
- `observability.Projector`, `observability.DashboardView`, and `observability.DashboardContext`.
- `dashboard` helpers for formatting counts, runtimes, session IDs, and inline text if useful.
- `httpapi.NewHandler` and its function seams for state and refresh.
- `config.Settings.Observability` for dashboard enablement and display timing settings.

## Gaps

- No route currently serves `/`.
- No embedded static asset tree exists under `internal/web`.
- No browser-facing HTML shell exists.
- No web route composition exists for dashboard HTML, static assets, and existing `/api/v1/*` routes.
- No web-specific tests freeze route behavior, static content types, HTML content, or snapshot unavailable behavior.

## Boundary Constraints

- Keep T17 inside the compatibility shell: `internal/web`, with narrow route composition to `internal/httpapi`.
- Keep core packages provider-neutral and unchanged unless a proven type gap appears.
- `observability` remains projection-only. The web layer must not own business state.
- Do not introduce a provider-agnostic workflow or tracker write API.
- Static assets should be package-local and embedded with `embed`, not loaded from runtime filesystem paths.

## Go-Native Implementation Direction

- Add `internal/web.Handler` or `NewHandler` that accepts:
  - a `SnapshotFunc`
  - optional `RefreshFunc`
  - presentation options such as `ProjectURL`, `DashboardURL`, `MaxAgents`, and `Now`
- Serve:
  - `GET /` as HTML
  - static assets such as `/dashboard.css`
  - `/api/v1/*` by delegating to `internal/httpapi`
- Render HTML server-side from `observability.Projector`.
- Include small browser JavaScript only for copy-button behavior and optional refresh. Do not require a build pipeline.
- Keep asset cache headers compatible with Symphony's long-lived static assets.

## Verification Implications

- First gate: `go test ./internal/web/...`.
- Broader gates: `go test ./internal/httpapi/... ./internal/observability/...`, `go test ./...`, `make build`, `make lint`, `make verify`, and OpenSpec validation.
- Tests should prove:
  - `/` returns a dashboard shell with expected user-visible text.
  - Unsupported methods and unknown routes do not break API behavior.
  - Static assets are served with stable content types and cache headers.
  - Snapshot errors render an unavailable panel rather than crashing.
  - HTML uses API links compatible with `/api/v1/:issue_identifier`.
