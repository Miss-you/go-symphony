## 1. Web Route and Projection

- [x] 1.1 Add the root web dashboard route at `/` and render it from `observability.DashboardView`.
- [x] 1.2 Preserve the user-visible shell: title, heading, live/offline status labels, four metric cards, `Rate limits`, `Running sessions`, and `Retry queue`.
- [x] 1.3 Add running-row affordances for `Copy ID` and `JSON details` links to `/api/v1/<issue_identifier>`.
- [x] 1.4 Render an unavailable panel when the snapshot function fails.
- [x] 1.5 Keep the web rendering path projection-only so it consumes snapshot data without owning mutable runtime state.
- [x] 1.6 Add coverage for empty-snapshot rendering and snapshot updates reflected on subsequent requests.

## 2. Static Asset Serving

- [x] 2.1 Add the bundled static asset source for `/dashboard.css`, `/vendor/phoenix_html/phoenix_html.js`, `/vendor/phoenix/phoenix.js`, and `/vendor/phoenix_live_view/phoenix_live_view.js`.
- [x] 2.2 Serve known dashboard assets with expected content types, bytes, and `cache-control: public, max-age=31536000`.
- [x] 2.3 Return plain `404 Not Found` for unknown asset paths without falling back to the dashboard shell.

## 3. API Delegation and Compatibility Verification

- [x] 3.1 Delegate `/api/v1/*` to `internal/httpapi.NewHandler(...)` without duplicating API DTO or error behavior.
- [x] 3.2 Return JSON 405 for non-GET `/` requests and JSON 404 for unknown non-asset web routes.
- [x] 3.3 Add package tests that lock dashboard HTML content, unavailable rendering, API delegation, asset behavior, and route errors.
- [x] 3.4 Run `go test ./internal/web/...` and fix failures until the package gate passes.
- [x] 3.5 Run `openspec validate web-dashboard-static-assets` and confirm the change is apply-ready.

## 4. Runtime Server Wiring

- [x] 4.1 Start an HTTP server from `internal/cli.StartRuntime` when `server.port` is configured.
- [x] 4.2 Mount `/`, `/dashboard.css`, and `/api/v1/*` through the web handler without adding T18 CLI flag behavior.
- [x] 4.3 Add runtime coverage proving configured server routes are reachable.
