# T05 Final Implementation v1

## Goal

Define the provider-neutral runtime domain contract in `internal/domain` so later core packages can stop depending on ad hoc maps, Elixir-shaped names, and provider-specific runtime leakage. `T05` is the point where the Go port freezes the first stable runtime vocabulary for orchestration, retry bookkeeping, polling state, worker event reporting, and snapshot projection.

## Non-Goals

- No orchestrator loop implementation, claim/reconcile logic, or retry scheduling algorithms. That belongs to `T06`.
- No tracker read/write interfaces or Linear GraphQL behavior. That belongs to `internal/tracker`, `internal/trackers/linear`, and `internal/toolbridge`.
- No workflow parsing, config loading, or env/path normalization. That stays in `internal/config`.
- No HTTP API DTOs, terminal renderer DTOs, or web presenter structs. Those later packages should project from the domain snapshot instead of turning `internal/domain` into a compatibility DTO dump.
- No universal tracker write API, workpad abstraction, or provider-agnostic default workflow.
- No Lark-specific runtime behavior.

## Required Compatibility Behavior

- Preserve the approved terminology mapping: compatibility-facing surfaces may still say `issue`, but the core domain type is `WorkItem`.
- Keep `WorkItem` rich enough for orchestration and prompt rendering, because the current Symphony prompt builder renders the normalized issue struct directly and Codex turn titles depend on `identifier` plus `title`.
- Preserve blocker gating semantics for `Todo` work by carrying lightweight blocker identity and state in the domain model.
- Preserve retry semantics as first-class runtime data instead of implicit timer bookkeeping only. The runtime needs a visible retry queue with item identity, attempt count, schedule time, and last error context.
- Preserve polling state as explicit runtime data so the orchestrator can expose “checking now” versus “next poll in” without giving observability its own state machine.
- Preserve snapshot semantics as the single projection source for later API/dashboard/web surfaces.
- Keep workers on an event-reporting contract only. They may emit `RunEvent`s, but they do not own or mutate shared scheduling state.

## Proposed Domain Types

`internal/domain` should export a small set of plain Go types that later packages can compose without widening the core boundary:

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

This is intentionally one layer lower than HTTP/dashboard DTOs and one layer higher than tracker/provider payloads.

## `WorkItem` Contract

`WorkItem` is the provider-neutral replacement for the current normalized Linear issue. In V1 it should keep exactly the fields the runtime already needs for orchestration, prompt rendering, and Codex session labeling:

- `ID string`
- `Identifier string`
- `Title string`
- `Description string`
- `State string`
- `Priority *int`
- `BranchName string`
- `URL string`
- `AssigneeID string`
- `Labels []string`
- `BlockedBy []Blocker`
- `Routable *bool`
- `CreatedAt *time.Time`
- `UpdatedAt *time.Time`

Field intent:

- `Identifier` and `Title` support Codex turn titles and human-readable runtime output.
- `Description`, `Labels`, `CreatedAt`, and `UpdatedAt` stay in scope because the current prompt-building path can render them directly.
- `Priority`, `CreatedAt`, and `Identifier` support dispatch ordering.
- `BlockedBy` supports `Todo` dispatch gating.
- `Routable` is the provider-neutral replacement for Linear’s current `assigned_to_worker` dispatch gate. Adapters compute it; the core only consumes it.
- `nil` means “not explicitly excluded from routing”. This preserves the current Symphony bias toward dispatch eligibility unless the adapter can prove the item should not run on this worker.

`WorkItem` must not include:

- provider-specific GraphQL node fragments
- relation types like `blocks`
- project slug or endpoint config
- tracker write/workpad/comment state
- raw provider metadata bags used as escape hatches

## `Blocker` Contract

`Blocker` stays intentionally small:

- `ID string`
- `Identifier string`
- `State string`

This preserves the runtime meaning of “blocked by a non-terminal item” without dragging provider-specific relation payloads into the core model.

## Runtime Projection Types

The domain package should separate the schedulable item from the runtime view of an in-flight run.

### `ActiveRun`

`ActiveRun` is the snapshot-facing view of one currently running item. It should contain:

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

What stays out of `ActiveRun`:

- process IDs
- monitor refs
- timer refs
- “last reported token” bookkeeping
- any mutable orchestrator-only wiring needed to manage goroutines

Those are orchestrator internals, not durable runtime-domain facts.

### `RetryEntry`

`RetryEntry` is the visible retry queue entry:

- `ItemID string`
- `ItemIdentifier string`
- `Attempt int`
- `DueAt time.Time`
- `LastError string`
- `WorkerHost string`
- `WorkspacePath string`

Use absolute `DueAt` in the core model. Later compatibility shells can derive `due_in_ms` from it.

### `PollingState`

`PollingState` should be Go-native and explicit:

- `Checking bool`
- `NextPollAt *time.Time`
- `Interval time.Duration`

Use absolute time plus `time.Duration` rather than storing only integers in milliseconds. Later presenters can derive countdown values from the snapshot.

### `Snapshot`

`Snapshot` is the single projection source for later observability surfaces:

- `Running []ActiveRun`
- `Retrying []RetryEntry`
- `Polling PollingState`
- `CodexTotals CodexTotals`
- `RateLimits *RateLimits`

This preserves the current Symphony snapshot responsibilities without copying its raw map shape directly into the Go core.

## Codex Usage And Rate Limits

The runtime already exposes aggregate token totals and rate-limit data, so `T05` should freeze a small typed shape instead of leaving them as unstructured maps.

### `CodexTotals`

- `InputTokens int`
- `OutputTokens int`
- `TotalTokens int`
- `SecondsRunning int`

### `RateLimits`

- `LimitID string`
- `Primary *RateLimitBucket`
- `Secondary *RateLimitBucket`
- `Credits *RateLimitCredits`

### `RateLimitBucket`

- `Remaining *int`
- `Limit *int`
- `ResetInSeconds *int`

### `RateLimitCredits`

- `HasCredits *bool`
- `Unlimited *bool`
- `Balance *float64`

This matches the current observability evidence closely enough for parity while keeping the model codex-specific rather than tracker-specific.

## Worker Event Contract

`RunEvent` is the only worker-to-orchestrator mutable-state input. It should stay deliberately small and tagged:

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

`RunEventKind` should be a closed string-backed enum with the V1 events the design already names:

- `workspace_created`
- `workspace_path_discovered`
- `runner_host_selected`
- `codex_event_received`
- `turn_completed`
- `run_completed`
- `run_failed`
- `retry_scheduled`

The event contract should not attempt to encode every future workflow action. If a later task needs new worker lifecycle events, it can extend this enum deliberately.

## Boundaries And Anti-Overdesign Rules

- Do not add a universal “runtime state” struct mirroring the orchestrator’s future private maps. `T06` should own mutable scheduling state privately and project into `Snapshot`.
- Do not move provider config, project routing rules, or tracker-specific write semantics into `WorkItem`.
- Do not introduce generic metadata maps to compensate for unclear design. If a field is not clearly required by orchestration, prompt rendering, or snapshot parity, leave it out.
- Do not let `internal/domain` become the compatibility-shell DTO package. API/web/dashboard packages should project from `Snapshot`, not force `internal/domain` to mirror user-facing payloads byte-for-byte.
- Keep time values Go-native in the core (`time.Time`, `time.Duration`) and derive milliseconds or ISO strings at presentation boundaries.

## Relationship To Later Tasks

- `T06` should consume `WorkItem`, `RetryEntry`, `PollingState`, `RunEvent`, and `Snapshot`, while keeping process refs, claim sets, and scheduling maps private to the orchestrator.
- `T10` and `T11` should normalize tracker reads into `WorkItem` plus `Blocker` rather than reintroducing provider-shaped issue structs in core packages.
- `T09` should map Codex app-server notifications into `RunEvent` and populate `CodexTotals` plus `RateLimits`.
- `T15`, `T16`, and `T17` should project from `Snapshot` instead of reading orchestrator-private state directly.

## Test Focus

`go test ./internal/domain/...` should act as a contract-locking test suite for the exported domain vocabulary, not as a placeholder smoke test.

It should prove:

- `WorkItem` contains the approved provider-neutral fields needed for orchestration and prompt rendering, with no `Issue`, `Linear`, `tracker`, or GraphQL-shaped field leakage.
- `Blocker` stays a minimal identity-plus-state type rather than a provider payload dump.
- `Snapshot` contains running, retrying, polling, aggregate Codex totals, and rate-limit data, which are the runtime facts later observability layers need.
- `RetryEntry` uses absolute schedule time and preserves attempt/error/workspace context.
- `PollingState` uses explicit checking state plus Go-native timing types.
- `RunEventKind` freezes the worker event vocabulary required by the approved design.
- `RunEvent` can represent workspace, runner, Codex, completion, failure, and retry notifications without giving workers direct access to orchestrator state.

The tests may use reflection-based API shape assertions where appropriate, because the main point of `T05` is to freeze a domain contract that later tasks will compile against.

## Deferred Items

- Any scheduler algorithms, claim-set transitions, or retry backoff logic move to `T06`.
- Any tracker adapter logic for computing `Routable`, blockers, or item normalization moves to `T10` and `T11`.
- Any prompt/template rendering logic moves to later workflow tasks even though `WorkItem` keeps the fields those tasks will need.
- Any API/dashboard/web JSON compatibility concerns move to their dedicated compatibility-shell tasks.
