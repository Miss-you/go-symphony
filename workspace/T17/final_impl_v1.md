# T17 Implementation Approach: Web Dashboard + Static Assets

## Goal

Recreate Symphony's user-visible `/` observability surface in Go, with the same compatibility intent as the original Elixir app:

- `/` is the dashboard, not a landing page.
- The page is rendered from the existing observability snapshot.
- Static assets are served from stable routes without an external build step.
- The web layer stays projection-only and does not own runtime state.

## Route Behavior

Mount a narrow `internal/web` handler in front of the existing HTTP API handler.

- `GET /` returns HTML for the dashboard shell.
- Non-`GET` requests to `/` return a JSON 405 response through the same compatibility error shape used elsewhere.
- `/api/v1/*` delegates to `internal/httpapi.NewHandler(...)`.
- Unknown web routes return JSON 404, not an HTML fallback.
- Static assets are routed explicitly and do not depend on runtime filesystem access.

Target user-visible content on `/`:

- Page title: `Symphony Observability`.
- Heading: `Operations Dashboard`.
- Live/offline status affordances remain visible in the page shell. The Go implementation does not need Phoenix LiveView transport, but it must keep a user-visible runtime status area rather than dropping the concept.
- Metric cards remain present for running items, retrying items, total tokens, and runtime.
- Sections: `Rate limits`, `Running sessions`, `Retry queue`.
- Running rows include `Copy ID` and `JSON details` linking to `/api/v1/<issue_identifier>`.
- Snapshot unavailable state renders a clear unavailable panel instead of failing the request.

## Static Assets

Serve package-local embedded assets under fixed paths, matching the original compatibility surface:

- `/dashboard.css`
- `/vendor/phoenix_html/phoenix_html.js`
- `/vendor/phoenix/phoenix.js`
- `/vendor/phoenix_live_view/phoenix_live_view.js`

Behavioral requirements:

- Content types are stable and explicit.
- Cache headers use `public, max-age=31536000`.
- Missing assets return plain `404 Not Found`.
- There is no asset pipeline or runtime asset lookup outside the embedded bundle.

The CSS should provide the page layout and typography expected by the original dashboard. Browser JavaScript stays minimal and local, limited to copy-button behavior and optional refresh behavior.

## Data Flow

Use the existing observability projection path and avoid a second dashboard model.

```text
domain.Snapshot
    -> observability.Projector.Project(...)
    -> observability.DashboardView
    -> web HTML renderer
```

Recommended web handler shape:

- Accept a `SnapshotFunc`.
- Optionally accept a `RefreshFunc`.
- Accept presentation options such as `DashboardURL`, `ProjectURL`, `MaxAgents`, and `Now`.
- Reuse `observability.Projector` for HTML projection and render from `observability.DashboardView`.
- Reuse `internal/httpapi.NewHandler(...)` for `/api/v1/*`; do not duplicate API DTO or error behavior in `internal/web`.

Request flow:

1. `GET /` loads the current snapshot.
2. The projector converts it into a dashboard view.
3. The HTML renderer emits the page shell and sections.
4. The page links back into `/api/v1/<issue_identifier>` for JSON detail access.
5. If the snapshot cannot be loaded, render the unavailable state instead of failing the request.

The web layer is read-only. It may call snapshot and refresh functions but must not mutate orchestration state directly.

## Tests

Gate T17 with focused `internal/web` tests first.

Required coverage:

- Exact package gate: `go test ./internal/web/...`.
- `GET /` returns HTML with the expected dashboard title and headings.
- `POST /` returns the compatibility 405 JSON error.
- `/api/v1/*` still behaves as the existing HTTP API handler defines.
- Static asset routes return the right content type and cache headers.
- Missing assets return `404 Not Found`.
- Unknown non-asset web routes return the shared JSON 404 shape.
- Snapshot failure renders the unavailable/error panel.
- Metric cards render running, retrying, total tokens, and runtime values from the projected snapshot.
- Running rows include `Copy ID` and `JSON details` links to `/api/v1/<issue_identifier>`.

Useful assertion areas:

- Route handling for `/`, `/api/v1/state`, `/api/v1/refresh`, and `/api/v1/:issue_identifier`.
- HTML text presence for `Operations Dashboard`, `Rate limits`, `Running sessions`, and `Retry queue`.
- HTML text presence for live/offline status labels and the four metric labels.
- Link targets and asset paths.
- Header stability for CSS and vendor assets, including exact content type and `cache-control: public, max-age=31536000`.

## OpenSpec Scope

Keep the change scoped to T17 and the compatibility shell.

- Work inside `internal/web` and its tests.
- Reuse `internal/httpapi` and `internal/observability`.
- Do not change provider-neutral core packages unless a concrete type gap appears.
- Do not introduce universal tracker writes, workflow abstractions, or new core state ownership.
- Do not expand this task into terminal dashboard, orchestrator, or API behavior changes.

If the web layer reveals a mismatch with the original `/` behavior, capture it as a T17-only compatibility decision rather than widening the core design.

## Risks

- LiveView parity is not literal; the Go version needs the same user-visible outcome, not the same transport.
- Browser interaction is limited without Phoenix sockets, so refresh behavior must stay simple and predictable.
- Asset parity can drift if content types or cache headers are left implicit.
- The dashboard must remain projection-only, so moving state into `internal/web` would be a design regression.
- Snapshot unavailability needs explicit rendering coverage, or the page may fail open instead of showing the expected error panel.
