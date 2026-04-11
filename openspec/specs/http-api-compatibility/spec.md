## Purpose

Define the Symphony-compatible JSON HTTP API contract for projecting runtime snapshots, issue details, refresh triggers, and API error envelopes from `internal/httpapi`.

## Requirements

### Requirement: HTTP API handler exposes Symphony-compatible routes
The system SHALL provide an HTTP handler for the Symphony-compatible JSON API under `/api/v1`.

#### Scenario: State route accepts GET
- **WHEN** a client sends `GET /api/v1/state`
- **THEN** the handler responds with HTTP `200`
- **AND** the response body is the state payload projected from the current snapshot

#### Scenario: Refresh route accepts POST
- **WHEN** a client sends `POST /api/v1/refresh`
- **THEN** the handler invokes the configured refresh function
- **AND** the handler responds with HTTP `202` when refresh is accepted

#### Scenario: Issue detail route accepts GET
- **WHEN** a client sends `GET /api/v1/<issue_identifier>`
- **THEN** the handler looks up the identifier in the current running and retry snapshot entries
- **AND** returns HTTP `200` with an issue detail payload when either entry exists

#### Scenario: Unsupported methods on known routes are rejected
- **WHEN** a client sends an unsupported method to `/api/v1/state`, `/api/v1/refresh`, or `/api/v1/<issue_identifier>`
- **THEN** the handler responds with HTTP `405`
- **AND** the body is `{"error":{"code":"method_not_allowed","message":"Method not allowed"}}`

#### Scenario: Unknown routes are rejected
- **WHEN** a client sends a request for a path not owned by the T15 API handler
- **THEN** the handler responds with HTTP `404`
- **AND** the body is `{"error":{"code":"not_found","message":"Route not found"}}`

### Requirement: State payload preserves compatibility DTO fields
The system SHALL project `domain.Snapshot` into the Symphony-compatible `/api/v1/state` JSON shape.

#### Scenario: State payload includes counts and arrays
- **WHEN** the handler receives a successful snapshot
- **THEN** the response contains `generated_at`, `counts.running`, `counts.retrying`, `running`, `retrying`, `codex_totals`, and `rate_limits`
- **AND** `running` and `retrying` are JSON arrays even when empty

#### Scenario: Running entries preserve compatibility names
- **WHEN** a snapshot contains a running entry
- **THEN** the corresponding JSON object contains `issue_id`, `issue_identifier`, `state`, `session_id`, `turn_count`, `last_event`, `last_message`, `started_at`, `last_event_at`, and `tokens`
- **AND** `tokens` contains `input_tokens`, `output_tokens`, and `total_tokens`
- **AND** absent times are encoded as `null`

#### Scenario: Retrying entries preserve compatibility names
- **WHEN** a snapshot contains a retry entry
- **THEN** the corresponding JSON object contains `issue_id`, `issue_identifier`, `attempt`, `due_at`, and `error`
- **AND** absent due times are encoded as `null`

#### Scenario: Snapshot timeout keeps HTTP 200 envelope
- **WHEN** the configured snapshot function returns the snapshot-timeout sentinel error
- **THEN** the handler responds with HTTP `200`
- **AND** the body contains `generated_at`
- **AND** the body contains `{"error":{"code":"snapshot_timeout","message":"Snapshot timed out"}}`

#### Scenario: Snapshot unavailable keeps HTTP 200 envelope
- **WHEN** the configured snapshot function is unavailable or returns the snapshot-unavailable sentinel error
- **THEN** the handler responds with HTTP `200`
- **AND** the body contains `generated_at`
- **AND** the body contains `{"error":{"code":"snapshot_unavailable","message":"Snapshot unavailable"}}`

### Requirement: Issue detail payload preserves compatibility DTO fields
The system SHALL serve issue-specific runtime details from the current snapshot without querying the tracker.

#### Scenario: Running issue detail
- **WHEN** a requested issue identifier matches a running snapshot entry
- **THEN** the response contains `issue_identifier`, `issue_id`, `status`, `workspace`, `attempts`, `running`, `retry`, `logs`, `recent_events`, `last_error`, and `tracked`
- **AND** `status` is `running`
- **AND** `retry` is `null` when no retry entry exists
- **AND** `logs.codex_session_logs` is an empty array
- **AND** `tracked` is an empty object

#### Scenario: Retry issue detail
- **WHEN** a requested issue identifier matches a retry snapshot entry and no running entry
- **THEN** `status` is `retrying`
- **AND** `running` is `null`
- **AND** `retry` contains `attempt`, `due_at`, and `error`
- **AND** `last_error` matches the retry error

#### Scenario: Running entry wins over retry entry
- **WHEN** a requested issue identifier exists in both running and retry snapshot entries
- **THEN** `status` is `running`
- **AND** `issue_id` is taken from the running entry
- **AND** both `running` and `retry` payloads may be present

#### Scenario: Workspace path uses runtime path or fallback
- **WHEN** a matching running or retry entry has a workspace path
- **THEN** `workspace.path` uses that path
- **WHEN** the matching entries do not have a workspace path
- **THEN** `workspace.path` is the configured workspace root joined with the issue identifier

#### Scenario: Recent events use last-event inference
- **WHEN** a matching running entry has `LastEventAt`
- **THEN** `recent_events` contains one object with `at`, `event`, and `message`
- **WHEN** no running last-event timestamp is present
- **THEN** `recent_events` is an empty array

#### Scenario: Unknown issue returns compatibility error
- **WHEN** a requested issue identifier is absent from running and retry snapshot entries
- **THEN** the handler responds with HTTP `404`
- **AND** the body is `{"error":{"code":"issue_not_found","message":"Issue not found"}}`

### Requirement: Refresh payload preserves compatibility semantics
The system SHALL expose a best-effort refresh trigger that returns the Symphony-compatible refresh acknowledgement.

#### Scenario: Refresh accepted
- **WHEN** the configured refresh function returns a successful result
- **THEN** the handler responds with HTTP `202`
- **AND** the body contains `queued`, `coalesced`, `requested_at`, and `operations`
- **AND** `operations` is `["poll","reconcile"]`

#### Scenario: Refresh unavailable
- **WHEN** the configured refresh function is unavailable or returns the refresh-unavailable sentinel error
- **THEN** the handler responds with HTTP `503`
- **AND** the body is `{"error":{"code":"orchestrator_unavailable","message":"Orchestrator is unavailable"}}`

### Requirement: HTTP API remains a projection-only compatibility shell
The HTTP API implementation SHALL keep runtime state ownership outside `internal/httpapi`.

#### Scenario: Handler does not own runtime state
- **WHEN** `internal/httpapi` builds state, issue, or refresh responses
- **THEN** it uses only configured snapshot and refresh function seams
- **AND** it does not import orchestrator-private state, tracker adapters, CLI lifecycle code, or provider-specific packages

#### Scenario: Compatibility JSON may use issue naming
- **WHEN** the handler serializes JSON DTOs
- **THEN** it may use compatibility field names such as `issue_id` and `issue_identifier`
- **AND** it does not rename core Go domain types away from provider-neutral item terminology
