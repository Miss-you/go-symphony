# T05 Final Implementation

## Review Gate

`final_impl_v1.md` passed the required rubric review.

Review results:

- `review_1.md`: 82 / 100, no high-severity issues
- `review_2.md`: 89 / 100, no high-severity issues
- average: 85.5 / 100

Key review corrections accepted into this final plan:

- keep the exported helper-type set limited to the concrete snapshot and event shapes that the implementation and contract tests actually proved necessary
- make `WorkItem` field scope explicitly compatibility-driven rather than an excuse to grow a provider-shaped core model
- make `Routable` a narrow adapter-computed gate with explicit nil semantics, not a second routing policy
- require export-surface tests so later tasks cannot quietly widen `internal/domain`

Acceptance decision:

- average score exceeds the `>= 80` threshold
- no reviewer reported a remaining high-severity issue
- remaining notes are scope-discipline items and are incorporated below

## Final Scope

`T05` freezes the provider-neutral runtime vocabulary in `internal/domain` for later core packages.

The stable exported surface for this task is:

- `WorkItem`
- `Blocker`
- `ActiveRun`
- `RetryEntry`
- `PollingState`
- `Snapshot`
- `RunEvent`
- `RunEventKind`
- `CodexTotals`
- `RateLimits`
- `RateLimitBucket`
- `RateLimitCredits`

These extra snapshot/event helper types remain in scope because the landed implementation and contract tests proved that later packages need named concrete shapes for running-item projections and aggregate Codex/rate-limit data. `T05` still does not permit generic metadata bags or provider-specific payload structs.

`T05` does not implement:

- orchestrator scheduling logic, claim sets, retry backoff, or reconciliation algorithms
- tracker read/write interfaces or Linear GraphQL behavior
- workflow/config loading
- HTTP/API/web/dashboard compatibility DTOs
- a universal tracker write API, workpad abstraction, or provider-agnostic default workflow

## Final Domain Design

### `WorkItem`

`WorkItem` is the provider-neutral replacement for the current normalized Symphony issue model.

It keeps only fields that current Symphony already exposes to runtime behavior or prompt rendering:

- `ID string`
- `Identifier string`
- `Title string`
- `Description string`
- `State string`
- `Priority *int`
- `URL string`
- `BranchName string`
- `AssigneeID string`
- `Labels []string`
- `BlockedBy []Blocker`
- `Routable *bool`
- `CreatedAt *time.Time`
- `UpdatedAt *time.Time`

Why this exact set is allowed:

- `Identifier`, `Title`, and `Description` are already part of the default prompt and Codex turn labeling path
- `Labels`, `CreatedAt`, and `UpdatedAt` are already available to current prompt templates and have explicit test coverage in the reference implementation
- `Priority` and `CreatedAt` are part of dispatch ordering semantics
- `BlockedBy` and `Routable` participate in dispatch gating
- `URL`, `BranchName`, and `AssigneeID` stay only because the current normalized issue surface already exposes them to compatibility-facing prompt/template usage; they are not permission to add broader provider metadata

`Routable` means only this: the tracker adapter has already determined whether the item is eligible for this worker from a provider-specific routing perspective. It does not replace state checks, blocker checks, or concurrency checks in the orchestrator.

Its pointer semantics are intentional:

- `nil` means “not explicitly denied by the adapter”
- `false` means “do not dispatch on this worker”
- `true` means “explicitly routable”

`WorkItem` must not include:

- provider config such as project slug, endpoint, or tracker kind
- provider-specific write/workpad/comment state
- generic metadata bags
- raw GraphQL payload fragments or relation-type enums

### `Blocker`

`Blocker` stays intentionally minimal:

- `ID string`
- `Identifier string`
- `State string`

This is enough to preserve the existing “blocked todo item” semantics without leaking provider relation payloads into the core.

### `RetryEntry`

`RetryEntry` represents one visible retry queue item:

- `ItemID string`
- `ItemIdentifier string`
- `Attempt int`
- `DueAt time.Time`
- `LastError string`
- `WorkerHost string`
- `WorkspacePath string`

Use absolute time in the core model. Later compatibility presenters can derive countdown or `due_in_ms` values.

### `PollingState`

`PollingState` should be Go-native and explicit:

- `Checking bool`
- `NextPollAt *time.Time`
- `Interval time.Duration`

This preserves the current “checking now” versus “next poll in” semantics while avoiding milliseconds-only core bookkeeping.

### `Snapshot`

`Snapshot` is the single projection source for later observability surfaces.

It is composed from:

- `Running []ActiveRun`
- retry queue items as `[]RetryEntry`
- `Polling PollingState`
- aggregate `CodexTotals` for the whole runtime
- optional `RateLimits` when the Codex protocol produces them

`ActiveRun` is the concrete running-item projection needed by later observability layers:

- `ItemID string`
- `ItemIdentifier string`
- `State string`
- `WorkerHost string`
- `WorkspacePath string`
- `SessionID string`
- `TurnCount int`
- `StartedAt time.Time`
- `LastEventAt *time.Time`
- `LastEventKind RunEventKind`
- `LastEventMessage string`
- `CodexTotals CodexTotals`

### `RunEvent` And `RunEventKind`

`RunEvent` is the only worker-to-orchestrator reporting contract. Workers emit events; they do not mutate shared scheduling state directly.

`RunEvent` should carry:

- `Kind RunEventKind`
- `ItemID string`
- `ItemIdentifier string`
- `At time.Time`
- `Attempt int`
- `WorkerHost string`
- `WorkspacePath string`
- `SessionID string`
- `Message string`
- `Err error`
- `CodexTotals CodexTotals`
- `RateLimits *RateLimits`

Use a closed string-backed `RunEventKind` with the approved V1 events:

- `workspace_created`
- `workspace_path_discovered`
- `runner_host_selected`
- `codex_event_received`
- `turn_completed`
- `run_completed`
- `run_failed`
- `retry_scheduled`

Do not turn `RunEvent` into a generic dump of every worker-local detail.

## Boundary Rules

- The orchestrator remains the only owner of mutable runtime state; `internal/domain` defines values, not the owner state machine.
- `observability` remains projection-only and must build from `Snapshot` rather than inventing a second runtime truth source.
- Tracker adapters normalize provider payloads into `WorkItem` and `Blocker`; provider-specific fetch/write behavior stays outside the core.
- Compatibility-shell packages may continue to use issue-centric language in DTOs and UI, but the core must stay on `WorkItem` terminology.

## Test Focus

`go test ./internal/domain/...` must prove the domain contract, not just package existence.

It should explicitly prove:

- the exported `internal/domain` surface contains the approved task types and does not silently grow unrelated exported domain types
- `WorkItem` contains the approved provider-neutral field set and does not reintroduce `Issue`, `Linear`, `tracker`, or generic metadata leakage
- `Blocker` remains a minimal identity-plus-state value
- `ActiveRun`, `CodexTotals`, and the rate-limit structs stay limited to the runtime projection data later observability layers actually need
- `RetryEntry` preserves attempt, error, worker/workspace context, and absolute retry time
- `PollingState` uses explicit checking state and Go-native timing types
- `Snapshot` carries the runtime facts later API/dashboard/web layers need, without forcing `internal/domain` to mirror compatibility DTOs byte-for-byte
- `RunEventKind` freezes the worker event vocabulary named by the design
- `RunEvent` can represent workspace, runner, Codex, completion, failure, and retry notifications without exposing orchestrator-private control data

Use reflection-based contract tests where helpful to lock exported type names, field names, and enum values. The point of `T05` is to give `T06+` a stable compile-time domain boundary.

## Deferred To Later Tasks

- scheduler algorithms and private orchestrator state shape move to `T06`
- tracker normalization logic moves to `T10` and `T11`
- Codex protocol-to-event mapping moves to `T09`
- API/web/dashboard JSON compatibility moves to `T15` through `T17`
