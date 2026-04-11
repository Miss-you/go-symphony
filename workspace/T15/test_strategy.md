# T15 Test Strategy

## What Must Be Proven

T15 is complete only if the HTTP API handler proves the compatibility surface for:

- `GET /api/v1/state`
- `POST /api/v1/refresh`
- `GET /api/v1/:issue_identifier`
- compatibility JSON error envelopes
- projection-only package boundaries

The tests should prove behavior from the `http.Handler` boundary, not by calling private DTO helpers directly.

## Package Tests

Primary command:

```bash
go test ./internal/httpapi/...
```

This proves:

- route precedence: fixed routes win before issue lookup
- method handling: unsupported methods return `405 method_not_allowed`
- unknown paths return `404 not_found`
- state DTO includes counts, running/retrying arrays, token totals, rate limits, and deterministic `generated_at`
- empty `running`, `retrying`, `logs.codex_session_logs`, and `recent_events` encode as `[]`, not `null`
- absent timestamps encode as `null`
- snapshot timeout and unavailable sentinels return HTTP `200` with the exact compatibility error body
- issue detail returns running-only, retry-only, both-present, missing issue, attempts, workspace path, logs, tracked, and recent-event inference correctly
- refresh success returns HTTP `202` with queued/coalesced/requested_at/operations
- refresh unavailable returns HTTP `503 orchestrator_unavailable`

## Boundary Tests

Add a package-level dependency guard in `internal/httpapi` tests or implementation review to prove `internal/httpapi` does not import:

- `internal/orchestrator`
- `internal/cli`
- `internal/tracker`
- provider-specific packages such as `internal/trackers/linear`

This matters because T15 is a compatibility projection, not a runtime owner.

## Broader Closure Checks

Run before marking the task done:

```bash
go test ./...
make build
make lint
make test-e2e
make verify
openspec validate --type change http-api-compatibility
openspec validate --specs
git diff --check
```

These checks prove the new package compiles with the full repo, lint rules still pass, OpenSpec artifacts are valid, and no whitespace diff issues were introduced.

## E2E Applicability

T15 does not start a real HTTP listener from CLI config. Live server and CLI `--port` behavior belong to T18, with web dashboard composition in T17. If `make test-e2e` has no T15-specific live listener case, record that as an applicability note rather than expanding this task.
