## ADDED Requirements

### Requirement: Orchestrator owns mutable scheduling state privately
The core runtime MUST implement `internal/orchestrator` as the sole owner of mutable scheduling state for polling, claims, running items, retries, stall recovery, and aggregate runtime counters. Workers MUST report facts to the orchestrator through `domain.RunEvent` rather than mutating shared runtime state directly.

#### Scenario: Worker progress updates a running entry
- **WHEN** a worker reports lifecycle progress through a `domain.RunEvent`
- **THEN** the orchestrator updates only its own private running-state entry
- **AND** no shared mutable runtime map is exposed outside `internal/orchestrator`

#### Scenario: Worker retry notification does not replace retry ownership
- **WHEN** a worker emits `RunEventRetryScheduled`
- **THEN** the orchestrator may record that notification as run metadata
- **AND** the worker does not directly create, replace, or reorder the orchestrator's private retry queue

### Requirement: Orchestrator preserves poll cadence and refresh coalescing
The orchestrator MUST schedule an immediate initial poll, expose a checking-versus-next-poll projection, and coalesce repeated refresh requests while a poll is already active or already due.

#### Scenario: Startup schedules an immediate poll
- **WHEN** the orchestrator service starts successfully
- **THEN** it schedules a poll due immediately
- **AND** snapshots can observe `Polling.Checking=true` during the active poll transition

#### Scenario: Repeated refresh requests coalesce
- **WHEN** a refresh request arrives while the orchestrator is already checking or an immediate poll is already queued
- **THEN** the orchestrator keeps only one pending immediate refresh
- **AND** stale queued tick deliveries are ignored

### Requirement: Orchestrator orders and gates dispatch source-faithfully
The orchestrator MUST sort candidate items by priority, creation time, and stable identity, and it MUST dispatch only items that satisfy the approved active-state, blocker, routing, claim, running, and concurrency gates.

#### Scenario: Candidate ordering is deterministic
- **WHEN** multiple dispatchable candidates are visible in the same poll cycle
- **THEN** the orchestrator evaluates them in ascending priority order
- **AND** older `CreatedAt` values win ties before stable identifier/id ordering is applied

#### Scenario: Blocked or unroutable todo item does not dispatch
- **WHEN** a candidate item is in `Todo` and has any blocker that is not terminal, or `Routable` is explicitly `false`
- **THEN** the orchestrator does not dispatch that item in the current cycle

#### Scenario: Stale candidate is revalidated before dispatch
- **WHEN** a candidate item looked dispatchable in the list response but the revalidation refresh shows it missing, terminal, non-active, or unroutable
- **THEN** the orchestrator skips that stale dispatch

### Requirement: Orchestrator preserves continuation and failure retry lineage
The orchestrator MUST own retry scheduling and distinguish continuation retry from failure retry while keeping claim retention, attempt progression, and stale-delivery protection explicit.

#### Scenario: Normal completion seeds continuation retry
- **WHEN** a running item reports `run_completed`
- **THEN** the orchestrator removes it from `running`, keeps the claim, and schedules continuation retry attempt `1` with a short delay of about one second

#### Scenario: Failure retry advances lineage and caps backoff
- **WHEN** a running item reports `run_failed` or stall recovery triggers a retry-worthy restart
- **THEN** the orchestrator schedules failure retry attempt `1` for a first failure or `N+1` for a carried lineage attempt `N`
- **AND** the retry delay is `min(10s * 2^(attempt-1), maxRetryBackoff)`

#### Scenario: Capacity-blocked continuation falls into failure backoff
- **WHEN** continuation retry revalidation succeeds but redispatch is blocked by current capacity or concurrency limits
- **THEN** the orchestrator keeps the claim and reschedules the item as failure attempt `2`
- **AND** it does not repeat the short continuation delay indefinitely

#### Scenario: Stale retry delivery is ignored
- **WHEN** an older scheduled retry callback arrives after a newer retry entry replaced it
- **THEN** the orchestrator ignores the stale callback and keeps the newer retry entry intact

### Requirement: Orchestrator reconciles running items and stalled runs before new dispatch
Each poll cycle MUST reconcile existing running items before dispatching new ones, and stall recovery MUST convert stale runs into failure-style retries without relying on observability code.

#### Scenario: Reconcile updates an active routed run in place
- **WHEN** refresh shows a currently running item still active and still routable
- **THEN** the orchestrator keeps the run and claim
- **AND** it replaces the stored item snapshot with the refreshed item

#### Scenario: Reconcile drops terminal, unroutable, non-active, or missing runs
- **WHEN** refresh shows a running item as terminal, explicitly unroutable, non-active, or missing entirely
- **THEN** the orchestrator stops the run, drops the run and claim state, and clears any retry entry
- **AND** terminal resolution remains eligible for cleanup intent while non-terminal invalidation does not

#### Scenario: Stall recovery schedules failure-style retry
- **WHEN** a running item has no worker activity for longer than the configured stall timeout
- **THEN** the orchestrator stops the run and schedules a failure-style retry with a stall-specific error

### Requirement: Orchestrator projects one deterministic runtime snapshot
The orchestrator MUST project private runtime state into `domain.Snapshot` only, with deterministic ordering and explicit semantics for aggregate totals and rate limits.

#### Scenario: Snapshot exposes running, retrying, and polling state
- **WHEN** a snapshot is requested
- **THEN** it returns `Running`, `Retrying`, and `Polling` values derived only from orchestrator-owned private state
- **AND** it does not expose timers, goroutine refs, or other private wiring

#### Scenario: Aggregate totals and rate limits have stable semantics
- **WHEN** multiple worker events report cumulative per-run token totals and optional rate-limit updates
- **THEN** the orchestrator snapshot reports lifetime cumulative aggregate `CodexTotals` without double-counting repeated cumulative updates
- **AND** it reports the latest non-nil observed `RateLimits` view
