## ADDED Requirements

### Requirement: Web dashboard is served at the root path
The system SHALL serve a compatibility web dashboard at `/`.

The dashboard MUST render from `observability.DashboardView`, and it MUST remain available when the current snapshot is empty.

The dashboard MUST include the `Symphony Observability` page title, the `Operations Dashboard` heading, live/offline status labels, metric cards for running items, retrying items, total tokens, and runtime, and sections for `Rate limits`, `Running sessions`, and `Retry queue`.

#### Scenario: Empty snapshot still renders
- **WHEN** the root path is requested and the current observability snapshot has no running or retrying items
- **THEN** the system returns the dashboard shell using the empty snapshot
- **AND** the request does not fail because no runtime items exist
- **AND** the response includes the dashboard title, heading, status labels, metric cards, and required sections

#### Scenario: Running row exposes compatibility affordances
- **WHEN** the root path is requested and the current observability snapshot has a running item
- **THEN** the running session row includes the item identifier
- **AND** the row includes a `Copy ID` control
- **AND** the row includes a `JSON details` link to `/api/v1/<issue_identifier>`

#### Scenario: Updated snapshot is reflected
- **WHEN** the observability projection changes between two requests to `/`
- **THEN** the second request reflects the updated projection

#### Scenario: Snapshot unavailable renders an error panel
- **WHEN** the dashboard cannot load the current snapshot
- **THEN** the root path still returns an HTML dashboard response
- **AND** the response includes a `Snapshot unavailable` panel with the error code or message

### Requirement: Required static assets are served
The system SHALL serve the static assets required by the web dashboard with stable lookup behavior.

The system MUST return the correct asset bytes for bundled assets, it MUST set an appropriate content type for each asset response, it MUST set `cache-control: public, max-age=31536000` for known assets, and it MUST return plain not found for unknown asset paths.

The bundled paths MUST include `/dashboard.css`, `/vendor/phoenix_html/phoenix_html.js`, `/vendor/phoenix/phoenix.js`, and `/vendor/phoenix_live_view/phoenix_live_view.js`.

#### Scenario: Known asset is available
- **WHEN** a request targets a bundled dashboard asset
- **THEN** the system returns that asset's bytes
- **AND** the response uses the asset's expected content type
- **AND** the response uses the long-lived dashboard asset cache header

#### Scenario: Unknown asset is rejected
- **WHEN** a request targets a path that is not part of the bundled dashboard assets
- **THEN** the system returns a not-found response

### Requirement: Web handler preserves API route behavior
The system SHALL delegate `/api/v1/*` requests to the existing HTTP API compatibility handler.

The web layer MUST NOT duplicate state DTOs, issue DTOs, refresh DTOs, or shared JSON error envelopes.

#### Scenario: API state route is delegated
- **WHEN** a request targets `/api/v1/state`
- **THEN** the response follows the existing HTTP API state contract

#### Scenario: Dashboard method mismatch uses JSON error
- **WHEN** a non-GET request targets `/`
- **THEN** the response is a JSON `method_not_allowed` error

#### Scenario: Unknown web route uses JSON not found
- **WHEN** a request targets an unknown non-asset path
- **THEN** the response is a JSON `not_found` error

### Requirement: Configured runtime mounts the web dashboard
The system SHALL start an HTTP server with the web dashboard handler when runtime settings include `server.port`.

The runtime wiring MUST mount `/`, bundled static assets, and delegated `/api/v1/*` routes without adding CLI flag parsing or startup acknowledgement behavior in T17.

#### Scenario: Server port exposes dashboard routes
- **WHEN** the runtime starts with a configured server port
- **THEN** `GET /` returns the web dashboard
- **AND** `GET /dashboard.css` returns the bundled stylesheet
- **AND** `GET /api/v1/state` returns the delegated API state payload

### Requirement: Web rendering remains projection-only
The system SHALL render the web dashboard from `observability.DashboardView` only.

The web layer MUST NOT own mutable runtime state, and the rendered output MUST be deterministic for a given snapshot and asset bundle.

#### Scenario: Same snapshot renders equivalent output
- **WHEN** the same observability snapshot is rendered twice without changing the asset bundle
- **THEN** the dashboard output is equivalent across both renders

#### Scenario: Rendering does not require runtime mutation
- **WHEN** the dashboard is rendered for a request
- **THEN** the web layer does not mutate orchestrator state or create a second runtime state source
