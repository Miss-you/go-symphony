## Why

`go-symphony` now has an end-to-end runtime projection, but it still lacks the Symphony-compatible JSON API that downstream dashboards, operators, and tests expect. T15 closes that compatibility surface without introducing a second observability state owner.

## What Changes

- add a thin `internal/httpapi` handler that serves `GET /api/v1/state`, `POST /api/v1/refresh`, and `GET /api/v1/:issue_identifier`
- project `domain.Snapshot` into the existing Symphony JSON field names, including counts, running entries, retry entries, token totals, rate limits, issue detail, logs, recent events, and tracked placeholders
- preserve compatibility error envelopes and status codes for unknown issues, unknown routes, unsupported methods, snapshot timeout/unavailable state payloads, and refresh unavailability
- keep state and refresh inputs behind package-local function seams so the HTTP layer does not import orchestrator, tracker, CLI, or provider-specific packages
- add package-level handler tests that freeze DTO shapes, nullability, empty-array behavior, route precedence, and refresh/error semantics

## Capabilities

### New Capabilities

- `http-api-compatibility`: JSON HTTP API compatibility for Symphony runtime state, issue detail, refresh triggers, and API error envelopes.

### Modified Capabilities

- None

## Impact

- `internal/httpapi`
- `openspec/specs/http-api-compatibility/spec.md`
- `workspace/T15/`
