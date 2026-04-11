# T17 Final Comparison

## Task Target

T17 requires web observability at `/` and served static assets while preserving the approved architecture:

- Web/dashboard behavior stays in the compatibility shell.
- Runtime truth remains in the orchestrator snapshot.
- Observability remains projection-only.
- Existing `/api/v1/*` API behavior is reused rather than duplicated.

## Comparison To Original Symphony

Original Symphony behavior:

- Serves the dashboard at `GET /`.
- Serves static assets at `/dashboard.css` and Phoenix vendor asset paths.
- Shows `Symphony Observability`, `Operations Dashboard`, live/offline status labels, metric cards, rate limits, running sessions, retry queue, `Copy ID`, and `JSON details` links.
- Renders an unavailable panel when snapshot loading fails.
- Starts the web endpoint only when a server port is configured.

Go implementation:

- `internal/web.NewHandler` serves `GET /`, required static assets, delegated `/api/v1/*`, JSON 405 for non-GET `/`, JSON 404 for unknown web routes, and plain `404 Not Found` for unknown asset paths.
- HTML renders from `observability.DashboardView`, not from web-owned runtime state.
- Static assets are embedded package-local files with stable content types and `cache-control: public, max-age=31536000`.
- `internal/cli.StartRuntime` starts the web server when `server.port` is configured and exposes the mounted dashboard URL for runtime tests.

## Intentional Adaptations

- Phoenix LiveView sockets are not recreated. The Go dashboard preserves the user-visible web surface and route/static/API contracts with server-rendered HTML.
- Vendor Phoenix asset paths are kept as compatibility paths with local placeholder JavaScript because the Go dashboard does not need Phoenix runtime code.
- CLI `--port`, startup acknowledgement copy, and shutdown rendering remain T18 scope.

## Residual Risk

- Visual parity is text/route/asset-contract based, not pixel-perfect. This matches the available original test evidence, which does not define a pixel contract.
- The Go dashboard does not yet implement browser live polling or socket updates. It reflects updated snapshots on later requests; richer browser refresh behavior can be revisited in T18 or parity hardening if needed.
