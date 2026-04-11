## Why

T17 needs the browser-facing compatibility surface at `/` and the assets it depends on before the Go port can claim web parity. The runtime projection work already exists in the approved design; this change captures the web dashboard contract so implementation can stay aligned with the shared observability model.

## What Changes

- Add a compatibility-faithful web dashboard route at `/` with the `Symphony Observability` title, `Operations Dashboard` heading, live/offline status affordances, metric cards, and the `Rate limits`, `Running sessions`, and `Retry queue` sections.
- Serve the static assets required by the dashboard as part of the web compatibility surface, including `/dashboard.css` and the Phoenix-compatible vendor asset paths.
- Render the dashboard from `observability.DashboardView` rather than a second runtime state machine.
- Delegate `/api/v1/*` to the existing HTTP API handler so web routing does not duplicate API DTOs or errors.
- Start the mounted web handler from `internal/cli.StartRuntime` when `server.port` is configured, leaving CLI flags and startup copy for T18.
- Preserve empty-state rendering, unavailable-snapshot rendering, missing-asset handling, cache headers, and stable asset lookup behavior.

## Capabilities

### New Capabilities
- `web-dashboard-static-assets`: Web dashboard rendering at `/`, shared projection-driven presentation, and required static asset serving for compatibility.

### Modified Capabilities
- None

## Impact

- Affects `internal/web` routing, rendering, and asset serving.
- Affects `internal/cli` only for minimal configured HTTP server lifecycle wiring.
- Affects observability presentation wiring through the shared presenter model.
- Adds browser-facing compatibility tests and asset fixtures.
- Establishes the contract that T17 implementation work will follow without changing core runtime ownership.
