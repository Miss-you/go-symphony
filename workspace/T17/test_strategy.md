# T17 Test Strategy

## Purpose

T17 has one job: prove that the Go web layer recreates Symphony’s `/` observability surface and its static asset routes without taking ownership of runtime state. This strategy maps each verification tier to the specific compatibility claim it proves.

## Proof Matrix

| Behavior or risk | Check | What it proves |
| --- | --- | --- |
| `internal/web` compiles as the compatibility shell entrypoint | `go test ./internal/web/...` | The package can stand up the dashboard handler, embed assets, and compose with `internal/httpapi` and `internal/observability` without pulling provider-specific logic into the web layer. |
| `GET /` serves the dashboard shell, not a landing page | `go test ./internal/web/...` | The handler returns HTML with the expected page title, `Operations Dashboard` heading, live/offline status area, metric cards, and the `Rate limits`, `Running sessions`, and `Retry queue` sections. |
| The page keeps the JSON detail links used by the original dashboard | `go test ./internal/web/...` | Running rows still link to `/api/v1/<issue_identifier>` and expose the `Copy ID` and `JSON details` affordances that the compatibility contract preserves. |
| Snapshot projection is reused instead of reintroduced in the web layer | `go test ./internal/web/...` | The page is rendered from the existing observability snapshot/view model, so the web package proves it is projecting state rather than storing or recomputing runtime truth. |
| Later requests reflect later snapshots | `go test ./internal/web/...` | A handler backed by a changing snapshot function shows updated counts or rows on the second request, proving the web layer does not cache stale runtime truth. |
| Snapshot unavailability is handled explicitly | `go test ./internal/web/...` | A failing snapshot function renders the unavailable/error panel instead of crashing the request path or returning a misleading empty dashboard. |
| `POST /` and unknown web routes stay on the compatibility error path | `go test ./internal/web/...` | Non-`GET` dashboard requests return the shared JSON 405 shape, and non-asset unknown routes return the shared JSON 404 shape instead of falling back to HTML. |
| Static asset routes remain stable and explicit | `go test ./internal/web/...` | `/dashboard.css` and the vendor asset routes are served from the embedded bundle with fixed content types, `cache-control: public, max-age=31536000`, and plain `404 Not Found` for missing assets. |
| `/api/v1/*` behavior is delegated, not duplicated | `go test ./internal/web/...` | The web layer forwards API routes to `internal/httpapi` and preserves the existing `/api/v1/state`, `/api/v1/refresh`, and `/api/v1/:issue_identifier` semantics instead of creating a second API implementation. |
| Configured runtime makes the route reachable | `go test ./internal/cli/...` | `StartRuntime` starts an HTTP server when `server.port` is configured and mounts `/`, `/dashboard.css`, and `/api/v1/state` without taking over T18 CLI flag or acknowledgement behavior. |
| The shared API and observability packages still compile and expose the expected surface | `go test ./internal/httpapi/... ./internal/observability/...` | The web layer’s dependencies remain healthy, and the dashboard code can keep reusing the existing DTO and projection contracts without widening them. |
| The whole repository still builds with the new web package wired in | `go test ./...` | The dashboard code compiles in context with the rest of the runtime, CLI, API, and terminal dashboard packages. This is the broad compile-and-link check for the tree. |
| The production binary still builds with embedded web assets | `make build` | The repo can produce the normal Go binary with the web dashboard and static assets included, which proves the asset embedding and handler wiring are build-safe. |
| Static analysis still accepts the web implementation | `make lint` | Formatting, vet, and repository lint gates stay green, which catches handler plumbing mistakes, missing imports, dead code, and other structural regressions that unit tests may not surface. |
| The repo-level end-to-end command still runs | `make test-e2e` | The web dashboard work does not break the top-level e2e harness, even though the web surface itself is proved more directly by the package tests above. |
| OpenSpec contract remains valid and apply-ready | `openspec validate web-dashboard-static-assets` | The change artifacts still satisfy the spec workflow after implementation and remain aligned with T17's documented scope. |

## Package-Scoped Verification

The first proof gate is package-local:

```bash
go test ./internal/web/...
```

This gate should answer the T17-specific questions directly:

1. Does `GET /` return the dashboard shell with the expected user-visible text?
2. Do non-`GET` requests and unknown routes stay on the shared JSON error path?
3. Are the static asset routes served with the correct content type and cache headers?
4. Do missing assets return plain `404 Not Found`?
5. Does the handler reuse the existing observability projection and render the unavailable panel when snapshot loading fails?
6. Do the `/api/v1/*` routes still behave exactly as the existing HTTP API handler defines?
7. Does a later request reflect a changed snapshot instead of stale cached output?

If any of those fail, the change is not ready for broader verification.

## Broader Gates

After the package gate passes, run:

```bash
go test ./internal/httpapi/... ./internal/observability/...
go test ./internal/cli/...
go test ./...
make build
make lint
make test-e2e
openspec validate web-dashboard-static-assets
```

These checks do not prove the `/` dashboard by themselves. They prove the web layer is still wired into the same projection and API contracts, the repository still compiles as a whole, the binary can still be built, static analysis still passes, and the repo-level e2e command still runs.

## Out Of Scope

This strategy does not try to prove:

- terminal dashboard rendering
- orchestrator scheduling behavior
- tracker or toolbridge behavior
- CLI startup or shutdown rendering
- pixel-perfect visual parity with Phoenix LiveView transport

Those concerns belong to later tasks. T17 only proves the web dashboard shell, the embedded static assets, the route compatibility behavior, and the reuse of the existing observability/API contracts.
