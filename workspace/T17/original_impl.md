# T17 Original Symphony Web Dashboard Research

## Source

The skill references `/Users/lihui/Documents/GitHub/symphony`, but that path is not present on this machine. The equivalent local checkout used for research is `/Users/apple/Documents/Github/symphony`.

Inspected files:

- `elixir/lib/symphony_elixir_web/router.ex`
- `elixir/lib/symphony_elixir_web/live/dashboard_live.ex`
- `elixir/lib/symphony_elixir_web/components/layouts.ex`
- `elixir/lib/symphony_elixir_web/controllers/observability_api_controller.ex`
- `elixir/lib/symphony_elixir_web/controllers/static_asset_controller.ex`
- `elixir/lib/symphony_elixir_web/static_assets.ex`
- `elixir/lib/symphony_elixir_web/endpoint.ex`
- `elixir/lib/symphony_elixir_web/presenter.ex`
- `elixir/lib/symphony_elixir_web/observability_pubsub.ex`
- `elixir/lib/symphony_elixir/http_server.ex`
- `elixir/lib/symphony_elixir/status_dashboard.ex`
- `elixir/priv/static/dashboard.css`
- `elixir/test/symphony_elixir/live_e2e_test.exs`
- `elixir/test/symphony_elixir/status_dashboard_snapshot_test.exs`
- `elixir/test/fixtures/status_dashboard_snapshots/*`

## `/` Route Behavior

- `GET /` serves a Phoenix LiveView dashboard through `DashboardLive`.
- The root layout sets page title `Symphony Observability`, includes CSRF metadata, links `/dashboard.css`, loads Phoenix/LiveView vendor scripts, and connects a LiveView socket at `/live`.
- The dashboard is the first screen. It is an operations surface, not a landing page.
- `POST /` is not a dashboard route and resolves to a 405 JSON error through the shared route/error behavior.
- The HTTP endpoint is optional. `HttpServer.start_link/1` returns `:ignore` when no port is configured, so web serving depends on the configured server port or CLI `--port`.

## Static Assets

- Static asset routes are explicit:
  - `/dashboard.css`
  - `/vendor/phoenix_html/phoenix_html.js`
  - `/vendor/phoenix/phoenix.js`
  - `/vendor/phoenix_live_view/phoenix_live_view.js`
- `StaticAssets.fetch/1` embeds `priv/static/dashboard.css` plus Phoenix dependency assets at compile time.
- Static assets carry fixed content types and `cache-control: public, max-age=31536000`.
- Missing assets return plain 404 `Not Found`.
- No separate JavaScript app bundle or asset build pipeline was found in the inspected slice.

## Data Flow

- `DashboardLive` calls `Presenter.state_payload(orchestrator, snapshot_timeout_ms)` on mount and after `ObservabilityPubSub` broadcasts `:observability_updated`.
- Browser updates are LiveView-driven rather than custom fetch polling.
- The page links to JSON issue details at `/api/v1/<issue_identifier>`.
- JSON API compatibility is:
  - `GET /api/v1/state`
  - `POST /api/v1/refresh`
  - `GET /api/v1/:issue_identifier`
- Error envelopes use `{error: %{code, message}}`.

## User-Visible Parity Targets

- Preserve the `Symphony Observability` page title.
- Preserve the `Operations Dashboard` heading and concise operational description.
- Preserve live/offline status affordances, adapted for Go without Phoenix LiveView.
- Preserve metric cards for running items, retry queue, total tokens, and runtime.
- Preserve `Rate limits`, `Running sessions`, and `Retry queue` sections.
- Running rows should include `Copy ID` and a `JSON details` link to `/api/v1/<issue_identifier>`.
- The page should render an unavailable/error panel when the snapshot cannot be loaded.
- Static assets must be served by stable routes and not depend on an external asset build step.
- The dashboard must remain projection-only and compatible with the existing JSON API shapes.

## Uncertainties

- Original web tests appear to assert route/text behavior, not pixel-perfect rendering.
- Terminal dashboard fixtures inform snapshot shape but do not fully constrain browser visuals.
- No browser-side API calls beyond LiveView transport and `/api/v1/*` links were found.
