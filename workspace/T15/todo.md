# T15 Residual Notes

## Deferred By Design

- CLI/server lifecycle wiring for `server.port`, host binding, and shutdown belongs to T18.
- Web dashboard route `/` and static assets belong to T17.
- Terminal dashboard rendering and full Codex message humanization belong to T16.
- Live runtime adapters do not currently emit async snapshot timeout/unavailable errors. T15 preserves the HTTP envelopes through typed handler errors; T18 can decide how runtime wiring maps real timeout/unavailable conditions into those errors.

## Verification Notes

- `make test-e2e` currently exercises the repository e2e suite but has no T15-specific live listener case because T15 intentionally stops at `http.Handler` compatibility.
- Package-level handler tests cover route, DTO, nullability, snapshot error, refresh error, and dependency-boundary behavior.
